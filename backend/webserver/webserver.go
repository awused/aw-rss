package webserver

import (
	"flag"

	"github.com/awused/rss-aggregator/backend/database"
	"github.com/awused/rss-aggregator/backend/rssfetcher"
	//. "github.com/awused/rss-aggregator/structs"
	"net"
	"net/http"
	"sync"

	"github.com/golang/glog"
)

var protocol = flag.String("proto", "tcp", "Network protocol used, tcp, udp, or unix")
var addr = flag.String("addr", ":8080", "The address the web server listens to for connections")

type webserver struct {
	db        *database.Database
	wg        sync.WaitGroup
	listener  net.Listener
	errorChan chan error
	closed    bool
	closeLock sync.Mutex
	rss       rssfetcher.RssFetcher
}

func WebServer() (*webserver, error) {
	glog.V(5).Info("WebServer() started")

	db, err := database.GetDatabase()
	if err != nil {
		glog.Error(err)
		return nil, err
	}

	var web webserver
	web.db = db
	web.errorChan = make(chan error)

	web.rss, err = rssfetcher.NewRssFetcher()
	if err != nil {
		glog.Error(err)
		return nil, err
	}

	glog.V(5).Info("WebServer() completed")
	return &web, nil
}

func (this *webserver) Close() error {
	glog.Info("Closing webserver")

	if this.closed {
		glog.Warning("Tried to close webserver that has already been closed")
		return nil
	}
	this.closeLock.Lock()
	defer this.closeLock.Unlock()
	if this.closed {
		glog.Warning("Tried to close webserver that has already been closed")
		return nil
	}

	// Close and kill the main routine
	this.closed = true
	this.listener.Close()

	defer glog.Info("Close() completed")
	// rss.Close() also closes the database
	return this.rss.Close()
}

func (this *webserver) Run() (err error) {
	go this.runRss()

	glog.Info("Webserver.Run() started")

	this.listener, err = net.Listen(*protocol, *addr)
	if err != nil {
		glog.Errorf("Failed to open listening socket for %s on %s: %s", *protocol, *addr, err)
	}
	glog.Infof("Listening for connections on %s", *addr)

	err = http.Serve(this.listener, this.getRouter())

	if !this.closed {
		glog.Error(err)
		return err
	} else {
		glog.Info("webserver closed, exiting")
	}

	return nil
}

func (this *webserver) runRss() {
	glog.Info("Starting rssfetcher")

	err := this.rss.Run()

	if err != nil {
		glog.Error(err)
	} else {
		glog.Info("rssfetcher closed")
	}
}
