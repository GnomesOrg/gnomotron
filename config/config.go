package config

import (
	"log"
	"os"
)

type Config struct {
	TOKEN     string
	CHAT_ID   string
	APIKEY    string
	MONGO_URI string
	MONGO_DB  string
}

func LoadConfig() *Config {
	cfg := Config{}
	cfg.TOKEN = os.Getenv("GNOMOTRON_TELEGRAM_TOKEN")
	cfg.CHAT_ID = os.Getenv("GNOMOTRON_TELEGRAM_CHAT_ID")
	cfg.APIKEY = os.Getenv("GNOMOTRON_TELEGRAM_API_KEY")
	cfg.MONGO_URI = os.Getenv("GNOMOTRON_MONGO_URI")
	cfg.MONGO_DB = os.Getenv("GNOMOTRON_MONGO_DB")

	if cfg.TOKEN == "" || cfg.CHAT_ID == "" || cfg.APIKEY == "" || cfg.MONGO_URI == "" || cfg.MONGO_DB == "" {
		log.Panic("All cfg fields must be specified via the environment variables")
	}

	return &cfg
}