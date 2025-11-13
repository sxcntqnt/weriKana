// config/config.go
package config

import "os"

type Config struct {
	SecretKey string
	Port      string
	Database  string
}

func LoadConfig() Config {
	return Config{
		SecretKey: os.Getenv("SECRET_KEY"),
		Port:      ":8000",
		Database:  "bank.db",
	}
}
