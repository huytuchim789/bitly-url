package config

import "github.com/caarlos0/env/v11"

type Config struct {
	Port        string `env:"PORT" envDefault:"8080"`
	DatabaseURL string `env:"DATABASE_URL" envDefault:"postgres://postgres:postgres@localhost:5432/bitly?sslmode=disable"`
	RedisURL    string `env:"REDIS_URL" envDefault:"redis://localhost:6379/0"`
	LogLevel    string `env:"LOG_LEVEL" envDefault:"info"`
	Environment string `env:"ENVIRONMENT" envDefault:"development"`
}

func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
