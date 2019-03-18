package webserver

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/awused/aw-rss/internal/structs"
	"github.com/go-chi/chi"
	"github.com/golang/glog"
)

func (ws *webserver) apiRoutes(r chi.Router) {
	r.Get("/feeds/list", ws.listFeeds)

	r.Get("/items/list", ws.listItems)
	//r.Post("/items/batch", ws.getBatchItems)
	r.Post("/items/{id}/read", ws.setItemRead(true))
	r.Post("/items/{id}/unread", ws.setItemRead(false))

	r.Get("/current", ws.currentState)
	r.Get("/updates/{timestamp}", ws.updatesSince)
}

/**
 * disabled = 1 to include disabled feeds
 */
func (ws *webserver) listFeeds(w http.ResponseWriter, r *http.Request) {
	glog.V(5).Infof("listFeeds() started")
	q := r.URL.Query()

	feeds, err := ws.db.GetFeeds(q.Get("disabled") == "1")
	if err != nil {
		glog.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	glog.V(3).Infof("Writing %d feeds to response; disabled = %t", len(feeds), q.Get("disabled") == "1")
	if err = json.NewEncoder(w).Encode(feeds); err != nil {
		glog.Error(err)
	}
}

/**
 * read = 1 to include read items
 */
func (ws *webserver) listItems(w http.ResponseWriter, r *http.Request) {
	glog.V(5).Infof("listItems() started")
	q := r.URL.Query()

	items, err := ws.db.GetItemsLegacy(q.Get("read") == "1")
	if err != nil {
		glog.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	glog.V(3).Infof("Writing %d items to response; read = %t", len(items), q.Get("read") == "1")
	if err = json.NewEncoder(w).Encode(items); err != nil {
		glog.Error(err)
	}
}

func (ws *webserver) setItemRead(readState bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			glog.Error(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		nit, err := ws.db.MutateItem(int64(id), structs.ItemSetRead(readState))
		if err != nil {
			glog.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err = json.NewEncoder(w).Encode(nit); err != nil {
			glog.Error(err)
		}
	}
}

func (ws *webserver) currentState(w http.ResponseWriter, r *http.Request) {
	cs, err := ws.db.GetCurrentState()
	if err != nil {
		glog.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(cs); err != nil {
		glog.Error(err)
	}
}

func (ws *webserver) updatesSince(w http.ResponseWriter, r *http.Request) {
	ut, err := strconv.ParseInt(chi.URLParam(r, "timestamp"), 10, 64)
	if err != nil {
		glog.Error(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	t := time.Unix(ut, 0).UTC()

	up, err := ws.db.GetUpdates(t)
	if err != nil {
		glog.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(up); err != nil {
		glog.Error(err)
	}
}
