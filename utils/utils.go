package utils

import (
	"log"

	"github.com/spf13/viper"
)

// InitConfig loads environment variables using Viper.
// It reads from `.env` if present, then falls back to OS env variables.
func InitConfig() {
	viper.SetConfigName(".env") // filename without extension
	viper.SetConfigType("env")  // treat it as dotenv format
	viper.AddConfigPath(".")    // search in current directory
	viper.AddConfigPath("./config")
	viper.AddConfigPath("..")

	// Automatically override with OS environment variables
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Println("No .env file found — falling back to environment vars")
	} else {
		log.Println("Loaded configuration from .env")
	}
}

