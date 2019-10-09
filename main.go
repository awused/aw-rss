package main

import (
	"os"
	"os/signal"
	"path"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/awused/aw-rss/internal/config"
	"github.com/awused/aw-rss/internal/webserver"
	log "github.com/sirupsen/logrus"
)

func main() {
	conf, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	file := initLogger(conf)
	if file != nil {
		defer file.Close()
	}

	server, err := webserver.NewWebServer(conf)
	if err != nil {
		log.Fatal(err)
	}

	serverChan := make(chan error)
	go func() {
		if err := server.Run(); err != nil {
			log.Error(err)
			serverChan <- err
		} else {
			log.Info("server.Run() exited normally")
		}
		close(serverChan)
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

Loop:
	for {
		select {
		case err = <-serverChan:
			if err != nil {
				log.Panicf("webserver.Run() exited unexpectedly with [%v]", err)
			}
			log.Panicf("webserver.Run() exited unexpectedly")
		case sig := <-sigs:
			switch sig {
			case syscall.SIGTERM:
				break Loop
			case syscall.SIGINT:
				break Loop
			}
		}
	}
	signal.Reset(syscall.SIGINT, syscall.SIGTERM)

	log.Info("SIGINT caught, exiting")
	server.Close()
	<-serverChan
}

func initLogger(conf config.Config) *os.File {
	// Slow, but not significant
	log.SetReportCaller(true)

	log.SetFormatter(&log.TextFormatter{
		ForceColors:   true,
		FullTimestamp: true,
		// The SetReportCaller option was clearly written by someone who resented it.
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			_, filename := path.Split(f.File)
			if filename == "logger.go" {
				return "", ""
			}

			s := strings.Split(f.Function, ".")
			funcname := s[len(s)-1]
			filename = filename + ":" + strconv.Itoa(f.Line)
			return funcname + "()", filename
		},
	})

	var file *os.File
	var err error
	if conf.LogFile != "" {
		// Don't persist logs between sessions, they're not useful
		file, err = os.OpenFile(
			conf.LogFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)

		if err != nil {
			log.Fatal(err)
		}
		log.SetOutput(file)
	}

	lvl, err := log.ParseLevel(conf.LogLevel)
	if err != nil {
		log.Fatal(err)
	}
	log.SetLevel(lvl)

	return file
}
