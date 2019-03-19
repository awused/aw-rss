package webserver

import (
	"flag"
	"net/http"
	"os"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

// TODO -- this must die
var staticRoot = flag.String("static", "/usr/local/www/rss-aggregator", "Directory containing the static files used")

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

	router.Route("/api", w.apiRoutes)
	router.Get("/*", http.FileServer(
		redirectingFileSystem{
			http.Dir(*staticRoot),
			"index.html"}).ServeHTTP)

	return router
}
