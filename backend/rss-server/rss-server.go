package main

import (
	"flag"
	"github.com/awused/rss-aggregator/backend/webserver"
	"github.com/golang/glog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	flag.Parse()
	defer glog.Flush()

	server, err := webserver.WebServer()
	if err != nil {
		panic(err)
	}
	defer server.Close()

	serverChan := make(chan struct{}, 1)
	go func() {
		if err := server.Run(); err != nil {
			glog.Error(err)
			panic(err)
		}
		glog.Info("server.Run() exited normally")
		serverChan <- struct{}{}
	}()

	/*fetcher, err := rssfetcher.RssFetcher()
	if err != nil {
		panic(err)
	}
	defer fetcher.Close()

	fetcherChan := make(chan struct{}, 1)
	go func() {
		if err := fetcher.Run(); err != nil {
			glog.Error(err)
			panic(err)
		}
		glog.Info("fetcher.Run() exited normally")
		fetcherChan <- struct{}{}
	}()*/

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGUSR1)

Loop:
	for {
		select {
		case <-serverChan:
			glog.Errorf("webserver.Run() exited unexpectedly")
			panic("webserver.Run() exited unexpectedly")
		case sig := <-sigs:
			switch sig {
			case syscall.SIGINT:
				break Loop
			case syscall.SIGUSR1:
				glog.Info("SIGUSR1")
			}
		}
	}
	signal.Reset(syscall.SIGINT)

	glog.Info("SIGINT caught, exiting")
	server.Close()
	<-serverChan
	os.Exit(0)
}
