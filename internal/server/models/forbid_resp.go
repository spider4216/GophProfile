package models

type ForbiddenResp struct {
	Error   string `json:"error"`
	Details string `json:"details"`
}
