package handlers

import (
	"path"

	genModel "github.com/spider4216/GophProfile/internal/models"
	"github.com/spider4216/GophProfile/internal/server/models"
)

func (h *Handler) mapMetasResp(avas []genModel.Avatar, host string) []models.GetMetaResp {
	var metas []models.GetMetaResp

	for _, ava := range avas {
		metas = append(metas, *h.mapMetaResp(&ava, host))
	}

	return metas
}

func (h *Handler) mapMetaResp(ava *genModel.Avatar, host string) *models.GetMetaResp {
	var thumbnails []models.Thumbnail

	for k, _ := range ava.ThumbnailS3Keys {
		url := path.Join("https://", host, "/api/v1/avatars/", ava.ID)
		url += "?size=" + k

		th := models.Thumbnail{
			Size: k,
			URL:  url,
		}

		thumbnails = append(thumbnails, th)
	}

	return &models.GetMetaResp{
		ID:         ava.ID,
		UserID:     ava.UserID,
		FileName:   ava.FileName,
		MimeType:   ava.MimeType,
		Size:       ava.SizeBytes,
		Thumbnails: thumbnails,
		CreatedAt:  ava.CreatedAt,
		UpdatedAt:  ava.UpdatedAt,
	}
}
