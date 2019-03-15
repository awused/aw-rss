package webserver

import (
	"flag"
	"net/http"
	"path"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

// TODO -- this must die
var staticRoot = flag.String("static", "/usr/local/www/rss-aggregator", "Directory containing the static files used")

const staticDir = "static"
const nodeDir = "node_modules"

func (w *webserver) getRouter() http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	router.Get("/static/*", http.FileServer(http.Dir(*staticRoot)).ServeHTTP)
	router.Get("/sw.js", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, path.Join(*staticRoot, staticDir, "compiled", "sw.js"))
	})
	// TODO -- remove entirely
	router.Get("/node_modules/*", http.FileServer(http.Dir(*staticRoot)).ServeHTTP)
	router.Route("/api", w.apiRoutes)
	router.Get("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(
			w, r, path.Join(*staticRoot, staticDir, "icons", "graphicsvibe-rss-feed.ico"))
	})
	router.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, path.Join(*staticRoot, staticDir, "index.html"))
	})

	return router
}
