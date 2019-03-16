package webserver

import (
	"flag"

	"github.com/awused/aw-rss/internal/database"
	"github.com/awused/aw-rss/internal/rssfetcher"

	"net"
	"net/http"
	"sync"

	"github.com/golang/glog"
)

var protocol = flag.String("proto", "tcp", "Network protocol used, tcp, udp, or unix")
var addr = flag.String("addr", "localhost:8080", "The address the web server listens to for connections")

// WebServer A web server
type WebServer interface {
	Run() error
	Close() error
}

type webserver struct {
	db        *database.Database
	wg        sync.WaitGroup
	listener  net.Listener
	closed    bool
	closeLock sync.Mutex
	rss       rssfetcher.RssFetcher
	rssError  error
}

// NewWebServer creates a new webserver
func NewWebServer() (WebServer, error) {
	glog.V(5).Info("WebServer() started")

	db, err := database.GetDatabase()
	if err != nil {
		glog.Error(err)
		return nil, err
	}

	var web webserver
	web.db = db

	web.rss, err = rssfetcher.NewRssFetcher()
	if err != nil {
		glog.Error(err)
		return nil, err
	}

	glog.V(5).Info("WebServer() completed")
	return &web, nil
}

func (w *webserver) Close() error {
	return w.close(nil)
}

func (w *webserver) close(rssError error) error {
	glog.Info("Closing webserver")
	if rssError != nil {
	}

	if w.closed {
		glog.Warning("Tried to close webserver that has already been closed")
		return nil
	}
	w.closeLock.Lock()
	defer w.closeLock.Unlock()
	if w.closed {
		glog.Warning("Tried to close webserver that has already been closed")
		return nil
	}

	// Close and kill the main routine
	w.closed = true
	w.rssError = rssError
	w.listener.Close()

	defer glog.Info("Close() completed")
	// rss.Close() also closes the database
	return w.rss.Close()
}

func (w *webserver) Run() (err error) {
	go w.runRss()

	glog.Info("Webserver.Run() started")

	w.listener, err = net.Listen(*protocol, *addr)
	if err != nil {
		glog.Errorf("Failed to open listening socket for %s on %s: %s", *protocol, *addr, err)
	}
	glog.Infof("Listening for connections on %s", *addr)

	err = http.Serve(w.listener, w.getRouter())

	w.closeLock.Lock()
	defer w.closeLock.Unlock()
	if !w.closed {
		glog.Error(err)
		return err
	}
	glog.Info("webserver closed, exiting")
	return w.rssError
}

func (w *webserver) runRss() {
	glog.Info("Starting rssfetcher")

	err := w.rss.Run()

	if err != nil {
		glog.Error(err)
		w.close(err)
	} else {
		glog.Info("rssfetcher closed")
	}
}
