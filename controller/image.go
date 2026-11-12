package controller

import (
	"fmt"
	"strings"
	"time"

	config "github.com/mdhasib01/go-rest-starter/config"
	dao "github.com/mdhasib01/go-rest-starter/dao"
	itn "github.com/mdhasib01/go-rest-starter/itn"
	model "github.com/mdhasib01/go-rest-starter/model"
	"github.com/mdhasib01/go-rest-starter/pkg/cloudinary"
	"github.com/mdhasib01/go-rest-starter/pkg/logger"
)

// maxImageBytes caps stored images; the cropper uploads ~100-500 KB.
const maxImageBytes = 5 * 1024 * 1024

// imageHost returns the Cloudinary client, or 503 when it is not configured.
var imageHost = func() (*cloudinary.Client, error) {
	c := config.Param.Cloudinary
	if !c.Configured() {
		return nil, model.NewError(itn.ErrorImageUploadDisabled, 503)
	}
	return cloudinary.New(c.CloudName, c.ApiKey, c.ApiSecret), nil
}

// imagePublicID returns the fixed Cloudinary public id of an image slot
// after checking that the caller may change it: their own avatar, or the
// logo / cover of a blood bank they administer.
func imagePublicID(req model.ImageTargetRequest, session model.Session) (string, error) {
	switch req.Target {
	case model.ImageUserAvatar:
		return fmt.Sprintf("bloodlink/users/%d/avatar", session.UserId), nil
	case model.ImageBankLogo, model.ImageBankCover:
		if req.BloodBankId < 1 {
			return "", model.NewError(itn.ErrorImageInvalid, 400)
		}
		if _, err := dao.GetBloodBank(req.BloodBankId); err != nil {
			return "", err
		}
		if _, err := requireBankRole(session, req.BloodBankId, model.BankRoleAdmin); err != nil {
			return "", err
		}
		kind := strings.TrimPrefix(req.Target, "bank_")
		return fmt.Sprintf("bloodlink/banks/%d/%s", req.BloodBankId, kind), nil
	default:
		return "", model.NewError(itn.ErrorImageInvalid, 400)
	}
}

// SignImageUpload returns the signed parameters for a direct upload.
func SignImageUpload(req model.ImageTargetRequest, session model.Session) (cloudinary.SignedUpload, error) {
	host, err := imageHost()
	if err != nil {
		return cloudinary.SignedUpload{}, err
	}
	publicID, err := imagePublicID(req, session)
	if err != nil {
		return cloudinary.SignedUpload{}, err
	}
	return host.SignUpload(publicID, time.Now()), nil
}

// SaveImage stores the URL of an image the browser uploaded with a
// signature from SignImageUpload. The public id must be the slot's own id
// and the image must exist on Cloudinary.
func SaveImage(req model.SaveImageRequest, session model.Session) (model.SavedImage, error) {
	host, err := imageHost()
	if err != nil {
		return model.SavedImage{}, err
	}
	publicID, err := imagePublicID(req.ImageTargetRequest, session)
	if err != nil {
		return model.SavedImage{}, err
	}
	if req.PublicId != publicID || req.Version < 1 {
		return model.SavedImage{}, model.NewError(itn.ErrorImageInvalid, 400)
	}

	// trust only Cloudinary's record of the upload, not the client's
	res, found, err := host.Resource(publicID)
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return model.SavedImage{}, model.NewError(itn.ErrorImageNotUploaded, 502)
	}
	if !found {
		return model.SavedImage{}, model.NewError(itn.ErrorImageNotUploaded, 400)
	}
	format := strings.ToLower(strings.TrimSpace(req.Format))
	if res.Version != req.Version || (format != "" && format != res.Format) ||
		!contains(cloudinary.AllowedFormats, res.Format) || res.Bytes > maxImageBytes {
		return model.SavedImage{}, model.NewError(itn.ErrorImageInvalid, 400)
	}

	url := host.DeliveryURL(publicID, res.Version, res.Format)

	if err := storeImage(req.ImageTargetRequest, session, url); err != nil {
		return model.SavedImage{}, err
	}
	return model.SavedImage{Target: req.Target, Url: url}, nil
}

// RemoveImage deletes the image from Cloudinary (best effort) and clears it.
func RemoveImage(req model.ImageTargetRequest, session model.Session) (model.SavedImage, error) {
	publicID, err := imagePublicID(req, session)
	if err != nil {
		return model.SavedImage{}, err
	}
	if host, err := imageHost(); err == nil {
		if err := host.Destroy(publicID); err != nil {
			logger.GetLogger().LogErrors(err, map[string]interface{}{"public_id": publicID})
		}
	}
	if err := storeImage(req, session, ""); err != nil {
		return model.SavedImage{}, err
	}
	return model.SavedImage{Target: req.Target}, nil
}

func storeImage(req model.ImageTargetRequest, session model.Session, url string) error {
	if req.Target == model.ImageUserAvatar {
		return dao.SetUserPhoto(session.UserId, url)
	}
	return dao.SetBankImage(req.BloodBankId, req.Target, url)
}
