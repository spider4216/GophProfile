package minio

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

const (
	region = "kz-1"
	useSSL = false
)

type S3Client struct {
	cli        *s3.S3
	bucketName string
}

func NewS3Client(login string, pass string, host string, bucket string) *S3Client {
	s3Cfg := &aws.Config{
		Region:           aws.String(region),
		Endpoint:         aws.String(host),
		S3ForcePathStyle: aws.Bool(true),
		Credentials:      credentials.NewStaticCredentials(login, pass, ""),
		DisableSSL:       aws.Bool(!useSSL),
	}

	sess := session.Must(session.NewSession(s3Cfg))
	client := s3.New(sess)

	return &S3Client{
		cli:        client,
		bucketName: bucket,
	}
}

func (s *S3Client) Upload(ctx context.Context, key string, reader io.ReadSeeker, ctype string) error {
	_, err := s.cli.PutObjectWithContext(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(key),
		Body:        reader,
		ContentType: aws.String(ctype),
	})
	if err != nil {
		return fmt.Errorf("cannot put object to bucket s3: %w", err)
	}

	return nil
}

func (s *S3Client) DeleteAva(ctx context.Context, key string) error {
	_, err := s.cli.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("cannot delete object from minio: %w", err)
	}

	return nil
}

func (s *S3Client) Download(ctx context.Context, key string) ([]byte, error) {
	// Получаем объект из S3
	result, err := s.cli.GetObjectWithContext(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to download object from bucket s3: %w", err)
	}

	defer result.Body.Close()

	// Читаем данные в buffer
	buf := &bytes.Buffer{}

	_, err = io.Copy(buf, result.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read object data in s3: %w", err)
	}

	return buf.Bytes(), nil
}
