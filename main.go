package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/awused/aw-rss/internal/webserver"
	"github.com/golang/glog"
)

func main() {
	flag.Parse()
	defer glog.Flush()

	server, err := webserver.NewWebServer()
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
}
