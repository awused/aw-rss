package rssfetcher

import (
	"io/ioutil"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/awused/aw-rss/internal/database"
	"github.com/awused/aw-rss/internal/quirks"
	"github.com/awused/aw-rss/internal/structs"
	"github.com/golang/glog"
	"github.com/mmcdole/gofeed"
	gofeedRss "github.com/mmcdole/gofeed/rss"
)

const dbPollPeriod = time.Duration(time.Minute * 5)
const minPollPeriod = time.Duration(time.Minute * 10)
const rssTimeout = 30 * time.Second
const startupRateLimit = 250 * time.Millisecond

// RssFetcher is responsible for reading fetching feeds and storing them in the
// database
type RssFetcher interface {
	Run() error
	Close() error
}

type feedError struct {
	f   *structs.Feed
	err error
}

type rssFetcher struct {
	db            *database.Database
	httpClient    *http.Client
	cloudflare    *cloudflare
	feeds         map[int64]*structs.Feed
	routines      map[int64]chan struct{}
	retryBackoff  map[int64]time.Duration
	mapLock       sync.RWMutex
	lastPolled    time.Time
	wg            sync.WaitGroup
	errorChan     chan feedError
	rateLimitChan chan struct{}
	closed        bool
	closeChan     chan struct{}
	closeLock     sync.Mutex
}

// NewRssFetcher returns a new RssFetcher
func NewRssFetcher() (RssFetcher, error) {
	glog.V(5).Info("rssFetcher() started")

	db, err := database.GetDatabase()
	if err != nil {
		glog.Error(err)
		return nil, err
	}

	var rss rssFetcher
	rss.db = db
	rss.httpClient = &http.Client{
		Timeout: rssTimeout,
	}
	rss.feeds = make(map[int64]*structs.Feed)
	rss.routines = make(map[int64]chan struct{})
	rss.retryBackoff = make(map[int64]time.Duration)
	rss.rateLimitChan = make(chan struct{})
	rss.closeChan = make(chan struct{})
	rss.errorChan = make(chan feedError)

	rss.cloudflare = newCloudflare(rss.closeChan)

	glog.V(5).Info("rssFetcher() completed")
	return &rss, nil
}

func (r *rssFetcher) Close() error {
	glog.Info("Closing rssFetcher")

	if r.closed {
		glog.Warning("Tried to close rssFetcher that has already been closed")
		return nil
	}
	r.closeLock.Lock()
	defer r.closeLock.Unlock()
	if r.closed {
		glog.Warning("Tried to close rssFetcher that has already been closed")
		return nil
	}
	// Close and kill the main routine last
	close(r.closeChan)
	r.closed = true

	r.mapLock.Lock()
	r.killOldRoutines(map[int64]*structs.Feed{})
	r.feeds = map[int64]*structs.Feed{}
	r.mapLock.Unlock()

	glog.Infof("Waiting up to 60 seconds for goroutines to finish")

	var c = make(chan struct{})
	go func() {
		r.wg.Wait()
		close(c)
	}()

	select {
	case <-time.After(time.Second * 60):
		glog.Errorf("Some goroutines failed to exit within 60 seconds")
	case <-c:
		glog.Info("All goroutines exited successfully")
	}

	defer glog.Info("Close() completed")
	return r.db.Close()
}

func (r *rssFetcher) Run() (err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = rec.(error)
		}
	}()

	r.wg.Add(1)
	go r.rateLimitRoutines()

	glog.Info("rssFetcher started running")
	for {
		if r.lastPolled.IsZero() || time.Since(r.lastPolled) > dbPollPeriod {
			glog.V(3).Info("Checking database for new feeds")

			newFeedsArray, err := r.db.GetCurrentFeeds()
			if err != nil {
				// Close unconditionally on DB error
				_ = r.Close()
				return err
			}

			glog.V(4).Infof("Got feeds: %s", newFeedsArray)

			var newFeeds = make(map[int64]*structs.Feed)
			for _, e := range newFeedsArray {
				newFeeds[e.ID()] = e
			}

			// Critical section for communicating on channels and spawning new routines.
			r.closeLock.Lock()

			if r.closed {
				r.closeLock.Unlock()
				glog.Info("rssFetcher closed, exiting")
				return nil
			}

			r.mapLock.Lock()
			r.killOldRoutines(newFeeds)
			r.startNewRoutines(newFeedsArray)
			r.feeds = newFeeds
			r.mapLock.Unlock()

			r.closeLock.Unlock()

			r.lastPolled = time.Now()
		}

		select {
		case fe := <-r.errorChan:
			r.restartFailedRoutine(fe)
			// TODO -- handle an update from fe.f
		case <-r.closeChan:
			glog.Info("rssFetcher closed, exiting")
			return nil
		case <-time.After(dbPollPeriod - time.Since(r.lastPolled)):
			// This polling is the last line of defense against out of band edits
			// TODO -- Add another routine to handle single updates
		}
	}
}

func (r *rssFetcher) rateLimitRoutines() {
LimitLoop:
	for true {
		select {
		case r.rateLimitChan <- struct{}{}:
			glog.V(2).Info("Allowed one routine to proceed")
		case <-r.closeChan:
			break LimitLoop
		}

		select {
		case <-time.After(startupRateLimit):
		case <-r.closeChan:
			break LimitLoop
		}
	}

	glog.Info("Killed rate limit routine")
	close(r.rateLimitChan)
	r.wg.Done()
}

// Main work done here for each feed
// TODO -- clean this up and refactor it
func (r *rssFetcher) routine(f *structs.Feed, kill <-chan struct{}) {
	defer func() {
		if rec := recover(); rec != nil {
			err := rec.(error)
			if glog.V(1) {
				glog.Error(err)
			}
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
		glog.V(3).Infof("Routine for [%s] completed", f)
		r.wg.Done()
		// We could attempt to send f on feedUpdateChan but
		// Any important updates should come through the webserver
	}()

	parser := gofeed.NewParser()

	glog.V(1).Infof("Routine for [%s] started", f)
	for {
		// The feed may have been updated
		r.mapLock.RLock()
		newF, ok := r.feeds[f.ID()]
		r.mapLock.RUnlock()
		if !ok {
			select {
			case <-kill:
				glog.V(1).Infof("Routine for [%s] killed by parent", f)
			default:
				// Should never happen
				glog.Warningf("Feed [%s] unexpectedly missing", f)
			}
			return
		}
		f = newF

		body := ""
		if strings.HasPrefix(f.URL(), "!") {
			body = r.runExternalCommandFeed(f, kill)
		} else {
			body = r.fetchHTTPFeed(f, kill)
		}

		select {
		case <-kill:
			glog.V(1).Infof("Routine for [%s] killed by parent", f)
			return
		default:
		}

		feed, err := parser.ParseString(body)
		if err != nil {
			glog.Errorf("Error calling parser.ParseString for [%s]: %v", f, err)
			glog.Info("TEMPORARY DEBUGGING")
			glog.Info(body)
			if strings.Contains(f.URL(), "mangadex.org") {
				r.cloudflare.setInvalid("mangadex.org")
			}
			panic(err)
		}

		f, err = r.db.MutateFeed(
			f.ID(), structs.FeedMergeGofeed(feed))
		if err != nil {
			glog.Errorf("Error updating feed [%s]: %v", f, err)
			panic(err)
		}

		err = r.db.InsertItems(structs.CreateNewItems(f, feed.Items))
		if err != nil {
			glog.Errorf("Error inserting items for feed [%s]: %v", f, err)
			panic(err)
		}

		f, err = r.db.MutateFeed(
			f.ID(), structs.FeedSetFetchSuccess)
		if err != nil {
			glog.Errorf("Error updating feed [%s]: %v", f, err)
			panic(err)
		}

		/*
			TODO -- Do this
			select {
			case r.feedUpdateChan <- f:
			case <-kill:
				glog.V(1).Infof("Routine for [%s] killed by parent", f)
				return
			}*/

		r.mapLock.Lock()
		if _, ok := r.retryBackoff[f.ID()]; ok {
			r.retryBackoff[f.ID()] = time.Minute
		}
		r.mapLock.Unlock()

		select {
		case <-kill:
			glog.V(1).Infof("Routine for [%s] killed by parent", f)
			return
		case <-time.After(r.getSleepTime(f, feed, body)):
		}
	}
}

func (r *rssFetcher) runExternalCommandFeed(f *structs.Feed, kill <-chan struct{}) string {
	<-r.rateLimitChan
	select {
	case <-kill:
		return ""
	default:
	}

	output, err := exec.Command("sh", "-c", f.URL()[1:]).Output()

	// Check immediately after the command
	// If this has been killed do not write updates to the DB
	select {
	case <-kill:
		return ""
	default:
	}

	if err != nil {
		glog.Errorf("Error running external command for [%s]: %v", f, err)
		panic(err)
	}

	return string(output)
}

func (r *rssFetcher) fetchHTTPFeed(f *structs.Feed, kill <-chan struct{}) string {
	<-r.rateLimitChan
	select {
	case <-kill:
		return ""
	default:
	}
	// Grab the cookie after rate limiting to maximize the chance that fetching
	// starts before the next thread is through
	// It is possible to do this in a way that doesn't rely on chance but I don't
	// think it's worth the complexity.
	c, ua, blocked, err := r.cloudflare.getCookie(f.URL())
	if err != nil {
		glog.Errorf("Error calling cloudflare.getCookie() for [%s]: %v", f, err)
		panic(err)
	}
	if blocked {
		// If we blocked there might be a large number of threads trying to proceed
		// at once
		<-r.rateLimitChan
		select {
		case <-kill:
			return ""
		default:
		}
	}
	body := r.fetchHTTPBody(f, kill, c, ua)

	cf, err := r.cloudflare.isCloudflareResponse(f.URL(), body)
	if err != nil {
		glog.Errorf("Error calling isCloudflareResponse() for [%s]: %v", f, err)
		panic(err)
	}
	if cf {
		// We don't need to rate limit GetNewCookie
		select {
		case <-kill:
			return ""
		default:
		}
		c, ua, err := r.cloudflare.getNewCookie(f.URL())
		select {
		case <-kill:
			return ""
		default:
		}
		if err != nil {
			glog.Errorf("Error calling cloudflare.GetNewCookie for [%s]: %v", f, err)
			glog.Error("Body was: \n" + body)
			panic(err)
		}

		<-r.rateLimitChan
		select {
		case <-kill:
			return ""
		default:
		}
		body = r.fetchHTTPBody(f, kill, c, ua)
	}

	return quirks.HandleBodyQuirks(f, body)
}

func (r *rssFetcher) fetchHTTPBody(
	f *structs.Feed,
	kill <-chan struct{},
	cookie string,
	userAgent string) string {
	req, err := http.NewRequest("GET", f.URL(), nil)
	checkErrMaybePanic(err)

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
		glog.Errorf("Error calling httpClient.Get for [%s]: %v", f, err)
		panic(err)
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	// Close unconditionally to avoid memory leaks
	_ = resp.Body.Close()
	if err != nil {
		glog.Errorf("Error reading response body for [%s]: %v", f, err)
		panic(err)
	}

	return string(bodyBytes)
}

func (r *rssFetcher) getSleepTime(f *structs.Feed, feed *gofeed.Feed, body string) time.Duration {
	sleepTime := minPollPeriod
	if feed.FeedType == "rss" {
		rssFeed, err := (&gofeedRss.Parser{}).Parse(strings.NewReader(body))
		if err != nil {
			glog.Warningf("RSS feed could not be parsed as RSS [%s]", f)
		} else if rssFeed.TTL != "" {
			ttl, err := strconv.Atoi(rssFeed.TTL)
			if err != nil {
				glog.Warningf("RSS feed [%s] had invalid TTL %s", f, rssFeed.TTL)
			} else {
				sleepTime = time.Duration(ttl) * time.Minute
			}
		}
	}
	if sleepTime < minPollPeriod {
		glog.V(3).Infof("Poll period for feed [%s] was %s; using minPollPeriod", f, sleepTime)
		sleepTime = minPollPeriod
	}

	glog.V(4).Infof("Waiting %d seconds until next update for [%s]", sleepTime/time.Second, f)
	return sleepTime
}

func (r *rssFetcher) killRoutine(f *structs.Feed) {
	routine, ok := r.routines[f.ID()]
	if !ok {
		glog.Warningf("Tried to kill non-existent routine for [%s]", f)
		return
	}
	glog.V(2).Infof("Killing routine for [%s]", f)
	close(routine)
	delete(r.routines, f.ID())
	delete(r.retryBackoff, f.ID())
}

func (r *rssFetcher) restartRoutine(
	f *structs.Feed, kill <-chan struct{}, delay time.Duration) {
	glog.V(1).Infof(
		"Restarting routine for [%s] in %s", f, delay)

	select {
	case <-kill:
		glog.V(1).Infof(
			"Routine for [%s] killed by parent before it could restart", f)
		r.wg.Done()
		return
	case <-time.After(delay):
		glog.V(2).Infof("Restarting routine for [%s] now", f)
		r.routine(f, kill)
	}
}

func (r *rssFetcher) killOldRoutines(newFeeds map[int64]*structs.Feed) {
	glog.V(5).Info("killOldRoutines() started")

	for i, f := range r.feeds {
		if _, ok := newFeeds[i]; !ok {
			r.killRoutine(f)
		}
	}

	glog.V(5).Info("killOldRoutines() completed")
}

func (r *rssFetcher) startNewRoutines(newFeeds []*structs.Feed) {
	glog.V(5).Info("startNewRoutines() started")

	for _, feed := range newFeeds {
		if _, ok := r.feeds[feed.ID()]; !ok {
			glog.V(2).Infof("Starting new routine for [%s]", feed)

			r.routines[feed.ID()] = make(chan struct{})
			r.retryBackoff[feed.ID()] = time.Minute

			r.wg.Add(1)
			go r.routine(feed, r.routines[feed.ID()])
		}
	}
	glog.V(5).Info("startNewRoutines() completed")
}

func (r *rssFetcher) restartFailedRoutine(fe feedError) {
	feed := fe.f
	id := feed.ID()

	r.mapLock.Lock()
	defer r.mapLock.Unlock()

	_, ok := r.feeds[id]
	if !ok {
		glog.Warningf("Tried to restart routine for non-existent feed %d", feed)
		return
	}
	backoff := r.retryBackoff[id]
	if isCloudflareError(fe.err) {
		backoff = time.Hour * 6
	}
	r.killRoutine(feed)
	r.routines[id] = make(chan struct{})
	r.retryBackoff[id] = backoff * 2
	if r.retryBackoff[id] > time.Hour*6 {
		r.retryBackoff[id] = time.Hour * 6 // Check at least once every six hours
	}

	glog.Warningf(
		"Error in routine for [Feed: %d], attempting to restart in %s",
		id, backoff)
	r.wg.Add(1)
	go r.restartRoutine(feed, r.routines[id], backoff)
}

/*func charsetReader(charset string, r io.Reader) (io.Reader, error) {
	if charset == "ISO-8859-1" || charset == "iso-8859-1" {
		return r, nil
	}
	return nil, fmt.Errorf("Unsupported character set encoding: %s", charset)
}*/

func checkErrMaybePanic(err error) {
	if err != nil {
		glog.ErrorDepth(1, err)
		panic(err)
	}
}
