package services

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/spider4216/GophProfile/internal/logger"
	"github.com/spider4216/GophProfile/internal/minio/miniotest"
	"github.com/spider4216/GophProfile/internal/models"
	"github.com/spider4216/GophProfile/internal/queue/qtest"
	"github.com/spider4216/GophProfile/internal/repositories/reptest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func prepareService(store map[string][][]byte, qstore map[string][][]byte, mstore map[string]map[string][]byte) *Service {
	logger := logger.Init("debug")
	repo := reptest.NewRepository(logger, store)
	queue := qtest.NewQueue(logger, qstore)
	minio := miniotest.NewS3Client("test", logger, mstore)

	return New(
		repo,
		logger,
		queue,
		minio,
	)
}

func TestCreateAvatar(t *testing.T) {
	store := map[string][][]byte{}
	qstore := map[string][][]byte{}
	mstore := map[string]map[string][]byte{}
	mstore["test"] = map[string][]byte{}

	service := prepareService(store, qstore, mstore)
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

func TestCreateTmpFile(t *testing.T) {
	store := map[string][][]byte{}
	qstore := map[string][][]byte{}
	mstore := map[string]map[string][]byte{}
	mstore["test"] = map[string][]byte{}

	service := prepareService(store, qstore, mstore)

	f, err := os.Create("/tmp/src")
	require.NoError(t, err)

	uid := uuid.NewString()

	err = service.CreateTmpFile(t.Context(), "test.jpg", f, uid)
	require.NoError(t, err)

	path := "/tmp/test_" + uid + ".jpg"

	_, err = os.Open(path)

	require.NoError(t, err)

	err = os.Remove(path)
	require.NoError(t, err)
}

func TestSendUploadEvent(t *testing.T) {
	store := map[string][][]byte{}
	qstore := map[string][][]byte{}
	mstore := map[string]map[string][]byte{}
	mstore["test"] = map[string][]byte{}

	service := prepareService(store, qstore, mstore)

	avaID := uuid.NewString()
	userID := uuid.NewString()
	s3Key := uuid.NewString()

	err := service.SendUploadEvent(t.Context(), userID, avaID, s3Key)
	require.NoError(t, err)

	raw := qstore["uploads"][0]

	assert.NotEmpty(t, raw)

	var e models.AvatarUploadEvent

	err = json.Unmarshal(raw, &e)
	require.NoError(t, err)

	assert.Equal(t, avaID, e.AvatarID)
	assert.Equal(t, userID, e.UserID)
	assert.Equal(t, s3Key, e.S3Key)
}

func TestSendDeleteEvents(t *testing.T) {
	store := map[string][][]byte{}
	qstore := map[string][][]byte{}
	mstore := map[string]map[string][]byte{}
	mstore["test"] = map[string][]byte{}

	service := prepareService(store, qstore, mstore)
	id1 := uuid.NewString()
	id2 := uuid.NewString()

	avas := []models.Avatar{
		{
			ID: id1,
		},
		{
			ID: id2,
		},
	}

	err := service.SendDeleteEvents(t.Context(), avas)
	require.NoError(t, err)

	assert.Equal(t, 2, len(qstore["deletes"]))

	raw1 := qstore["deletes"][0]
	raw2 := qstore["deletes"][1]

	var e1 models.AvatarDeleteEvent
	var e2 models.AvatarDeleteEvent

	err = json.Unmarshal(raw1, &e1)
	require.NoError(t, err)

	err = json.Unmarshal(raw2, &e2)
	require.NoError(t, err)

	assert.Equal(t, id1, e1.AvatarID)
	assert.Equal(t, id2, e2.AvatarID)
}

func TestGetBinaryAva(t *testing.T) {
	store := map[string][][]byte{}
	qstore := map[string][][]byte{}
	mstore := map[string]map[string][]byte{}
	mstore["test"] = map[string][]byte{}

	s3key := uuid.NewString()

	data := []byte("Hello World")
	// Кладем в minio слайс байт в бакет и ключ
	mstore["test"][s3key] = data

	service := prepareService(store, qstore, mstore)

	b, err := service.GetBinaryAva(t.Context(), s3key)
	require.NoError(t, err)

	assert.Equal(t, data, b)
}

func TestGetBinaryThumbnail(t *testing.T) {
	store := map[string][][]byte{}
	qstore := map[string][][]byte{}
	mstore := map[string]map[string][]byte{}
	mstore["test"] = map[string][]byte{}

	thumbS3key := uuid.NewString()

	data := []byte("Hello Thumbnail")
	// Кладем в minio слайс байт в бакет и ключ - это будет гаш thumbnail
	mstore["test"][thumbS3key] = data

	service := prepareService(store, qstore, mstore)

	avaID := uuid.NewString()
	ava := models.Avatar{
		ID: avaID,
		ThumbnailS3Keys: map[string]string{
			"100x100": thumbS3key,
		},
	}

	b, err := service.GetBinaryThumbnail(t.Context(), &ava, "100x100")
	require.NoError(t, err)
	assert.Equal(t, data, b)

	b, err = service.GetBinaryThumbnail(t.Context(), &ava, "300x300")
	assert.Error(t, err)
	assert.Nil(t, b)
}
