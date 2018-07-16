package main

import (
	"flag"
	"github.com/awused/rss-aggregator/backend/rssfetcher"
	"github.com/golang/glog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Disable logging to files when running the standalone fetcher
	flag.Set("logtostderr", "true")
	flag.Set("alsologtostderr", "false")
	flag.Parse()
	defer glog.Flush()

	fetcher, err := rssfetcher.NewRssFetcher()
	if err != nil {
		panic(err)
	}
	defer fetcher.Close()

	c := make(chan struct{}, 1)
	go func() {
		if err := fetcher.Run(); err != nil {
			panic(err)
		}
		glog.Info("fetcher.Run() exited normally")
		c <- struct{}{}
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGUSR1)

Loop:
	for {
		switch <-sigs {
		case syscall.SIGINT:
			break Loop
		case syscall.SIGUSR1:
			glog.Info("SIGUSR1")
		}
	}
	signal.Reset(syscall.SIGINT)

	glog.Info("SIGINT caught, exiting")
	fetcher.Close()
	<-c
	os.Exit(0)
}
