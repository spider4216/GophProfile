package models

import (
	"time"

	"github.com/spider4216/GophProfile/internal/enum"
)

type UploadResp struct {
	ID        string            `json:"id"`
	UserID    string            `json:"user_id"`
	URL       string            `json:"url"`
	Status    enum.UploadStatus `json:"status"`
	CreatedAt time.Time         `json:"created_at"`
}
