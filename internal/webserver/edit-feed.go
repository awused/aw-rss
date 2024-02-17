package webserver

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/awused/aw-rss/internal/structs"
	"github.com/go-chi/chi/v5"
	log "github.com/sirupsen/logrus"
)

type editFeedRequest struct {
	Edit structs.FeedEdit `json:"edit"`
}

func (ws *webserver) editFeed(w http.ResponseWriter, r *http.Request) {
	var req editFeedRequest

	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	f, err := ws.db.MutateFeed(int64(id), structs.ApplyFeedEdit(req.Edit))
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
