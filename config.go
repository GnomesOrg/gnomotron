package main

import (
	"log"
	"os"
)

type Config struct {
	TOKEN  string
	CHATID string
	APIKEY string
}

func LoadConfig() *Config {
	cfg := Config{}
	cfg.TOKEN = os.Getenv("GNOMOTRON_TELEGRAM_TOKEN")
	cfg.CHATID = os.Getenv("GNOMOTRON_TELEGRAM_CHAT_ID")
	cfg.APIKEY = os.Getenv("GNOMOTRON_TELEGRAM_API_KEY")

	if cfg.TOKEN == "" || cfg.CHATID == "" || cfg.APIKEY == "" {
		log.Panic("TOKEN, CHAT_ID, and API_KEY must be specified via the environment variables")
	}

	return &cfg
}
