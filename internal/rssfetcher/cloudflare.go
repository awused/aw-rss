package rssfetcher

// TODO -- move to its own module

import (
	"errors"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/awused/aw-rss/internal/config"
	log "github.com/sirupsen/logrus"
)

const cookieScript = `
import cloudscraper
import sys

c, ua = cloudscraper.get_cookie_string(sys.argv[1])
print(c)
print(ua)
`

const fetchScript = `
import cloudscraper
import sys

print(cloudscraper.create_scraper().get(sys.argv[1]).text)
`

// This needs to be configureable by the user
// Don't run arbitrary JS from untrusted domains, only handle problematic sites
// as they're identified
var trustedHosts = map[string]bool{}

var (
	errUntrustedHost        = errors.New("Host not trusted for cloudflare bypass")
	errInsecureTransport    = errors.New("Cloudflare bypass requires https")
	errCloudflareCaptcha    = errors.New("Cloudflare bypass cannot handle the captcha challenges")
	errCloudflareBroken     = errors.New("Cfscrape is missing or out of date")
	errCloudflareBadGateway = errors.New("Bad gateway error from cloudflare")
	errCloudflareCooldown   = errors.New("Cloudflare bypass previously failed, waiting up to 6 hours")
	errNoCookies            = errors.New("Another thread failed to fetch cloudflare cookies")
)

func isCloudflareError(err error) bool {
	return err == errUntrustedHost ||
		err == errInsecureTransport ||
		err == errCloudflareCaptcha ||
		err == errCloudflareBroken ||
		err == errCloudflareBadGateway ||
		err == errCloudflareCooldown ||
		err == errNoCookies
}

const cloudflareSentinelOne = "<title>Attention Required! | Cloudflare</title>"
const cloudflareSentinelTwo = "<title>Just a moment...</title>"
const cloudflareSentinelThree = "<title>Please Wait... | Cloudflare</title>"
const cloudflareBadGateway = "502: Bad gateway</title>"
const cloudflareGatewayTimeout = "504: Gateway time-out</title>"
const cloudflareServerDown = "521: Web server is down</title>"
const cloudflareSSLFailure = "525: SSL handshake failed</title>"
const cloudflareNormal = "Checking if the site connection is secure"
const cloudflareMissing = "ModuleNotFoundError: No module named 'cfscrape'"
const cloudflareBroken = "Cloudflare may have changed their technique," +
	"or there may be a bug in the script."

type cloudflare struct {
	cookies    map[string]string
	userAgents map[string]string
	lastFetch  map[string]time.Time
	cookieLock sync.RWMutex
	pythonLock sync.Mutex
	closeChan  <-chan struct{}

	brokenCfscrape bool
	// If we have permanent failures, disable these hosts for 6 hours at a time
	invalidUntil map[string]time.Time
	failureLock  sync.Mutex
}

func newCloudflare(conf config.Config, closeChan <-chan struct{}) *cloudflare {
	for _, v := range conf.CloudflareDomains {
		trustedHosts[v] = true
	}

	return &cloudflare{
		cookies:      make(map[string]string),
		userAgents:   make(map[string]string),
		lastFetch:    make(map[string]time.Time),
		closeChan:    closeChan,
		invalidUntil: make(map[string]time.Time),
	}
}

func (cf *cloudflare) isCloudflareResponse(feedURL string, body string) (bool, error) {
	if len(body) < 500 {
		return false, nil
	}

	if strings.Contains(body[0:499], cloudflareBadGateway) ||
		strings.Contains(body[0:499], cloudflareGatewayTimeout) ||
		strings.Contains(body[0:499], cloudflareServerDown) ||
		strings.Contains(body[0:499], cloudflareSSLFailure) {
		h, _, err := host(feedURL)
		if err == nil {
			cf.setInvalid(h)
		}
		return true, errCloudflareBadGateway
	}

	if !strings.Contains(body[0:499], cloudflareSentinelOne) &&
		!strings.Contains(body[0:499], cloudflareSentinelTwo) &&
		!strings.Contains(body[0:499], cloudflareSentinelThree) {
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
	cookie string, userAgent string, permanentFailure error) {
	h, scheme, err := host(feedURL)
	if err != nil {
		return "", "", nil
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
		return "", "", nil
	}

	c, ua, err := cf.getExistingCookie(h)
	if err == errNoCookies {
		err = nil
	}
	return c, ua, err
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
			return "", "", errInsecureTransport
		}
		return "", "", errUntrustedHost
	}

	cf.pythonLock.Lock()
	defer cf.pythonLock.Unlock()
	select {
	case <-cf.closeChan:
		return "", "", nil
	default:
	}

	log.Infof("Fetching new cloudflare cookie for [%s]", h)
	return cf.runPython(feedURL, h)
}

func (cf *cloudflare) runPython(feedURL, h string) (string, string, error) {
	lastFetch, ok := cf.lastFetch[h]
	if ok && time.Now().Before(lastFetch.Add(time.Minute*10)) {
		return "", "", errCloudflareBroken
	}

	cf.lastFetch[h] = time.Now()

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
			// TODO -- Instead of permanent failures, try again after an even longer
			// time period
			cf.failureLock.Lock()
			cf.brokenCfscrape = true
			cf.failureLock.Unlock()
			return "", "", errCloudflareBroken
		}

		log.Error(str)
		return "", "", errCloudflareCaptcha
	}

	lines := strings.Split(str, "\n")

	if len(lines) < 2 {
		log.Errorf("Missing cloudflare cookie or user agent for " + feedURL)
		cf.setInvalid(h)
		return "", "", errCloudflareBroken
	}

	cf.cookieLock.Lock()
	cf.cookies[h] = lines[0]
	cf.userAgents[h] = lines[1]
	cf.cookieLock.Unlock()

	return lines[0], lines[1], nil
}

func (cf *cloudflare) getFeedContents(feedURL string) (string, error) {
	h, scheme, err := host(feedURL)
	if err != nil {
		return "", err
	}

	cf.failureLock.Lock()
	inv := cf.invalidUntil[h]
	broken := cf.brokenCfscrape
	cf.failureLock.Unlock()

	if time.Now().Before(inv) {
		if broken {
			return "", errCloudflareBroken
		}
		return "", errCloudflareCooldown
	}

	if scheme != "https" || !trustedHosts[h] {
		cf.setInvalid(h)
		if scheme != "https" {
			return "", errInsecureTransport
		}
		return "", errUntrustedHost
	}

	cf.pythonLock.Lock()
	defer cf.pythonLock.Unlock()
	select {
	case <-cf.closeChan:
		return "", nil
	default:
	}

	log.Infof("Fetching cloudflare contents for [%s]", h)
	return cf.fetchPython(feedURL, h)
}

func (cf *cloudflare) fetchPython(feedURL, h string) (string, error) {
	cf.lastFetch[h] = time.Now()

	out, err :=
		exec.Command("python3", "-c", fetchScript, feedURL).CombinedOutput()
	str := string(out)
	if err != nil {
		cf.setInvalid(h)

		if strings.Contains(str, cloudflareBroken) ||
			strings.Contains(str, cloudflareMissing) {
			// There are probably more errors we can put here
			// But brokenCfscrape is permanent so we want to avoid setting it on
			// transient errors
			// TODO -- Instead of permanent failures, try again after an even longer
			// time period
			cf.failureLock.Lock()
			cf.brokenCfscrape = true
			cf.failureLock.Unlock()
			return "", errCloudflareBroken
		}

		log.Error(str)
		return "", errCloudflareCaptcha
	}

	return str, nil
}

func (cf *cloudflare) setInvalid(h string) {
	cf.failureLock.Lock()
	// Any new fetches in the next hour will be aborted
	// The retry mechanism in rssfetcher will then wait six hours for them
	// This keeps the maximum time between fetches to 7 hours
	cf.invalidUntil[h] = time.Now().Add(time.Hour)
	cf.failureLock.Unlock()
}
