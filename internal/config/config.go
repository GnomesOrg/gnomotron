package config

import (
	"log"
	"strings"
	"github.com/spf13/viper"
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
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/app")
	viper.ReadInConfig()

	viper.AutomaticEnv()

	viper.SetEnvKeyReplacer(strings.NewReplacer(
		"GNOMOTRON_TELEGRAM_TOKEN", "TOKEN",
		"GNOMOTRON_TELEGRAM_API_KEY", "APIKEY",
		"GNOMOTRON_MONGO_URI", "MONGO_URI",
		"GNOMOTRON_MONGO_DB", "MONGO_DB",
		"BOT_NAME", "BOT_NAME",
	))

	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		log.Panicf("Failed to load config: %v", err)
	}

	return cfg
}
