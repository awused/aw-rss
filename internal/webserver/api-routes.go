package webserver

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/awused/aw-rss/internal/database"
	"github.com/awused/aw-rss/internal/structs"
	"github.com/go-chi/chi"
	log "github.com/sirupsen/logrus"
)

// It'd be nice to replace this with grpc but grpc-web is too much of a pain
func (ws *webserver) apiRoutes(r chi.Router) {
	r.Post("/items", ws.getItems)
	r.Post("/items/{id}/read", ws.setItemRead(true))
	r.Post("/items/{id}/unread", ws.setItemRead(false))

	r.Get("/feeds/disabled", ws.getDisabledFeeds)
	r.Post("/feeds/add", ws.addFeed)
	r.Post("/feeds/{id}/edit", ws.editFeed)
	r.Post("/feeds/{id}/read", ws.markFeedAsRead)

	r.Post("/categories/add", ws.addCategory)
	r.Post("/categories/reorder", ws.reorderCategories)
	r.Post("/categories/{id}/edit", ws.editCategory)

	r.Get("/current", ws.currentState)
	r.Get("/updates/{timestamp}", ws.updatesSince)
}

func (ws *webserver) getItems(w http.ResponseWriter, r *http.Request) {
	var req database.GetItemsRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := ws.db.GetItems(req)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err = json.NewEncoder(w).Encode(resp); err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (ws *webserver) setItemRead(readState bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			log.Error(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		nit, err := ws.db.MutateItem(int64(id), structs.ItemSetRead(readState))
		if err != nil {
			log.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err = json.NewEncoder(w).Encode(nit); err != nil {
			log.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func (ws *webserver) getDisabledFeeds(w http.ResponseWriter, r *http.Request) {
	feeds, err := ws.db.GetDisabledFeeds()
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err = json.NewEncoder(w).Encode(feeds); err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (ws *webserver) currentState(w http.ResponseWriter, r *http.Request) {
	cs, err := ws.db.GetCurrentState()
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(cs); err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (ws *webserver) updatesSince(w http.ResponseWriter, r *http.Request) {
	ut, err := strconv.ParseInt(chi.URLParam(r, "timestamp"), 10, 64)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	t := time.Unix(ut, 0).UTC()

	up, err := ws.db.GetUpdates(t)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(up); err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

type markFeedReadRequest struct {
	MaxItemID *int64 `json:"maxItemId"`
}

type markFeedReadResponse struct {
	Items []*structs.Item `json:"items"`
}

func (ws *webserver) markFeedAsRead(w http.ResponseWriter, r *http.Request) {
	feedid, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var req markFeedReadRequest

	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.MaxItemID == nil {
		http.Error(w, "maxItemID is a required field", http.StatusBadRequest)
		return
	}

	items, err := ws.db.MarkItemsReadByFeed(feedid, *req.MaxItemID)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := markFeedReadResponse{Items: items}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

type reorderCategoriesRequest struct {
	CategoryIDs []int64 `json:"categoryIds"`
}

type reorderCategoriesResponse struct {
	Categories []*structs.Category `json:"categories"`
}

func (ws *webserver) reorderCategories(w http.ResponseWriter, r *http.Request) {
	var req reorderCategoriesRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(req.CategoryIDs) == 0 {
		return
	}

	categories, err := ws.db.ReorderCategories(req.CategoryIDs)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := reorderCategoriesResponse{Categories: categories}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
