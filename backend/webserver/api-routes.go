package webserver

import (
	"encoding/json"
	"github.com/golang/glog"
	"github.com/zenazn/goji/web"
	"net/http"
	"strconv"
)

func (this *webserver) getApiRouter() http.Handler {
	apiRouter := web.New()
	apiRouter.Get("/feeds/list", this.listFeeds)

	apiRouter.Get("/items/list", this.listItems)
	apiRouter.Post("/items/:id/read", this.markItemAsRead)
	apiRouter.Post("/items/:id/unread", this.markItemAsUnread)

	apiRouter.Compile()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		glog.V(3).Infof("Handling API route %s:%s", r.Method, r.URL.Path)
		apiRouter.ServeHTTP(w, r)
	})
}

/**
 * disabled = 1 to include disabled feeds
 */
func (this *webserver) listFeeds(c web.C, w http.ResponseWriter, r *http.Request) {
	glog.V(5).Infof("listFeeds() started")
	q := r.URL.Query()

	feeds, err := this.db.GetFeeds(q.Get("disabled") == "1")
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
func (this *webserver) listItems(c web.C, w http.ResponseWriter, r *http.Request) {
	glog.V(5).Infof("listItems() started")
	q := r.URL.Query()

	items, err := this.db.GetItems(q.Get("read") == "1")
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

func (this *webserver) markItemAsRead(c web.C, w http.ResponseWriter, r *http.Request) {
	glog.V(5).Infof("markItemAsRead() started")

	id, err := strconv.Atoi(c.URLParams["id"])
	if err != nil {
		glog.Error(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	it, err := this.db.GetItem(int64(id))
	if err != nil {
		glog.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if it == nil {
		glog.Infof("Tried to mark non-existent item %d as read", id)
		if err = json.NewEncoder(w).Encode(struct{
					Error string `json:"error"`
				} {
					Error: "No such item",
				}); err != nil {
			glog.Error(err)
		}
		return
	}

	it.Read = true
	err = this.db.UpdateItem(it)
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

func (this *webserver) markItemAsUnread(c web.C, w http.ResponseWriter, r *http.Request) {
	glog.V(5).Infof("markItemAsUnread() started")

	id, err := strconv.Atoi(c.URLParams["id"])
	if err != nil {
		glog.Error(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	it, err := this.db.GetItem(int64(id))
	if err != nil {
		glog.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if it == nil {
		glog.Infof("Tried to mark non-existent item %d as unread", id)
		if err = json.NewEncoder(w).Encode(struct{
					Error string `json:"error"`
				} {
					Error: "No such item",
				}); err != nil {
			glog.Error(err)
		}
		return
	}

	it.Read = false
	err = this.db.UpdateItem(it)
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
