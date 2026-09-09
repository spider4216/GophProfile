package config

import (
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	DbDsn         string        `env:"DB_DSN"`                       // Connection string для БД
	LogLvl        string        `env:"LOG_LEVEL" envDefault:"debug"` // Уровень логирования
	MaxBodySize   int64         `env:"MAX_BODY_SIZE" envDefault:"2048"`
	ServerAddress string        `env:"SERVER_ADDRESS"` // Адрес запуска HTTP-сервера
	ReadTimeout   time.Duration `env:"READ_TIMEOUT" envDefault:"5s"`
	WriteTimeout  time.Duration `env:"WRITE_TIMEOUT" envDefault:"10s"`
	IdleTimeout   time.Duration `env:"IDLE_TIMEOUT" envDefault:"30s"`
	PKPath        string        `env:"PK_PATH" envDefault:"certs/private.pem"` // Путь до приватного ключа для режимо HTTPS
	CrtPath       string        `env:"CRT_PATH" envDefault:"certs/cert.pem"`   // Путь до сертификата для режимо HTTPS
}

func New() (*Config, error) {
	cfg := &Config{}

	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
