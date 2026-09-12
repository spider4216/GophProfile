package config

import (
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	// todo Dsn to upper
	DbDsn         string        `env:"DB_DSN"`                       // Connection string для БД
	LogLvl        string        `env:"LOG_LEVEL" envDefault:"debug"` // Уровень логирования
	MaxBodySize   int64         `env:"MAX_BODY_SIZE" envDefault:"2048"`
	ServerAddress string        `env:"SERVER_ADDRESS"` // Адрес запуска HTTP-сервера
	ReadTimeout   time.Duration `env:"READ_TIMEOUT" envDefault:"5s"`
	WriteTimeout  time.Duration `env:"WRITE_TIMEOUT" envDefault:"10s"`
	IdleTimeout   time.Duration `env:"IDLE_TIMEOUT" envDefault:"30s"`
	PKPath        string        `env:"PK_PATH" envDefault:"certs/private.pem"` // Путь до приватного ключа для режимо HTTPS
	CrtPath       string        `env:"CRT_PATH" envDefault:"certs/cert.pem"`   // Путь до сертификата для режимо HTTPS
	RabbitDSN     string        `env:"RABBIT_DSN"`
	SupportImgExt []string      `env:"SUPPORT_IMG_EXT" envDefault:"image/jpeg,image/png,image/webp"`
	MaxImgSize    int64         `env:"MAX_ING_SIZE" envDefault:"10485760"`
	CacheTTL      int           `env:"CACHE_TTL" envDefault"86400"`
	// todo minio to general cfg
	MinioHost  string `env:"MINIO_HOST"`
	MinioUser  string `env:"MINIO_ROOT_USER"`
	MinioPass  string `env:"MINIO_ROOT_PASSWORD"`
	BucketName string `env:"BUCKET_NAME" envDefault:"avatars"`
}

func New() (*Config, error) {
	cfg := &Config{}

	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
