package rssfetcher

import (
	"errors"
	"net/url"
	"os/exec"
	"strings"
	"sync"

	"github.com/golang/glog"
)

const cookieScript = `
import cfscrape
import sys

scraper = cfscrape.create_scraper()  # returns a requests.Session object
c, ua = cfscrape.get_cookie_string(sys.argv[1])
print(c)
print(ua)
`

// This needs to be configureable by the user
// Don't run arbitrary JS from untrusted domains, only handle problematic sites
// as they're identified
var trustedHosts = map[string]bool{
	"mangadex.org": true,
}

var (
	ErrUntrustedHost     = errors.New("Host not trusted for cloudflare bypass")
	ErrUnsecureTransport = errors.New("Cloudflare bypass requires https")
)

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

func CloudflareSupported(feedUrl string) bool {
	_, err := host(feedUrl)
	return err == nil
}

func host(feedUrl string) (string, error) {
	u, err := url.Parse(feedUrl)
	if err != nil {
		return "", err
	}

	if u.Scheme != "https" {
		return "", ErrUnsecureTransport
	}

	if !trustedHosts[u.Host] {
		glog.V(1).Infof("Host [%s] not trusted for cloudflare bypass", u.Host)
		return "", ErrUntrustedHost
	}

	return u.Host, nil
}

func IsCloudflareResponse(body string) bool {
	return strings.Contains(body, "This process is automatic. Your browser "+
		"will redirect to your requested content shortly.")
}

func (this *cloudflare) GetCookie(feedUrl string) (
	cookie string, userAgent string, blocked bool) {
	h, err := host(feedUrl)
	if err != nil {
		return "", "", false
	}

	this.fetchingLock.Lock()
	fetchChan, blocking := this.fetching[h]
	this.fetchingLock.Unlock()

	if blocking {
		select {
		case <-fetchChan:
		case <-this.closeChan:
			return "", "", true
		}
	}

	c, ua := this.getExistingCookie(h)
	return c, ua, blocking
}

func (this *cloudflare) getExistingCookie(h string) (string, string) {
	this.cookieLock.RLock()
	defer this.cookieLock.RUnlock()

	c := this.cookies[h]
	ua := this.userAgents[h]
	return c, ua
}

func (this *cloudflare) GetNewCookie(feedUrl string) (string, string, error) {
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
		c, ua := this.getExistingCookie(h)
		if c != "" {
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

	glog.Infof("Fetching new cloudflare cookie for [%s]", h)
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
