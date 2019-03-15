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
	errUntrustedHost     = errors.New("Host not trusted for cloudflare bypass")
	errUnsecureTransport = errors.New("Cloudflare bypass requires https")
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

func cloudflareSupported(feedURL string) bool {
	_, err := host(feedURL)
	return err == nil
}

func host(feedURL string) (string, error) {
	u, err := url.Parse(feedURL)
	if err != nil {
		return "", err
	}

	if u.Scheme != "https" {
		return "", errUnsecureTransport
	}

	if !trustedHosts[u.Host] {
		glog.V(1).Infof("Host [%s] not trusted for cloudflare bypass", u.Host)
		return "", errUntrustedHost
	}

	return u.Host, nil
}

func isCloudflareResponse(body string) bool {
	return strings.Contains(body, "This process is automatic. Your browser "+
		"will redirect to your requested content shortly.")
}

func (cf *cloudflare) getCookie(feedURL string) (
	cookie string, userAgent string, blocked bool) {
	h, err := host(feedURL)
	if err != nil {
		return "", "", false
	}

	cf.fetchingLock.Lock()
	fetchChan, blocking := cf.fetching[h]
	cf.fetchingLock.Unlock()

	if blocking {
		select {
		case <-fetchChan:
		case <-cf.closeChan:
			return "", "", true
		}
	}

	c, ua := cf.getExistingCookie(h)
	return c, ua, blocking
}

func (cf *cloudflare) getExistingCookie(h string) (string, string) {
	cf.cookieLock.RLock()
	defer cf.cookieLock.RUnlock()

	c := cf.cookies[h]
	ua := cf.userAgents[h]
	return c, ua
}

func (cf *cloudflare) getNewCookie(feedURL string) (string, string, error) {
	select {
	case <-cf.closeChan:
		return "", "", nil
	default:
	}

	h, err := host(feedURL)
	if err != nil {
		return "", "", err
	}

	cf.fetchingLock.Lock()
	fetchChan, ok := cf.fetching[h]
	if !ok {
		fetchChan = make(chan struct{})
		cf.fetching[h] = fetchChan
		defer cf.stopFetching(h)
	}
	cf.fetchingLock.Unlock()

	if ok {
		select {
		case <-fetchChan:
		case <-cf.closeChan:
			return "", "", nil
		}
		c, ua := cf.getExistingCookie(h)
		if c != "" {
			return c, ua, nil
		}
		// Abort and retry through the normal mechamism
		return c, ua,
			errors.New("Another thread failed to fetch cloudflare cookies")
	}

	// This thread is now responsible for fetching cookies for this host
	cf.pythonLock.Lock()
	defer cf.pythonLock.Unlock()
	select {
	case <-cf.closeChan:
		return "", "", nil
	default:
	}

	glog.Infof("Fetching new cloudflare cookie for [%s]", h)
	return cf.runPython(feedURL, h)
}

func (cf *cloudflare) runPython(feedURL, host string) (string, string, error) {
	out, err :=
		exec.Command("python3", "-c", cookieScript, feedURL).CombinedOutput()
	if err != nil {
		glog.Warning(string(out))
		return "", "", err
	}

	lines := strings.Split(string(out), "\n")

	if len(lines) < 2 {
		return "", "",
			errors.New("Missing cloudflare cookie or user agent for " + feedURL)
	}

	cf.cookieLock.Lock()
	cf.cookies[host] = lines[0]
	cf.userAgents[host] = lines[1]
	cf.cookieLock.Unlock()

	return lines[0], lines[1], nil
}

func (cf *cloudflare) stopFetching(h string) {
	cf.fetchingLock.Lock()
	c := cf.fetching[h]
	close(c)
	delete(cf.fetching, h)
	cf.fetchingLock.Unlock()
}
