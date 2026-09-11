package config

import (
	"github.com/caarlos0/env/v11"
)

type Config struct {
	// todo Dsn to upper
	LogLvl     string `env:"LOG_LEVEL" envDefault:"debug"` // Уровень логирования
	RabbitDSN  string `env:"RABBIT_DSN"`
	MinioHost  string `env:"MINIO_HOST"`
	MinioUser  string `env:"MINIO_ROOT_USER"`
	MinioPass  string `env:"MINIO_ROOT_PASSWORD"`
	DbDSN      string `env:"DB_DSN"` // Connection string для БД
	BucketName string `env:"BUCKET_NAME" envDefault:"avatars"`
}

func New() (*Config, error) {
	cfg := &Config{}

	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
