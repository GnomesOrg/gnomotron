package config

import (
	"log"

	"github.com/spf13/viper"
)

type Config struct {
	TOKEN                 string
	APIKEY                string
	MONGO_URI             string
	MONGO_DB              string
	BOT_NAME              string
	MAX_DIALOGUE_SIZE     int
	BOT_DEBUG             bool
	GPT_ENDPOINT          string
	STT_ENDPOINT          string
	STT_HOST string
}

func LoadConfig() *Config {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/app")

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Warning: No config file found, relying on environment variables: %v", err)
	}

	viper.BindEnv("TOKEN", "GNOMOTRON_TELEGRAM_TOKEN")
	viper.BindEnv("APIKEY", "GNOMOTRON_TELEGRAM_API_KEY")
	viper.BindEnv("MONGO_URI", "GNOMOTRON_MONGO_URI")
	viper.BindEnv("MONGO_DB", "GNOMOTRON_MONGO_DB")
	viper.BindEnv("BOT_NAME", "BOT_NAME")
	viper.BindEnv("STT_HOST", "STT_HOST")

	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		log.Panicf("Failed to load config: %v", err)
	}

	return cfg
}
