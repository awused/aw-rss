package config

import (
	"flag"

	"github.com/awused/awconf"
)

// Config is the internal representation of the config
type Config struct {
	Database          string
	Port              int
	Host              string
	LogFile           string
	LogLevel          string
	Dedupe            bool
	CloudflareDomains []string
}

// LoadConfig loads the config using awconf
func LoadConfig() (Config, error) {
	flag.Parse()
	var conf Config

	err := awconf.LoadConfig("aw-rss", &conf)
	return conf, err
}
