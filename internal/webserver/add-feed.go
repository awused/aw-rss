package webserver

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/antchfx/htmlquery"
	"github.com/awused/aw-rss/internal/rssfetcher"
	"github.com/awused/aw-rss/internal/structs"
	"github.com/mmcdole/gofeed"
	log "github.com/sirupsen/logrus"
)

type addFeedRequest struct {
	URL       string `json:"url"`
	UserTitle string `json:"title"`

	// When the user wants to force a feed through
	// Even if it doesn't appear to be valid
	Force bool `json:"force"`
}

type addFeedResponse struct {
	// "success" and a feed or "candidates" and some candidates
	// "invalid" if it wasn't a valid feed and no candidates were found
	Status string `json:"status"`

	// When the URL provided isn't a feed but feeds were detected at that site
	// User must pick one
	Candidates []string `json:"candidates,omitempty"`

	Feed *structs.Feed `json:"feed,omitempty"`
}

func (ws *webserver) addFeed(w http.ResponseWriter, r *http.Request) {
	var req addFeedRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	rawURL := unconditionalURLRewrite(req.URL)

	u, err := url.Parse(rawURL)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		log.Error(err)
		http.Error(w, "url scheme must be http or https", http.StatusBadRequest)
		return
	}

	if !req.Force {
		req, err := http.NewRequest("GET", rawURL, nil)
		if err != nil {
			log.Error(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Workaround for dolphinemu.org, but doesn't seem to break any other feeds.
		req.Header.Add("Cache-Control", "no-cache")

		// Pretend to be wget. Some sites don't like an empty user agent.
		// Reddit in particular will _always_ say to retry in a few seconds,
		// even if you wait hours.
		req.Header.Add("User-Agent", "Wget/1.19.5 (freebsd11.1)")

		httpClient := &http.Client{
			Timeout: rssfetcher.RssTimeout,
		}

		resp, err := httpClient.Do(req)

		if err != nil {
			log.Error(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		bodyBytes, err := ioutil.ReadAll(resp.Body)
		// Close unconditionally to avoid memory leaks
		_ = resp.Body.Close()
		if err != nil {
			log.Error(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		body := string(bodyBytes)
		// TODO -- quirks.HandleBodyQuirks()
		_, err = gofeed.NewParser().ParseString(body)
		if err == nil {
			log.Infoln("Successfully parsed new feed: ", rawURL)
		} else {
			log.Infoln("Attempting to parse: ", rawURL, "as HTML")

			parsed, err := htmlquery.Parse(strings.NewReader(body))
			if err != nil {
				log.Error(err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			nodes := htmlquery.Find(parsed, "(//head/link|//body/link)[@type='application/rss+xml' or @type='application/atom+xml']")
			if len(nodes) == 1 {
				rawURL = htmlquery.SelectAttr(nodes[0], "href")
				log.Info("Found feed URL in HTML: ", rawURL)
			} else if len(nodes) > 1 {
				log.Error("Support for selecting between multiple detected feeds is unimplemented")
				http.Error(w, "Unimplemented", http.StatusBadRequest)
				return
			} else {
				log.Error("No feeds found for ", rawURL)
				log.Error(body)
				http.Error(w, "No feed found", http.StatusBadRequest)
				return
			}

			rawURL = unconditionalURLRewrite(rawURL)

			u, err := url.Parse(rawURL)
			if err != nil {
				log.Error(err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			if u.Scheme != "http" && u.Scheme != "https" {
				log.Error(err)
				http.Error(w, "url scheme must be http or https", http.StatusBadRequest)
				return
			}
		}
	}

	f, err := ws.db.InsertNewFeed(rawURL, req.UserTitle)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := addFeedResponse{Status: "success", Feed: f}

	if err = json.NewEncoder(w).Encode(resp); err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	ws.rss.InformFeedChanged()
	return
}

const youtubeChannelRE = `^https://www.youtube.com/channel/(UC[a-zA-Z0-9_-]+)`
const youtubeFeedPrefix = "https://www.youtube.com/feeds/videos.xml?channel_id="

// Unconditionally drop parameters mangadex feeds to prevent duplicates.
// As of 2020-08 the only parameter is h and not specifying it is, at least
// currently, equivalent to h=1.
const mangadexRE = `^https://mangadex.org/([^?])+`

const yandereRE = `^https://yande.re/post\?(.*&)?tags=([^?&]+)`
const yanderePrefix = "https://yande.re/post/atom?tags="

var youtubeChannelRegex = regexp.MustCompile(youtubeChannelRE)
var mangadexRegex = regexp.MustCompile(mangadexRE)
var yandereRegex = regexp.MustCompile(yandereRE)

// Responsible for URL rewrites that are always performed.
// These cannot be overwritten with Force so should be very limited.
func unconditionalURLRewrite(url string) string {
	url = strings.TrimSpace(url)

	matches := youtubeChannelRegex.FindStringSubmatch(url)
	if matches != nil {
		return youtubeFeedPrefix + matches[1]
	}

	matches = mangadexRegex.FindStringSubmatch(url)
	if matches != nil {
		return matches[0]
	}

	matches = yandereRegex.FindStringSubmatch(url)
	if matches != nil {
		return yanderePrefix + strings.TrimRight(matches[2], "+")
	}

	return url
}
