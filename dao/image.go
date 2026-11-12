package dao

import (
	model "github.com/mdhasib01/go-rest-starter/model"
	"github.com/mdhasib01/go-rest-starter/pkg/logger"
)

// SetUserPhoto stores (or clears, with "") the user's profile photo URL.
func SetUserPhoto(userID int, url string) error {
	if _, err := DB.Exec(`UPDATE users SET photo = NULLIF($2, ''), updated_at = now() WHERE id = $1`, userID, url); err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return model.UnknownError
	}
	return nil
}

// SetBankImage stores (or clears) a blood bank's logo or cover URL.
func SetBankImage(bankID int, target, url string) error {
	column := "logo"
	if target == model.ImageBankCover {
		column = "cover"
	}
	// column is one of two constants, never user input
	if _, err := DB.Exec(`UPDATE blood_banks SET `+column+` = NULLIF($2, ''), updated_at = now() WHERE id = $1`, bankID, url); err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return model.UnknownError
	}
	return nil
}
