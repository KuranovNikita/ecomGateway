package config

import (
	"log"
	"os"
)

type Config struct {
	Env string
}

func MustLoad() *Config {
	env := os.Getenv("ENV")
	if env == "" {
		log.Fatal("ENV is not set")
	}

	var cfg Config

	cfg.Env = env

	return &cfg
}
