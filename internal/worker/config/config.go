package config

import (
	"github.com/caarlos0/env/v11"
)

type Config struct {
	// todo Dsn to upper
	LogLvl    string `env:"LOG_LEVEL" envDefault:"debug"` // Уровень логирования
	RabbitDSN string `env:"RABBIT_DSN"`
}

func New() (*Config, error) {
	cfg := &Config{}

	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
