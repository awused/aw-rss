package webserver

import (
	"encoding/json"
	"net/http"

	"github.com/awused/aw-rss/internal/structs"
	log "github.com/sirupsen/logrus"
)

type editFeedRequest struct {
	ID   int64            `json:"id"`
	Edit structs.FeedEdit `json:"edit"`
}

func (ws *webserver) editFeed(w http.ResponseWriter, r *http.Request) {
	var req editFeedRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	f, err := ws.db.MutateFeed(req.ID, structs.ApplyFeedEdit(req.Edit))
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err = json.NewEncoder(w).Encode(f); err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	ws.rss.InformFeedChanged()
}
