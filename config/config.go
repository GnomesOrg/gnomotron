package config

import (
	"log"
	"os"
	"strconv"
	"time"
)

type Config struct {
	TOKEN                   string
	CHAT_ID                 string
	APIKEY                  string
	MONGO_URI               string
	MONGO_DB                string
	REM_FETCH_SLEEPTIME_MIN time.Duration
	REM_CHECK_SLEEPTIME_MIN time.Duration
	BOT_NAME					string
}

func LoadConfig() *Config {
	cfg := Config{}
	cfg.TOKEN = os.Getenv("GNOMOTRON_TELEGRAM_TOKEN")
	cfg.CHAT_ID = os.Getenv("GNOMOTRON_TELEGRAM_CHAT_ID")
	cfg.APIKEY = os.Getenv("GNOMOTRON_TELEGRAM_API_KEY")
	cfg.MONGO_URI = os.Getenv("GNOMOTRON_MONGO_URI")
	cfg.MONGO_DB = os.Getenv("GNOMOTRON_MONGO_DB")
	cfg.BOT_NAME = os.Getenv("BOT_NAME")

	remFetchSlp, err := strconv.ParseInt(os.Getenv("GNOMOTRON_REMINDER_FETCH_SLEEPTIME"), 10, 64)
	if err != nil {
		log.Panic("GNOMOTRON_REMINDER_FETCH_SLEEPTIME should be int")
	}
	cfg.REM_FETCH_SLEEPTIME_MIN = time.Duration(int64(time.Minute) * remFetchSlp)
	remCheckSlp, err := strconv.ParseInt(os.Getenv("GNOMOTRON_REMINDER_CHECK_SLEEPTIME"), 10, 64)
	if err != nil {
		log.Panic("GNOMOTRON_REMINDER_CHECK_SLEEPTIME should be int")
	}
	cfg.REM_CHECK_SLEEPTIME_MIN = time.Duration(int64(time.Minute) * remCheckSlp)

	if cfg.TOKEN == "" || cfg.CHAT_ID == "" || cfg.APIKEY == "" || cfg.MONGO_URI == "" || cfg.MONGO_DB == "" || cfg.BOT_NAME == "" {
		log.Panic("All cfg fields must be specified via the environment variables")
	}

	return &cfg
}
