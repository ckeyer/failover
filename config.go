package main

import (
	"encoding/json"
	"os"
)

type Config struct {
	ControllerURL string
	ApiKey        string
	Monitor       map[string][]*Checker
}

func LoadConfig(confile string) *Config {
	f, err := os.Open(confile)
	if err != nil {
		log.Fatalf("can not open config file %s, error: %s", confile, err.Error())
	}

	conf := new(Config)
	err = json.NewDecoder(f).Decode(conf)
	if err != nil {
		log.Fatalf("can not decode config file, error %s", err.Error())
	}

	return conf
}
