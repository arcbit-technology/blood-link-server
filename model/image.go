package model

// Image targets: where an uploaded image is shown.
const (
	ImageUserAvatar = "user_avatar"
	ImageBankLogo   = "bank_logo"
	ImageBankCover  = "bank_cover"
)

var ImageTargets = []string{ImageUserAvatar, ImageBankLogo, ImageBankCover}

// ImageTargetRequest names an image slot (body of POST /image/signature,
// query of DELETE /image).
type ImageTargetRequest struct {
	Target      string `json:"target"`
	BloodBankId int    `json:"blood_bank_id"`
}

// SaveImageRequest is sent after the browser uploaded to Cloudinary.
type SaveImageRequest struct {
	ImageTargetRequest
	PublicId string `json:"public_id"`
	Version  int64  `json:"version"`
	Format   string `json:"format"`
}

// SavedImage is the stored image URL ("" after removal).
type SavedImage struct {
	Target string `json:"target"`
	Url    string `json:"url"`
}
