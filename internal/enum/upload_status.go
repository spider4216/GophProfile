package enum

type UploadStatus string

const (
	Uploading UploadStatus = "uploading"
	Uploaded  UploadStatus = "uploaded"
)

func (us UploadStatus) String() string {
	return string(us)
}

type ProcStatus string

const (
	ProcPending ProcStatus = "pending"
	ProcDone    ProcStatus = "done"
)

func (us ProcStatus) String() string {
	return string(us)
}
