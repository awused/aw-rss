package webserver

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi"
	"github.com/golang/glog"
)

func (w *webserver) apiRoutes(r chi.Router) {
	r.Get("/feeds/list", w.listFeeds)

	r.Get("/items/list", w.listItems)
	r.Post("/items/{id}/read", w.markItemAsRead)
	r.Post("/items/{id}/unread", w.markItemAsUnread)

	r.Get("/current", w.currentState)
	r.Get("/updates/{timestamp}", w.updatesSince)
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

	items, err := ws.db.GetItems(q.Get("read") == "1")
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

// TODO -- Rewrite these around MutateItem and atomic read->write->read again transactions
func (ws *webserver) markItemAsRead(w http.ResponseWriter, r *http.Request) {
	glog.V(5).Infof("markItemAsRead() started")

	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		glog.Error(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	it, err := ws.db.GetItem(int64(id))
	if err != nil {
		glog.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if it == nil {
		glog.Infof("Tried to mark non-existent item %d as read", id)
		if err = json.NewEncoder(w).Encode(struct {
			Error string `json:"error"`
		}{
			Error: "No such item",
		}); err != nil {
			glog.Error(err)
		}
		return
	}

	it.Read = true
	err = ws.db.UpdateItem(it)
	if err != nil {
		glog.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	glog.V(3).Infof("markItemAsRead() completed for item [%s]", it)
	if err = json.NewEncoder(w).Encode(it); err != nil {
		glog.Error(err)
	}
}

func (ws *webserver) markItemAsUnread(w http.ResponseWriter, r *http.Request) {
	glog.V(5).Infof("markItemAsUnread() started")

	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		glog.Error(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	it, err := ws.db.GetItem(int64(id))
	if err != nil {
		glog.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if it == nil {
		glog.Infof("Tried to mark non-existent item %d as unread", id)
		if err = json.NewEncoder(w).Encode(struct {
			Error string `json:"error"`
		}{
			Error: "No such item",
		}); err != nil {
			glog.Error(err)
		}
		return
	}

	it.Read = false
	err = ws.db.UpdateItem(it)
	if err != nil {
		glog.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	glog.V(3).Infof("markItemAsUnread() completed for item [%s]", it)
	if err = json.NewEncoder(w).Encode(it); err != nil {
		glog.Error(err)
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
