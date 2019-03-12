package rssfetcher

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/awused/rss-aggregator/backend/database"
	. "github.com/awused/rss-aggregator/backend/structs"
	"github.com/golang/glog"
	"github.com/mmcdole/gofeed"
	gofeedRss "github.com/mmcdole/gofeed/rss"
)

const dbPollPeriod = time.Duration(time.Minute * 5)
const minPollPeriod = time.Duration(time.Minute * 10)
const rssTimeout = 30 * time.Second
const startupRateLimit = 250 * time.Millisecond

type RssFetcher interface {
	Run() error
	Close() error
}

type rssFetcher struct {
	db            *database.Database
	httpClient    *http.Client
	cloudflare    *cloudflare
	feeds         map[int64]*Feed
	routines      map[int64]chan struct{}
	retryBackoff  map[int64]int64
	mapLock       sync.RWMutex
	lastPolled    time.Time
	wg            sync.WaitGroup
	errorChan     chan int64
	rateLimitChan chan struct{}
	closed        bool
	closeChan     chan struct{}
	closeLock     sync.Mutex
}

func NewRssFetcher() (r *rssFetcher, err error) {
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
		//TODO -- probably remove entirely //Transport: filteredRoundTripper{},
	}
	rss.feeds = make(map[int64]*Feed)
	rss.routines = make(map[int64]chan struct{})
	rss.retryBackoff = make(map[int64]int64)
	rss.rateLimitChan = make(chan struct{})
	rss.closeChan = make(chan struct{})
	rss.errorChan = make(chan int64)

	rss.cloudflare = newCloudflare(rss.closeChan)

	glog.V(5).Info("rssFetcher() completed")
	return &rss, nil
}

func (this *rssFetcher) Close() error {
	glog.Info("Closing rssFetcher")

	if this.closed {
		glog.Warning("Tried to close rssFetcher that has already been closed")
		return nil
	}
	this.closeLock.Lock()
	defer this.closeLock.Unlock()
	if this.closed {
		glog.Warning("Tried to close rssFetcher that has already been closed")
		return nil
	}
	// Close and kill the main routine last
	close(this.closeChan)
	this.closed = true

	this.mapLock.Lock()
	this.killOldRoutines(map[int64]*Feed{})
	this.feeds = map[int64]*Feed{}
	this.mapLock.Unlock()

	glog.Infof("Waiting up to 60 seconds for goroutines to finish")

	var c = make(chan struct{})
	go func() {
		this.wg.Wait()
		close(c)
	}()

	select {
	case <-time.After(time.Second * 60):
		glog.Errorf("Some goroutines failed to exit within 60 seconds")
	case <-c:
		glog.Info("All goroutines exited successfully")
	}

	defer glog.Info("Close() completed")
	return this.db.Close()
}

func (this *rssFetcher) Run() (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()

	this.wg.Add(1)
	go this.rateLimitRoutines()

	glog.Info("rssFetcher started running")
	for {
		if this.lastPolled.IsZero() || time.Since(this.lastPolled) > dbPollPeriod {
			glog.V(3).Info("Checking database for new feeds")

			newFeedsArray, err := this.db.GetFeeds(false)
			if err != nil {
				// Close unconditionally on DB error
				_ = this.Close()
				return err
			}

			glog.V(4).Infof("Got feeds: %s", newFeedsArray)

			var newFeeds = make(map[int64]*Feed)
			for _, e := range newFeedsArray {
				newFeeds[e.Id()] = e
			}

			// Critical section for communicating on channels and spawning new routines.
			this.closeLock.Lock()

			if this.closed {
				this.closeLock.Unlock()
				glog.Info("rssFetcher closed, exiting")
				return nil
			}

			this.mapLock.Lock()
			this.killOldRoutines(newFeeds)
			this.startNewRoutines(newFeedsArray)
			this.feeds = newFeeds
			this.mapLock.Unlock()

			this.closeLock.Unlock()

			this.lastPolled = time.Now()
		}

		select {
		case id := <-this.errorChan:
			this.mapLock.RLock()
			feed, ok := this.feeds[id]
			backoff := this.retryBackoff[feed.Id()]
			this.mapLock.RUnlock()
			if ok {
				glog.Warningf(
					"Error in routine for [Feed: %d], attempting to restart in %d minutes",
					feed.Id(), backoff)
				this.restartFailedRoutine(id)
			} else {
				glog.Warningf("Error in routine for feed %d", id)
			}
		case <-this.closeChan:
			glog.Info("rssFetcher closed, exiting")
			return nil
		case <-time.After(dbPollPeriod - time.Since(this.lastPolled)):
		}
	}
}

func (this *rssFetcher) rateLimitRoutines() {
LimitLoop:
	for true {
		select {
		case this.rateLimitChan <- struct{}{}:
			glog.V(2).Info("Allowed one routine to proceed")
		case <-this.closeChan:
			break LimitLoop
		}

		select {
		case <-time.After(startupRateLimit):
		case <-this.closeChan:
			break LimitLoop
		}
	}

	glog.Info("Killed rate limit routine")
	close(this.rateLimitChan)
	this.wg.Done()
}

// Main work done here for each feed
// TODO -- clean this up and refactor it
func (this *rssFetcher) routine(f *Feed, kill <-chan struct{}) {
	defer func() {
		if r := recover(); r != nil {
			if glog.V(1) {
				glog.Error(r.(error))
			}
			select {
			case this.errorChan <- f.Id():
			case <-this.closeChan:
			}
		}
		glog.V(3).Infof("Routine for [%s] completed", f)
		this.wg.Done()
	}()

	parser := gofeed.NewParser()

	glog.V(1).Infof("Routine for [%s] started", f)
	for {
		// The feed may have been updated
		this.mapLock.RLock()
		newF, ok := this.feeds[f.Id()]
		oldBackoff := this.retryBackoff[f.Id()]
		this.mapLock.RUnlock()
		if ok {
			// Feed may not be present if the routine is being removed
			f = newF
		}

		body := ""
		if strings.HasPrefix(f.Url(), "!") {
			body = this.runExternalCommandFeed(f, kill)
		} else {
			body = this.fetchHTTPFeed(f, kill)
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
			f.LastFetchFailed = true
			checkErrMaybePanic(this.db.NonUserUpdateFeed(f))
			panic(err)
		}

		// The feed may have been updated
		this.mapLock.RLock()
		newF, ok = this.feeds[f.Id()]
		this.mapLock.RUnlock()
		if ok {
			// Feed may not be present if the routine is being removed
			f = newF
		}

		f.LastFetchFailed = false
		f.LastSuccessTime = time.Now().UTC()
		if oldBackoff != 1 {
			this.mapLock.Lock()
			this.retryBackoff[f.Id()] = 1
			this.mapLock.Unlock()
		}
		f.HandleUpdate(feed)
		err = this.db.NonUserUpdateFeed(f)
		if err != nil {
			glog.Errorf("Error updating feed [%s]: %v", f, err)
			f.LastFetchFailed = true
			checkErrMaybePanic(this.db.NonUserUpdateFeed(f))
			panic(err)
		}

		err = this.db.InsertItems(CreateNewItems(f, feed.Items))
		if err != nil {
			glog.Errorf("Error inserting items for feed [%s]: %v", f, err)
			f.LastFetchFailed = true
			checkErrMaybePanic(this.db.NonUserUpdateFeed(f))
			panic(err)
		}

		select {
		case <-kill:
			glog.V(1).Infof("Routine for [%s] killed by parent", f)
			return
		case <-time.After(this.getSleepTime(f, feed, body)):
		}
	}
}

func (this *rssFetcher) runExternalCommandFeed(f *Feed, kill <-chan struct{}) string {
	<-this.rateLimitChan
	select {
	case <-kill:
		return ""
	default:
	}

	output, err := exec.Command("sh", "-c", f.Url()[1:]).Output()

	// Check immediately after the command
	// If this has been killed do not write updates to the DB
	select {
	case <-kill:
		return ""
	default:
	}

	if err != nil {
		glog.Errorf("Error running external command for [%s]: %v", f, err)
		f.LastFetchFailed = true
		checkErrMaybePanic(this.db.NonUserUpdateFeed(f))
		panic(err)
	}

	return string(output)
}

func (this *rssFetcher) fetchHTTPFeed(f *Feed, kill <-chan struct{}) string {
	<-this.rateLimitChan
	select {
	case <-kill:
		return ""
	default:
	}
	// Grab the cookie after rate limiting to maximize the chance that fetching
	// starts before the next thread is through
	// It is possible to do this in a way that doesn't rely on chance but I don't
	// think it's worth the complexity.
	c, ua, blocked := this.cloudflare.GetCookie(f.Url())
	if blocked {
		// If we blocked there might be a large number of threads trying to proceed
		// at once
		<-this.rateLimitChan
		select {
		case <-kill:
			return ""
		default:
		}
	}
	body := this.fetchHTTPBody(f, kill, c, ua)

	if CloudflareSupported(f.Url()) && IsCloudflareResponse(body) {
		// We don't need to rate limit GetNewCookie
		select {
		case <-kill:
			return ""
		default:
		}
		c, ua, err := this.cloudflare.GetNewCookie(f.Url())
		select {
		case <-kill:
			return ""
		default:
		}
		if err == ErrUnsecureTransport || err == ErrUntrustedHost {
			// Should never happen
			return body
		}
		if err != nil {
			glog.Errorf("Error calling cloudflare.GetNewCookie for [%s]: %v", f, err)
			f.LastFetchFailed = true
			checkErrMaybePanic(this.db.NonUserUpdateFeed(f))
			panic(err)
		}

		<-this.rateLimitChan
		select {
		case <-kill:
			return ""
		default:
		}
		body = this.fetchHTTPBody(f, kill, c, ua)
	}

	return body
}

func (this *rssFetcher) fetchHTTPBody(
	f *Feed,
	kill <-chan struct{},
	cookie string,
	userAgent string) string {
	req, err := http.NewRequest("GET", f.Url(), nil)
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

	resp, err := this.httpClient.Do(req)
	// Check immediately after the HTTP request
	// If this has been killed do not write updates to the DB
	select {
	case <-kill:
		return ""
	default:
	}

	if err != nil {
		glog.Errorf("Error calling httpClient.Get for [%s]: %v", f, err)
		f.LastFetchFailed = true
		checkErrMaybePanic(this.db.NonUserUpdateFeed(f))
		panic(err)
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	// Close unconditionally to avoid memory leaks
	_ = resp.Body.Close()
	if err != nil {
		glog.Errorf("Error reading response body for [%s]: %v", f, err)
		f.LastFetchFailed = true
		checkErrMaybePanic(this.db.NonUserUpdateFeed(f))
		panic(err)
	}

	return string(bodyBytes)
}

func (this *rssFetcher) getSleepTime(f *Feed, feed *gofeed.Feed, body string) time.Duration {
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

func (this *rssFetcher) killRoutine(f *Feed) {
	routine, ok := this.routines[f.Id()]
	if !ok {
		glog.Warningf("Tried to kill non-existent routine for [%s]", f)
		return
	}
	glog.V(2).Infof("Killing routine for [%s]", f)
	close(routine)
	delete(this.routines, f.Id())
	delete(this.retryBackoff, f.Id())
}

func (this *rssFetcher) restartRoutine(f *Feed, c <-chan struct{}, minutes int64) {
	glog.V(1).Infof("Restarting routine for [%s] in %d minutes", f, minutes)

	select {
	case <-c:
		glog.V(1).Infof("Routine for [%s] killed by parent before it could restart", f)
		this.wg.Done()
		return
	case <-time.After(time.Minute * time.Duration(minutes)):
		glog.V(2).Infof("Restarting routine for [%s] now", f)
		this.routine(f, c)
	}
}

func (this *rssFetcher) killOldRoutines(newFeeds map[int64]*Feed) {
	glog.V(5).Info("killOldRoutines() started")

	for i, f := range this.feeds {
		if _, ok := newFeeds[i]; !ok {
			this.killRoutine(f)
		}
	}

	glog.V(5).Info("killOldRoutines() completed")
}

func (this *rssFetcher) startNewRoutines(newFeeds []*Feed) {
	glog.V(5).Info("startNewRoutines() started")

	for _, feed := range newFeeds {
		if _, ok := this.feeds[feed.Id()]; !ok {
			glog.V(2).Infof("Starting new routine for [%s]", feed)

			this.routines[feed.Id()] = make(chan struct{})
			this.retryBackoff[feed.Id()] = 1

			this.wg.Add(1)
			go this.routine(feed, this.routines[feed.Id()])
		}
	}
	glog.V(5).Info("startNewRoutines() completed")
}

func (this *rssFetcher) restartFailedRoutine(id int64) {
	this.mapLock.Lock()
	defer this.mapLock.Unlock()

	feed, ok := this.feeds[id]
	if !ok {
		glog.Warningf("Tried to restart routine for non-existent feed %d", feed)
		return
	}
	backoffMinutes := this.retryBackoff[id]
	this.killRoutine(feed)
	this.routines[id] = make(chan struct{})
	this.retryBackoff[id] = backoffMinutes * 2
	if this.retryBackoff[id] > 6*60 {
		this.retryBackoff[id] = 6 * 60 // Check at least once every six hours
	}

	this.wg.Add(1)
	go this.restartRoutine(feed, this.routines[id], backoffMinutes)
}

func charsetReader(charset string, r io.Reader) (io.Reader, error) {
	if charset == "ISO-8859-1" || charset == "iso-8859-1" {
		return r, nil
	}
	return nil, fmt.Errorf("Unsupported character set encoding: %s", charset)
}

func checkErrMaybePanic(err error) {
	if err != nil {
		glog.ErrorDepth(1, err)
		panic(err)
	}
}
