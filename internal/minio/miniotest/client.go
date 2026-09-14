package miniotest

import (
	"context"
	"io"
	"log/slog"
)

type S3ClientTest struct {
	bucketName string
	logger     *slog.Logger
	data       map[string][][]byte
}

func NewS3Client(bucket string, logger *slog.Logger, mstore map[string][][]byte) *S3ClientTest {
	return &S3ClientTest{
		bucketName: bucket,
		logger:     logger,
		data:       mstore,
	}
}

func (m *S3ClientTest) InitBucket() error {
	return nil
}

func (m *S3ClientTest) Upload(ctx context.Context, key string, reader io.ReadSeeker, ctype string) error {
	return nil
}

func (m *S3ClientTest) DeleteAva(ctx context.Context, key string) error {
	return nil
}

func (m *S3ClientTest) Download(ctx context.Context, key string) ([]byte, error) {
	return nil, nil
}
