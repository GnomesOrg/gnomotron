package config

import (
	"log"
	"os"
	"strconv"
)

type Config struct {
	TOKEN             string
	APIKEY            string
	MONGO_URI         string
	MONGO_DB          string
	BOT_NAME          string
	MAX_DIALOGUE_SIZE int
}

func LoadConfig() *Config {
	cfg := Config{}
	cfg.TOKEN = os.Getenv("GNOMOTRON_TELEGRAM_TOKEN")
	cfg.APIKEY = os.Getenv("GNOMOTRON_TELEGRAM_API_KEY")
	cfg.MONGO_URI = os.Getenv("GNOMOTRON_MONGO_URI")
	cfg.MONGO_DB = os.Getenv("GNOMOTRON_MONGO_DB")
	cfg.BOT_NAME = os.Getenv("BOT_NAME")
	cfg.MAX_DIALOGUE_SIZE, _ = strconv.Atoi(os.Getenv("MAX_DIALOGUE_SIZE"))

	if cfg.TOKEN == "" || cfg.APIKEY == "" || cfg.MONGO_URI == "" || cfg.MONGO_DB == "" || cfg.BOT_NAME == "" || cfg.MAX_DIALOGUE_SIZE <= 0 {
		log.Panic("All config fields must be specified via the environment variables")
	}

	return &cfg
}
