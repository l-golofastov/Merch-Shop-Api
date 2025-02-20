package config

import (
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"log"
	"time"
)

type Config struct {
	HttpServer
	Postgres
}

type HttpServer struct {
	ServerAddress string        `envconfig:"SERVER_ADDRESS" default:"localhost:8080"`
	Timeout       time.Duration `envconfig:"SERVER_TIMEOUT" default:"4s"`
	IdleTimeout   time.Duration `envconfig:"SERVER_IDLE_TIMEOUT" default:"60s"`
}

type Postgres struct {
	Host     string `envconfig:"POSTGRES_HOST"`
	Port     string `envconfig:"POSTGRES_PORT"`
	Username string `envconfig:"POSTGRES_USERNAME"`
	Password string `envconfig:"POSTGRES_PASSWORD"`
	DBName   string `envconfig:"POSTGRES_DBNAME"`
}

func MustLoad() *Config {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading env variables", err)
	}

	var cfg Config
	err := envconfig.Process("", &cfg)
	if err != nil {
		log.Fatal("Error processing env variables:", err)
	}

	return &cfg
}
