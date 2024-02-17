package webserver

import (
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	log "github.com/sirupsen/logrus"
)

// redirectingFileSystem is an implementation of http.FileSystem that
// redirects all 404s to an index, which is useful for client side routing
type redirectingFileSystem struct {
	dir http.FileSystem
	// Default path relative to the root of the directory
	// Must be inside the directory, or it will fail
	index string
}

func (rfs redirectingFileSystem) Open(name string) (http.File, error) {
	f, err := rfs.dir.Open(name)

	if err != nil {
		return rfs.dir.Open(rfs.index)
	}
	return f, err
}

func (w *webserver) getRouter(dist fs.FS) http.Handler {
	middleware.DefaultLogger = middleware.RequestLogger(
		&middleware.DefaultLogFormatter{
			Logger:  log.StandardLogger(),
			NoColor: false,
		})

	router := chi.NewRouter()
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	router.Route("/api", w.apiRoutes)
	router.Get("/*", http.FileServer(
		redirectingFileSystem{
			dir:   http.FS(dist),
			index: "index.html",
		}).ServeHTTP)

	return router
}
