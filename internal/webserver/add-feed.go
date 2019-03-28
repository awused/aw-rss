package webserver

import (
	"encoding/json"
	"net/http"
	"net/url"

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

	u, err := url.Parse(req.URL)
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
		f, err := ws.db.InsertNewFeed(req.URL, req.UserTitle)
		if err != nil {
			log.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		resp := addFeedResponse{Status: "success", Feed: f}

		if err = json.NewEncoder(w).Encode(resp); err != nil {
			log.Error(err)
		}
		ws.rss.InformFeedChanged()
		return
	}

	http.Error(w, "Unimplemented", http.StatusBadRequest)
	return
}
