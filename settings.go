package main

import (
	"github.com/kelseyhightower/envconfig"
)

type settings struct {
	AppName     string `envconfig:"APP_NAME" default:"self-service-template"`
	DatabaseURL string `envconfig:"DATABASE_URL"`
	Host        string `envconfig:"HOST" default:"0.0.0.0"`
	Port        int    `envconfig:"PORT" default:"8080"`
	LogLevel    string `envconfig:"LOG_LEVEL" default:"INFO"`
}

var Settings settings

func InitSettings() {
	if err := envconfig.Process("", &Settings); err != nil {
		panic("failed to load settings from env: " + err.Error())
	}
}
