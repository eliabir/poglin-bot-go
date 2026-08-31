package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	ApiKey string `env:"DISCORD_API"`
}

func Load() (*Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("parsing configfailed: %w", err)
	}

	return &cfg, nil
}
