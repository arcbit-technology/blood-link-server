package rest

import (
	model "github.com/mdhasib01/go-rest-starter/model"
)

func getActiveProfileID(session model.Session) int {
	if session.BuyerProfileId > 0 {
		return session.BuyerProfileId
	}
	if session.SellerProfileId > 0 {
		return session.SellerProfileId
	}
	return 0
}

func isValidFileType(contentType string) bool {
	switch contentType {
	case "image/png", "image/jpg", "image/jpeg", "video/mp4", "application/pdf", "audio/wav", "audio/webm", "audio/ogg", "audio/mpeg", "audio/x-wav", "audio/mp3":
		return true
	default:
		return false
	}
}

func getFileType(contentType string) model.FileType {
	switch contentType {
	case "image/png", "image/jpg", "image/jpeg":
		return model.ImageType
	case "video/mp4":
		return model.VideoType
	case "application/pdf":
		return model.DocumentType
	case "video/webm", "audio/wav", "audio/webm", "audio/ogg", "audio/mpeg", "audio/x-wav", "audio/mp3":
		return model.AudioType
	default:
		return ""
	}
}
