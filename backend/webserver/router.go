package webserver

import (
	"flag"
	"net/http"
	"path"

	"github.com/golang/glog"
	"github.com/zenazn/goji/web"
)

var staticRoot = flag.String("static", "/usr/local/www/rss-aggregator", "Directory containing the static files used")

const staticDir = "static"
const nodeDir = "node_modules"

func (this *webserver) getRouter() http.Handler {
	router := web.New()
	router.Get("/static/*", http.FileServer(http.Dir(*staticRoot)))
	router.Get("/sw.js", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, path.Join(*staticRoot, staticDir, "compiled", "sw.js"))
	})
	router.Get("/node_modules/*", http.FileServer(http.Dir(*staticRoot)))
	router.Get("/dev", http.RedirectHandler("/dev/", 301))
	router.Get("/dev/*", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, path.Join(*staticRoot, staticDir, "dev.html"))
	})
	router.Handle("/api/*", http.StripPrefix("/api", this.getApiRouter()))
	router.Get("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(
			w, r, path.Join(*staticRoot, staticDir, "icons", "graphicsvibe-rss-feed.ico"))
	})
	router.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, path.Join(*staticRoot, staticDir, "index.html"))
	})

	router.Compile()

	// Wrap middleware around the router
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		glog.V(3).Infof("Handling route %s:%s", r.Method, r.URL.Path)

		router.ServeHTTP(w, r)
	})
}
