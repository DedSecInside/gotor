package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

type SOCKS5Config struct {
	Port string
	Host string
}

type Config struct {
	Proxy    SOCKS5Config
	LogLevel string
	UseTor   bool
}

var cfg Config

func GetConfig() *Config {
	return &cfg
}

func init() {
	err := godotenv.Load(".env")
	if err != nil {
		zap.L().Sugar().Infow("unable to load .env file, program may behave strangely without environment variables",
			"error", err,
		)
		return
	}

	host := os.Getenv("SOCKS5_HOST")
	if host != "" {
		host = "localhost"
	}
	cfg.Proxy.Host = host

	port := os.Getenv("SOCKS5_PORT")
	if port != "" {
		port = "9050"
	}
	cfg.Proxy.Port = port

	if strings.ToLower(os.Getenv("DEBUG")) == "true" {
		cfg.LogLevel = "debug"
	} else {
		cfg.LogLevel = "info"
	}

	if strings.ToLower(os.Getenv("USE_TOR")) == "false" {
		cfg.UseTor = false
	} else {
		cfg.UseTor = true
	}
}
