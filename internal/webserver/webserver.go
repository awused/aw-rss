package webserver

import (
	"strconv"

	"github.com/awused/aw-rss/internal/config"
	"github.com/awused/aw-rss/internal/database"
	"github.com/awused/aw-rss/internal/rssfetcher"

	"net"
	"net/http"
	"sync"

	log "github.com/sirupsen/logrus"
)

// WebServer A web server
type WebServer interface {
	Run() error
	Close() error
}

type webserver struct {
	conf      config.Config
	db        *database.Database
	wg        sync.WaitGroup
	listener  net.Listener
	closed    bool
	closeLock sync.Mutex
	rss       rssfetcher.RssFetcher
	rssError  error
}

// NewWebServer creates a new webserver
func NewWebServer(c config.Config) (WebServer, error) {
	web := webserver{conf: c}

	db, err := database.NewDatabase(web.conf.Database)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	web.db = db

	web.rss, err = rssfetcher.NewRssFetcher(web.conf, db)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	return &web, nil
}

func (w *webserver) Close() error {
	return w.close(nil)
}

func (w *webserver) close(rssError error) error {
	if w.closed {
		log.Info("Tried to close webserver that has already been closed")
		return nil
	}
	log.Info("Closing webserver")

	w.closeLock.Lock()
	defer w.closeLock.Unlock()
	if w.closed {
		log.Warning("Tried to close webserver that has already been closed")
		return nil
	}

	// Close and kill the main routine
	w.closed = true
	w.rssError = rssError
	w.listener.Close()

	defer log.Info("Close() completed")
	// rss.Close() also closes the database
	return w.rss.Close()
}

func (w *webserver) Run() (err error) {
	go w.runRss()

	log.Info("Webserver.Run() started")

	addr := "localhost:" + strconv.Itoa(w.conf.Port)

	w.listener, err = net.Listen("tcp", addr)
	if err != nil {
		log.Errorf("Failed to open listening socket on %s: %s", addr, err)
	}
	log.Infof("Listening for connections on %s", addr)

	err = http.Serve(w.listener, w.getRouter())

	w.closeLock.Lock()
	defer w.closeLock.Unlock()
	if !w.closed {
		log.Error(err)
		return err
	}
	log.Info("webserver closed, exiting")
	return w.rssError
}

func (w *webserver) runRss() {
	log.Info("Starting rssfetcher")

	err := w.rss.Run()

	if err != nil {
		log.Error(err)
		w.close(err)
	} else {
		log.Info("rssfetcher closed")
	}
}
