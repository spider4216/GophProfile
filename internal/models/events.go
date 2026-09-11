package models

type AvatarUploadEvent struct {
	AvatarID string `json:"avatar_id"`
	UserID   string `json:"user_id"`
	S3Key    string `json:"s3_key"`
}

type ProcessingOp string

func (p ProcessingOp) String() string {
	return string(p)
}

type AvatarProcessEvent struct {
	AvatarID   string         `json:"avatar_id"`
	Operations []ProcessingOp `json:"operations"`
}
