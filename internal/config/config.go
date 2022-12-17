package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type SOCKS5Config struct {
	Port string
	Host string
}

type Config struct {
	Proxy    SOCKS5Config
	LogLevel string
}

var cfg Config

func GetConfig() *Config {
	return &cfg
}

func init() {
	err := godotenv.Load(".env")
	if err != nil {
		panic(err)
	}

	cfg.Proxy.Host = os.Getenv("SOCKS5_HOST")
	cfg.Proxy.Port = os.Getenv("SOCKS5_PORT")

	if strings.ToLower(os.Getenv("DEBUG")) == "true" {
		cfg.LogLevel = "debug"
	} else {
		cfg.LogLevel = "info"
	}
}
