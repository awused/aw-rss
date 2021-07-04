package webserver

import (
	"errors"
	"io/fs"
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
	Run(dist fs.FS) error
	Close() error
}

type webserver struct {
	conf      config.Config
	db        *database.Database
	wg        sync.WaitGroup
	listener  net.Listener
	closed    bool
	closeLock sync.Mutex
	running   bool
	rss       rssfetcher.RssFetcher
	rssError  error
}

// NewWebServer creates a new webserver
func NewWebServer(c config.Config) (WebServer, error) {
	web := webserver{conf: c}

	db, err := database.NewDatabase(web.conf)
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
	w.closeLock.Lock()
	defer w.closeLock.Unlock()
	if w.closed {
		log.Warning("Tried to close webserver that has already been closed")
		return nil
	}
	log.Info("Closing webserver")

	// Close and kill the main routine
	w.closed = true
	defer log.Info("Close() completed")

	if !w.running {
		log.Info("Close() called before Run()")
	}
	w.rssError = rssError
	w.listener.Close()

	// rss.Close() also closes the database
	return w.rss.Close()
}

func (w *webserver) Run(dist fs.FS) (err error) {
	log.Info("Webserver.Run() started")

	host := "localhost"
	if w.conf.Host != "" {
		host = w.conf.Host
	}
	addr := host + ":" + strconv.Itoa(w.conf.Port)

	w.closeLock.Lock()
	if w.closed {
		log.Errorf("Tried to run webserver that has already been closed")
		w.closeLock.Unlock()
		return errors.New("Closed")
	}

	go w.runRss()
	w.running = true
	w.listener, err = net.Listen("tcp", addr)
	w.closeLock.Unlock()
	if err != nil {
		log.Errorf("Failed to open listening socket on %s: %s", addr, err)
		return err
	}
	log.Infof("Listening for connections on %s", addr)

	err = http.Serve(w.listener, w.getRouter(dist))

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
