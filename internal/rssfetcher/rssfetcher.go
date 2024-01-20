package rssfetcher

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/awused/aw-rss/internal/config"
	"github.com/awused/aw-rss/internal/database"
	"github.com/awused/aw-rss/internal/quirks"
	"github.com/awused/aw-rss/internal/structs"
	"github.com/mmcdole/gofeed"
	gofeedRss "github.com/mmcdole/gofeed/rss"
	log "github.com/sirupsen/logrus"
)

// ErrClosed is returned when attempting to start a closed RssFetcher
var ErrClosed = errors.New("RssFetcher already closed")

const dbPollPeriod = time.Duration(time.Minute * 5)

// TODO -- make this configurable, along with a maxPollPeriod
const minPollPeriod = time.Duration(time.Minute * 15)
const minCloudflarePeriod = time.Duration(time.Minute * 90)
const RssTimeout = 30 * time.Second
const startupRateLimit = 250 * time.Millisecond

// RssFetcher is responsible for reading fetching feeds and storing them in the
// database
type RssFetcher interface {
	// Run begins fetching RSS feeds and only stops when closed or when
	// encountering an unrecoverable error.
	Run() error
	// Close stops a running RssFetcher and cleans up.
	Close() error
	// InformFeedChanged informs the fetcher that a feed has changed
	InformFeedChanged()
	// RerunFeed immediately fetches a feed if it is waiting between fetches or after an error
	RerunFeed(int64)
	// RerunFailing immediately reruns all failing feeds without waiting for backoff
	RerunFailing()
}

type feedError struct {
	f   *structs.Feed
	err error
}

type routineChannels struct {
	kill  chan struct{}
	rerun chan struct{}
}

type rssFetcher struct {
	conf       config.Config
	db         *database.Database
	httpClient *http.Client
	cloudflare *cloudflare
	// All guarded by lock
	feeds        map[int64]*structs.Feed
	routines     map[int64]routineChannels
	retryBackoff map[int64]time.Duration
	rerunFailing chan struct{}
	// End guarded by lock
	lock       sync.RWMutex
	lastPolled time.Time
	wg         sync.WaitGroup
	errorChan  chan feedError
	// Used when a feed has changed in a way that will impact fetching
	feedsChangedChan chan struct{}
	// Per-host critical sections
	hostLocks map[string]*sync.Mutex
	closed    bool
	closeChan chan struct{}
	closeLock sync.Mutex
}

// NewRssFetcher returns a new RssFetcher
func NewRssFetcher(conf config.Config,
	db *database.Database) (RssFetcher, error) {

	var rss rssFetcher
	rss.conf = conf
	rss.db = db
	rss.httpClient = &http.Client{
		Timeout: RssTimeout,
	}
	rss.feeds = make(map[int64]*structs.Feed)
	rss.routines = make(map[int64]routineChannels)
	rss.retryBackoff = make(map[int64]time.Duration)
	rss.rerunFailing = make(chan struct{})
	rss.hostLocks = make(map[string]*sync.Mutex)
	rss.closeChan = make(chan struct{})
	rss.errorChan = make(chan feedError)
	rss.feedsChangedChan = make(chan struct{})

	rss.cloudflare = newCloudflare(rss.conf, rss.closeChan)

	return &rss, nil
}

// InformFeedChanged informs the fetcher that a feed has changed in a way that
// impacts fetching.
func (r *rssFetcher) InformFeedChanged() {
	select {
	case r.feedsChangedChan <- struct{}{}:
	case <-r.closeChan:
	}
}

func (r *rssFetcher) RerunFeed(fid int64) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	rchans, ok := r.routines[fid]
	if !ok {
		log.Info("Can't rerun feed ", fid, " since it is not currently running")
		return
	}

	select {
	case rchans.rerun <- struct{}{}:
		log.Info("Restarted feed from user request: ", fid)
	default:
	}
}

func (r *rssFetcher) RerunFailing() {
	r.lock.Lock()
	defer r.lock.Unlock()

	old := r.rerunFailing
	r.rerunFailing = make(chan struct{})

	close(old)
	log.Info("Rerunning all failing feeds")
}

func (r *rssFetcher) Close() error {
	log.Info("Closing rssFetcher")

	if r.closed {
		log.Warning("Tried to close rssFetcher that has already been closed")
		return nil
	}
	r.closeLock.Lock()
	defer r.closeLock.Unlock()
	if r.closed {
		log.Warning("Tried to close rssFetcher that has already been closed")
		return nil
	}
	// Kill the main routine, though Run() will not return until after
	// Close() releases the lock.
	close(r.closeChan)
	r.closed = true

	r.lock.Lock()
	r.killOldRoutines(map[int64]*structs.Feed{})
	r.feeds = map[int64]*structs.Feed{}
	r.lock.Unlock()

	log.Infof("Waiting up to 60 seconds for goroutines to finish")

	var c = make(chan struct{})
	go func() {
		r.wg.Wait()
		close(c)
	}()

	select {
	case <-time.After(time.Second * 60):
		log.Errorf("Some goroutines failed to exit within 60 seconds")
	case <-c:
		log.Info("All goroutines exited successfully")
	}

	defer log.Info("Close() completed")
	return r.db.Close()
}

func (r *rssFetcher) Run() (err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = rec.(error)
		}
	}()

	r.closeLock.Lock()
	if r.closed {
		r.closeLock.Unlock()
		return ErrClosed
	}
	r.closeLock.Unlock()

	forcePoll := false

	log.Info("rssFetcher started running")
	for {
		if forcePoll ||
			r.lastPolled.IsZero() || time.Since(r.lastPolled) > dbPollPeriod {
			log.Debug("Checking database for new feeds")
			forcePoll = false

			newFeedsArray, err := r.db.GetCurrentFeeds()
			if err == database.ErrClosed {
				r.closeLock.Lock()
				defer r.closeLock.Unlock()

				if r.closed {
					// The database was closed in the brief window between the last time
					// closeChan was checked and when the DB was polled. No error.
					return nil
				}
				return fmt.Errorf("Database unexpectedly closed")
			} else if err != nil {
				// Close unconditionally on unexpected DB error.
				_ = r.Close()
				return err
			}

			log.Tracef("Got feeds: %s", newFeedsArray)

			var newFeeds = make(map[int64]*structs.Feed)
			for _, e := range newFeedsArray {
				newFeeds[e.ID()] = e
			}

			// Critical section for communicating on channels and spawning new routines.
			r.closeLock.Lock()

			if r.closed {
				r.closeLock.Unlock()
				log.Info("rssFetcher closed, exiting")
				return nil
			}

			r.lock.Lock()
			r.killOldRoutines(newFeeds)
			r.startNewRoutines(newFeedsArray)
			r.feeds = newFeeds
			r.lock.Unlock()

			r.closeLock.Unlock()

			r.lastPolled = time.Now()
		}

		select {
		case fe := <-r.errorChan:
			r.restartFailedRoutine(fe)
			// TODO -- handle an update from fe.f
		case <-r.closeChan:
			log.Info("rssFetcher closed, exiting")
			// Acquire the lock so Run() does not return before Close() completes
			r.closeLock.Lock()
			r.closeLock.Unlock()
			return nil
		case <-time.After(dbPollPeriod - time.Since(r.lastPolled)):
			// This polling is the last line of defense against out of band edits
		case <-r.feedsChangedChan:
			forcePoll = true
			// This is a rare enough case that it's simplest to just poll the DB again
		}
	}
}

// Main work done here for each feed
// TODO -- clean this up and refactor it
func (r *rssFetcher) routine(f *structs.Feed, rchans routineChannels) {
	defer func() {
		if rec := recover(); rec != nil {
			err := rec.(error)
			newF, nerr := r.db.MutateFeed(
				f.ID(),
				structs.FeedSetFetchFailed(time.Now().UTC()))
			if nerr == nil {
				f = newF
			}
			select {
			case r.errorChan <- feedError{f, err}:
			case <-r.closeChan:
			}
		}
		log.Tracef("Routine for [%s] completed", f)
		r.wg.Done()
		// We could attempt to send f on feedUpdateChan but
		// any important updates should come through the webserver
	}()

	parser := gofeed.NewParser()

	log.Debugf("Routine for [%s] started", f)
	for {
		// The feed may have been updated
		r.lock.RLock()
		newF, ok := r.feeds[f.ID()]
		r.lock.RUnlock()
		if !ok {
			select {
			case <-rchans.kill:
				log.Debugf("Routine for [%s] killed by parent", f)
			default:
				// Should never happen
				log.Warningf("Feed [%s] unexpectedly missing", f)
			}
			return
		}
		f = newF

		body := ""
		if strings.HasPrefix(f.URL(), "!") {
			body = r.runExternalCommandFeed(f, rchans.kill)
		} else {
			body = r.fetchHTTPFeed(f, rchans.kill)
		}
		body = quirks.HandleBodyQuirks(f, body)

		select {
		case <-rchans.kill:
			log.Debugf("Routine for [%s] killed by parent", f)
			return
		default:
		}

		feed, err := parser.ParseString(body)
		if err != nil {
			log.Errorf("Error calling parser.ParseString for [%s]: %v", f, err)
			log.Warning("Body was: " + body)
			panic(err)
		}

		newF, err = r.db.MutateFeed(
			f.ID(), structs.FeedMergeGofeed(feed))
		if err != nil {
			log.Errorf("Error updating feed [%s]: %v", f, err)
			panic(err)
		}
		f = newF

		err = r.db.InsertItems(structs.CreateNewItems(f, feed.Items))
		if err != nil {
			log.Errorf("Error inserting items for feed [%s]: %v", f, err)
			panic(err)
		}

		newF, err = r.db.MutateFeed(
			f.ID(), structs.FeedSetFetchSuccess)
		if err != nil {
			log.Errorf("Error updating feed [%s]: %v", f, err)
			panic(err)
		}
		f = newF

		/*
			TODO -- Do this
			select {
			case r.feedUpdateChan <- f:
			case <-kill:
				log.Debugf("Routine for [%s] killed by parent", f)
				return
			}*/

		r.lock.Lock()
		if _, ok := r.retryBackoff[f.ID()]; ok {
			r.retryBackoff[f.ID()] = time.Minute
		}
		r.lock.Unlock()

		select {
		case <-rchans.kill:
			log.Debugf("Routine for [%s] killed by parent", f)
			return
		case <-rchans.rerun:
		case <-time.After(r.getSleepTime(f, feed, body)):
		}
	}
}

func (r *rssFetcher) runExternalCommandFeed(
	f *structs.Feed, kill <-chan struct{}) string {
	// "Host" is the executable
	// This is not correct when there are spaces in the path, but it fails
	// in a safe manner.
	split := strings.SplitN(f.URL(), " ", 3)
	h := split[0]
	if h == "!rss-scrapers" && len(split) > 2 {
		// As a special case for https://github.com/awused/rss-scrapers
		h = h + " " + split[1]
	}

	r.lock.Lock()
	lock, ok := r.hostLocks[h]
	if !ok {
		lock = &sync.Mutex{}
		r.hostLocks[h] = lock
	}
	r.lock.Unlock()

	lock.Lock()
	defer lock.Unlock()

	// Check if we've been killed while acquiring the lock
	select {
	case <-kill:
		return ""
	default:
	}

	output, err := exec.Command("sh", "-c", f.URL()[1:]).CombinedOutput()

	if err != nil {
		log.Errorf("Error running external command for [%s]: %v", f, err)
		log.Error("Output was: \n" + string(output))
		panic(err)
	}

	return string(output)
}

func (r *rssFetcher) fetchHTTPFeed(
	f *structs.Feed,
	kill <-chan struct{}) string {
	h, _, err := host(f.URL())
	if err != nil {
		log.Errorf("Could not parse host for [%s]: %v", f, err)
		panic(err)
	}

	r.lock.Lock()
	lock, ok := r.hostLocks[h]
	if !ok {
		lock = &sync.Mutex{}
		r.hostLocks[h] = lock
	}
	r.lock.Unlock()

	lock.Lock()
	defer lock.Unlock()

	// Check if we've been killed while acquiring the lock.
	select {
	case <-kill:
		return ""
	default:
	}

	dur, shouldDelay := quirks.MaybeGetHostDelay(h)
	if shouldDelay {
		select {
		case <-kill:
			return ""
		case <-time.After(dur):
		}
	}

	// c, ua, err := r.cloudflare.getCookie(f.URL())
	// if err != nil {
	// 	log.Errorf("Error calling cloudflare.getCookie() for [%s]: %v", f, err)
	// 	panic(err)
	// }
	cf := r.cloudflare.getIsCloudflare(f.URL())
	c, ua := "", ""
	body := ""

	if !cf {
		body = r.fetchHTTPBody(f, kill, c, ua)

		cf, err = r.cloudflare.isCloudflareResponse(f.URL(), body)
		if err != nil {
			log.Errorf("Error calling isCloudflareResponse() for [%s]: %v", f, err)
			log.Errorf("Body was: \n" + body)
			panic(err)
		}
	}

	if cf {
		// Check if we've been killed before making HTTP calls
		select {
		case <-kill:
			return ""
		default:
		}
		// c, ua, err := r.cloudflare.getNewCookie(f.URL())
		// select {
		// case <-kill:
		// 	return ""
		// default:
		// }
		// if err != nil {
		// 	log.Errorf("Error calling cloudflare.GetNewCookie for [%s]: %v", f, err)
		// 	log.Error("Body was: \n" + body)
		// 	panic(err)
		// }
		//
		//body = r.fetchHTTPBody(f, kill, c, ua)

		body, err = r.cloudflare.getFeedContents(f.URL())
		if err != nil {
			log.Errorf("Error calling cloudflare.getFeedContents for [%s]: %v", f, err)
			log.Error("Body was: \n" + body)
			panic(err)
		}
	}

	return body
}

func (r *rssFetcher) fetchHTTPBody(
	f *structs.Feed,
	kill <-chan struct{},
	cookie string,
	userAgent string) string {
	req, err := http.NewRequest("GET", f.URL(), nil)
	if err != nil {
		log.Panic(err)
	}

	// Workaround for dolphinemu.org, but doesn't seem to break any other feeds.
	req.Header.Add("Cache-Control", "no-cache")

	if cookie != "" {
		req.Header.Add("Cookie", cookie)
	}
	if userAgent != "" {
		req.Header.Add("User-Agent", userAgent)
	} else {
		// Pretend to be wget. Some sites don't like an empty user agent.
		// Reddit in particular will _always_ say to retry in a few seconds,
		// even if you wait hours.
		req.Header.Add("User-Agent", "Wget/1.19.5 (freebsd11.1)")
	}

	resp, err := r.httpClient.Do(req)
	// Check immediately after the HTTP request
	// If this has been killed do not write updates to the DB
	select {
	case <-kill:
		return ""
	default:
	}

	if err != nil {
		log.Errorf("Error calling httpClient.Get for [%s]: %v", f, err)
		panic(err)
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	// Close unconditionally to avoid memory leaks
	_ = resp.Body.Close()
	if err != nil {
		log.Errorf("Error reading response body for [%s]: %v", f, err)
		panic(err)
	}

	return string(bodyBytes)
}

func (r *rssFetcher) getSleepTime(f *structs.Feed, feed *gofeed.Feed, body string) time.Duration {
	sleepTime := minPollPeriod
	if feed.FeedType == "rss" {
		rssFeed, err := (&gofeedRss.Parser{}).Parse(strings.NewReader(body))
		if err != nil {
			log.Warningf("RSS feed could not be parsed as RSS [%s]", f)
		} else if rssFeed.TTL != "" {
			ttl, err := strconv.Atoi(rssFeed.TTL)
			if err != nil {
				log.Warningf("RSS feed [%s] had invalid TTL %s", f, rssFeed.TTL)
			} else {
				sleepTime = time.Duration(ttl) * time.Minute
			}
		}
	}

	if r.cloudflare.getIsCloudflare(f.URL()) && sleepTime < minCloudflarePeriod {
		sleepTime = minCloudflarePeriod
	}

	if sleepTime < minPollPeriod {
		log.Debugf("Poll period for feed [%s] was %s; using minPollPeriod", f, sleepTime)
		sleepTime = minPollPeriod
	}

	log.Tracef("Waiting %d seconds until next update for [%s]", sleepTime/time.Second, f)
	return sleepTime
}

func (r *rssFetcher) killRoutine(f *structs.Feed) {
	rchans, ok := r.routines[f.ID()]
	if !ok {
		log.Warningf("Tried to kill non-existent routine for [%s]", f)
		return
	}
	log.Debugf("Killing routine for [%s]", f)
	close(rchans.kill)
	// We do not close rchans.rerun to avoid races
	delete(r.routines, f.ID())
	delete(r.retryBackoff, f.ID())
}

func (r *rssFetcher) restartRoutine(
	f *structs.Feed, rchans routineChannels, delay time.Duration, rerunFailing <-chan struct{}) {
	log.Debugf(
		"Restarting routine for [%s] in %s", f, delay)

	select {
	case <-rchans.kill:
		log.Debugf(
			"Routine for [%s] killed by parent before it could restart", f)
		r.wg.Done()
		return
	case <-rchans.rerun:
	case <-rerunFailing:
	case <-time.After(delay):
	}

	log.Debugf("Restarting routine for [%s] now", f)
	r.routine(f, rchans)
}

func (r *rssFetcher) killOldRoutines(newFeeds map[int64]*structs.Feed) {
	for i, f := range r.feeds {
		if _, ok := newFeeds[i]; !ok {
			r.killRoutine(f)
		}
	}
}

func (r *rssFetcher) startNewRoutines(newFeeds []*structs.Feed) {
	for _, feed := range newFeeds {
		if _, ok := r.feeds[feed.ID()]; !ok {
			log.Debugf("Starting new routine for [%s]", feed)

			r.routines[feed.ID()] = routineChannels{
				kill:  make(chan struct{}),
				rerun: make(chan struct{}),
			}
			r.retryBackoff[feed.ID()] = time.Minute

			r.wg.Add(1)
			go r.routine(feed, r.routines[feed.ID()])
		}
	}
}

func (r *rssFetcher) restartFailedRoutine(fe feedError) {
	feed := fe.f
	id := feed.ID()

	r.lock.Lock()
	defer r.lock.Unlock()

	_, ok := r.feeds[id]
	if !ok {
		log.Warningf("Tried to restart routine for non-existent feed [%s]", feed)
		return
	}
	backoff := r.retryBackoff[id]
	if isCloudflareError(fe.err) {
		backoff = time.Hour * 6
	}
	// This should be a no-op since the routine will already have exited,
	// but will clear old values out of the maps.
	r.killRoutine(feed)
	r.routines[id] = routineChannels{
		kill:  make(chan struct{}),
		rerun: make(chan struct{}),
	}
	r.retryBackoff[id] = backoff * 2
	if r.retryBackoff[id] > time.Hour*6 {
		r.retryBackoff[id] = time.Hour * 6 // Check at least once every six hours
	}

	log.Warningf(
		"Error in routine for [Feed: %d], attempting to restart in %s",
		id, backoff)
	r.wg.Add(1)
	rerunFailing := r.rerunFailing
	go r.restartRoutine(feed, r.routines[id], backoff, rerunFailing)
}

func host(feedURL string) (string, string, error) {
	u, err := url.Parse(feedURL)
	if err != nil {
		return "", "", err
	}

	return u.Host, u.Scheme, nil
}
