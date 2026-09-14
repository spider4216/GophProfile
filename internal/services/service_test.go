package services

import (
	"encoding/json"
	"testing"

	"github.com/spider4216/GophProfile/internal/logger"
	"github.com/spider4216/GophProfile/internal/minio/miniotest"
	"github.com/spider4216/GophProfile/internal/models"
	"github.com/spider4216/GophProfile/internal/queue/qtest"
	"github.com/spider4216/GophProfile/internal/repositories/reptest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func prepareService(store map[string][][]byte) *Service {
	logger := logger.Init("debug")
	repo := reptest.NewRepository(logger, store)
	queue := qtest.NewQueue(logger)
	minio := miniotest.NewS3Client("test", logger)

	return New(
		repo,
		logger,
		queue,
		minio,
	)
}

func TestCreateAvatar(t *testing.T) {
	store := map[string][][]byte{}

	service := prepareService(store)
	ava, err := service.CreateAvatar(
		t.Context(),
		"test.jpg",
		"application/jpeg",
		12345,
		"testkey",
	)

	require.NoError(t, err)
	assert.NotEmpty(t, ava)

	raw := store["avatars"][0]

	assert.NotEmpty(t, raw)

	var avaSrc models.Avatar

	err = json.Unmarshal(raw, &avaSrc)
	require.NoError(t, err)

	assert.Equal(t, ava.ID, avaSrc.ID)
	assert.Equal(t, ava.FileName, avaSrc.FileName)
}
