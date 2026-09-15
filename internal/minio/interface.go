package minio

import (
	"context"
	"io"
)

type S3ClientInterface interface {
	InitBucket() error
	Upload(ctx context.Context, key string, reader io.ReadSeeker, ctype string) error
	DeleteAva(ctx context.Context, key string) error
	Download(ctx context.Context, key string) ([]byte, error)
}
