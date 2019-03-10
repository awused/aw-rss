package rssfetcher

import (
	"errors"
	"net/url"
	"os/exec"
	"strings"
	"sync"
)

const cookieScript = `
import cfscrape
import sys

scraper = cfscrape.create_scraper()  # returns a requests.Session object
c, ua = cfscrape.get_cookie_string(sys.argv[1])
print(c)
print(ua)
`

type cloudflare struct {
	cookies      map[string]string
	userAgents   map[string]string
	cookieLock   sync.RWMutex
	fetching     map[string]chan struct{}
	fetchingLock sync.Mutex
	pythonLock   sync.Mutex
	closeChan    <-chan struct{}
}

func newCloudflare(closeChan <-chan struct{}) *cloudflare {
	return &cloudflare{
		cookies:    make(map[string]string),
		userAgents: make(map[string]string),
		fetching:   make(map[string]chan struct{}),
		closeChan:  closeChan,
	}
}

func host(feedUrl string) (string, error) {
	u, err := url.Parse(feedUrl)
	if err != nil {
		return "", err
	}

	return u.Host, nil
}

func (this *cloudflare) getExistingCookie(feedUrl string) (string, string, bool) {
	h, err := host(feedUrl)
	if err != nil {
		return "", "", false
	}

	this.cookieLock.RLock()
	defer this.cookieLock.RUnlock()

	c, has := this.cookies[h]
	ua := this.userAgents[h]
	return c, ua, has
}

func (this *cloudflare) getNewCookie(feedUrl string) (string, string, error) {
	select {
	case <-this.closeChan:
		return "", "", nil
	default:
	}

	h, err := host(feedUrl)
	if err != nil {
		return "", "", err
	}

	this.fetchingLock.Lock()
	fetchChan, ok := this.fetching[h]
	if !ok {
		fetchChan = make(chan struct{})
		this.fetching[h] = fetchChan
		defer this.stopFetching(h)
	}
	this.fetchingLock.Unlock()

	if ok {
		select {
		case <-fetchChan:
		case <-this.closeChan:
			return "", "", nil
		}
		c, ua, b := this.getExistingCookie(feedUrl)
		if b {
			return c, ua, nil
		} else {
			return c, ua,
				errors.New("Another thread failed to fetch cloudflare cookies")
		}
	}

	// This thread is now responsible for fetching cookies for this host
	this.pythonLock.Lock()
	defer this.pythonLock.Unlock()
	select {
	case <-this.closeChan:
		return "", "", nil
	default:
	}

	return this.runPython(feedUrl, h)
}

func (this *cloudflare) runPython(feedUrl, host string) (string, string, error) {
	out, err := exec.Command("python3", "-c", cookieScript, feedUrl).Output()
	if err != nil {
		return "", "", err
	}

	lines := strings.Split(string(out), "\n")

	if len(lines) < 2 {
		return "", "",
			errors.New("Missing cloudflare cookie or user agent for " + feedUrl)
	}

	this.cookieLock.Lock()
	this.cookies[host] = lines[0]
	this.userAgents[host] = lines[1]
	this.cookieLock.Unlock()

	return lines[0], lines[1], nil
}

func (this *cloudflare) stopFetching(h string) {
	this.fetchingLock.Lock()
	c := this.fetching[h]
	close(c)
	delete(this.fetching, h)
	this.fetchingLock.Unlock()
}
