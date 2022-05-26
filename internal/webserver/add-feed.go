package webserver

import (
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/awused/aw-rss/internal/structs"
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

	if req.Force {
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

	http.Error(w, "Unimplemented", http.StatusBadRequest)
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
