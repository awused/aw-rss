package webserver

import (
	"flag"
	"net/http"
	"os"
	"path"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

// TODO -- this must die
var staticRoot = flag.String("static", "/usr/local/www/rss-aggregator", "Directory containing the static files used")

const staticDir = "static"
const nodeDir = "node_modules"

// redirectingFileSystem is an implementation of http.FileSystem that
// redirects all 404s to an index, which is useful for client side routing
type redirectingFileSystem struct {
	dir http.Dir
	// Default path relative to the root of the directory
	// Must be inside the directory, or it will fail
	index string
}

func (rfs redirectingFileSystem) Open(name string) (http.File, error) {
	f, err := rfs.dir.Open(name)

	if os.IsNotExist(err) {
		return rfs.dir.Open(rfs.index)
	}
	return f, err
}

func (w *webserver) getRouter() http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	// TODO -- remove these two routes
	router.Get("/sw.js", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, path.Join(*staticRoot, staticDir, "compiled", "sw.js"))
	})
	router.Get("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(
			w, r, path.Join(*staticRoot, staticDir, "icons", "graphicsvibe-rss-feed.ico"))
	})

	router.Route("/api", w.apiRoutes)
	router.Get("/*", http.FileServer(
		redirectingFileSystem{
			http.Dir(*staticRoot),
			path.Join(staticDir, "index.html")}).ServeHTTP)

	return router
}
