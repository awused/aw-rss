package rssfetcher

// TODO -- move to its own module

import (
	"errors"
	"net/url"
	"os/exec"
	"strings"
	"sync"
	"time"

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
	errUntrustedHost      = errors.New("Host not trusted for cloudflare bypass")
	errUnsecureTransport  = errors.New("Cloudflare bypass requires https")
	errCloudflareCaptcha  = errors.New("Cloudflare bypass cannot handle the captcha challenges")
	errCloudflareBroken   = errors.New("Cfscrape is missing or out of date")
	errCloudflareCooldown = errors.New("Cloudflare bypass previously failed, waiting up to 6 hours")
	errNoCookies          = errors.New("Another thread failed to fetch cloudflare cookies")
)

func isCloudflareError(err error) bool {
	return err == errUntrustedHost ||
		err == errUnsecureTransport ||
		err == errCloudflareCaptcha ||
		err == errCloudflareBroken ||
		err == errCloudflareCooldown ||
		err == errNoCookies
}

const cloudflareSentinelOne = "<title>Attention Required! | Cloudflare</title>"
const cloudflareSentinelTwo = "<title>Just a moment...</title>"
const cloudflareNormal = "This process is automatic. Your browser " +
	"will redirect to your requested content shortly."
const cloudflareMissing = "ModuleNotFoundError: No module named 'cfscrape'"
const cloudflareBroken = "Cloudflare may have changed their technique," +
	"or there may be a bug in the script."

type cloudflare struct {
	cookies      map[string]string
	userAgents   map[string]string
	cookieLock   sync.RWMutex
	fetching     map[string]chan struct{}
	fetchingLock sync.Mutex
	pythonLock   sync.Mutex
	closeChan    <-chan struct{}

	brokenCfscrape bool
	// If we have permanent failures, disable these hosts for 6 hours at a time
	invalidUntil map[string]time.Time
	failureLock  sync.Mutex
}

func newCloudflare(closeChan <-chan struct{}) *cloudflare {
	return &cloudflare{
		cookies:      make(map[string]string),
		userAgents:   make(map[string]string),
		fetching:     make(map[string]chan struct{}),
		closeChan:    closeChan,
		invalidUntil: make(map[string]time.Time),
	}
}

func host(feedURL string) (string, string, error) {
	u, err := url.Parse(feedURL)
	if err != nil {
		return "", "", err
	}

	return u.Host, u.Scheme, nil
}

func (cf *cloudflare) isCloudflareResponse(feedURL string, body string) (bool, error) {
	if len(body) < 500 {
		return false, nil
	}
	if !strings.Contains(body[0:499], cloudflareSentinelOne) &&
		!strings.Contains(body[0:499], cloudflareSentinelTwo) {
		return false, nil
	}
	if strings.Contains(body, cloudflareNormal) {
		return true, nil
	}

	h, _, err := host(feedURL)
	if err == nil {
		cf.setInvalid(h)
	}
	return true, errCloudflareCaptcha
}

func (cf *cloudflare) getCookie(feedURL string) (
	cookie string, userAgent string, blocked bool, permanentFailure error) {
	h, scheme, err := host(feedURL)
	if err != nil {
		return "", "", false, nil
	}

	cf.failureLock.Lock()
	inv := cf.invalidUntil[h]
	broken := cf.brokenCfscrape
	cf.failureLock.Unlock()

	if time.Now().Before(inv) {
		if broken {
			return "", "", false, errCloudflareBroken
		}
		return "", "", false, errCloudflareCooldown
	}

	if scheme != "https" || !trustedHosts[h] {
		return "", "", false, nil
	}

	cf.fetchingLock.Lock()
	fetchChan, blocking := cf.fetching[h]
	cf.fetchingLock.Unlock()

	if blocking {
		select {
		case <-fetchChan:
		case <-cf.closeChan:
			return "", "", true, nil
		}
	}

	c, ua, err := cf.getExistingCookie(h)
	if err == errNoCookies {
		err = nil
	}
	return c, ua, blocking, err
}

func (cf *cloudflare) getExistingCookie(
	h string) (string, string, error) {

	cf.failureLock.Lock()
	inv := cf.invalidUntil[h]
	broken := cf.brokenCfscrape
	cf.failureLock.Unlock()

	if time.Now().Before(inv) {
		if broken {
			return "", "", errCloudflareBroken
		}
		return "", "", errCloudflareCooldown
	}

	cf.cookieLock.RLock()
	defer cf.cookieLock.RUnlock()

	c, ok := cf.cookies[h]
	ua := cf.userAgents[h]
	if !ok {
		return "", "", errNoCookies
	}
	return c, ua, nil
}

func (cf *cloudflare) getNewCookie(
	feedURL string) (cookie string, userAgent string, err error) {
	select {
	case <-cf.closeChan:
		return "", "", nil
	default:
	}

	h, scheme, err := host(feedURL)
	if err != nil {
		return "", "", err
	}

	cf.failureLock.Lock()
	inv := cf.invalidUntil[h]
	broken := cf.brokenCfscrape
	cf.failureLock.Unlock()

	if time.Now().Before(inv) {
		if broken {
			return "", "", errCloudflareBroken
		}
		return "", "", errCloudflareCooldown
	}

	if scheme != "https" || !trustedHosts[h] {
		cf.setInvalid(h)
		if scheme != "https" {
			return "", "", errUnsecureTransport
		}
		return "", "", errUntrustedHost
	}

	cf.fetchingLock.Lock()
	fetchChan, ok := cf.fetching[h]
	if !ok {
		fetchChan = make(chan struct{})
		cf.fetching[h] = fetchChan
		defer cf.finishFetching(h)
	}
	cf.fetchingLock.Unlock()

	if ok {
		select {
		case <-fetchChan:
		case <-cf.closeChan:
			return "", "", nil
		}
		return cf.getExistingCookie(h)
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

func (cf *cloudflare) runPython(feedURL, h string) (string, string, error) {
	out, err :=
		exec.Command("python3", "-c", cookieScript, feedURL).CombinedOutput()
	str := string(out)
	if err != nil {
		cf.setInvalid(h)

		if strings.Contains(str, cloudflareBroken) ||
			strings.Contains(str, cloudflareMissing) {
			// There are probably more errors we can put here
			// But brokenCfscrape is permanent so we want to avoid setting it on
			// transient errors
			cf.failureLock.Lock()
			cf.brokenCfscrape = true
			cf.failureLock.Unlock()
			return "", "", errCloudflareBroken
		}

		glog.Error(str)
		return "", "", errCloudflareCaptcha
	}

	lines := strings.Split(string(out), "\n")

	if len(lines) < 2 {
		glog.Errorf("Missing cloudflare cookie or user agent for " + feedURL)
		cf.setInvalid(h)
		return "", "", errCloudflareBroken
	}

	cf.cookieLock.Lock()
	cf.cookies[h] = lines[0]
	cf.userAgents[h] = lines[1]
	cf.cookieLock.Unlock()

	return lines[0], lines[1], nil
}

func (cf *cloudflare) setInvalid(h string) {
	cf.failureLock.Lock()
	// Any new fetches in the next hour will be aborted
	// The retry mechanism in rssfetcher will then wait six hours for them
	// This keeps the maximum time between fetches to 7 hours
	cf.invalidUntil[h] = time.Now().Add(time.Hour)
	cf.failureLock.Unlock()
}

func (cf *cloudflare) finishFetching(h string) {
	cf.fetchingLock.Lock()
	c := cf.fetching[h]
	close(c)
	delete(cf.fetching, h)
	cf.fetchingLock.Unlock()
}
