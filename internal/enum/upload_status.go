package enum

type UploadStatus string

const (
	Uploading UploadStatus = "uploading"
	Uploaded  UploadStatus = "uploaded"
)

func (us UploadStatus) String() string {
	return string(us)
}
