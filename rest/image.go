package rest

import (
	"net/http"
	"strconv"

	controller "github.com/mdhasib01/go-rest-starter/controller"
	"github.com/mdhasib01/go-rest-starter/model"
)

// SignImageUpload godoc
// @Summary sign a direct image upload to Cloudinary
// @Description target: user_avatar (own photo) | bank_logo | bank_cover (blood bank admin). Upload the file to upload_url with public_id, timestamp, signature, api_key, overwrite, invalidate and allowed_formats exactly as returned, then call PUT /image.
// @Tags images
// @Accept  json
// @Produce  json
// @Security Bearer
// @Param Authorization header string true "Bearer Token" default("Bearer YOUR_ACCESS_TOKEN")
// @Param body body model.ImageTargetRequest true "target, blood_bank_id"
// @Success 200 {object} cloudinary.SignedUpload
// @Failure 403 {string} string
// @Failure 503 {string} string
// @Router /image/signature [post]
func SignImageUpload(w http.ResponseWriter, r *http.Request) {
	session, err := isAuthorized(r, anyAuthenticatedUser)
	if err != nil {
		JSON(w, nil, err)
		return
	}
	var req model.ImageTargetRequest
	if err := readRequestBody(r, &req); err != nil {
		JSON(w, nil, err)
		return
	}
	res, err := controller.SignImageUpload(req, session)
	JSON(w, res, err)
}

// SaveImage godoc
// @Summary save an uploaded image as profile photo / blood bank logo / cover
// @Tags images
// @Accept  json
// @Produce  json
// @Security Bearer
// @Param Authorization header string true "Bearer Token" default("Bearer YOUR_ACCESS_TOKEN")
// @Param body body model.SaveImageRequest true "target, blood_bank_id, public_id, version, format (from the Cloudinary upload response)"
// @Success 200 {object} model.SavedImage
// @Failure 400 {string} string
// @Router /image [put]
func SaveImage(w http.ResponseWriter, r *http.Request) {
	session, err := isAuthorized(r, anyAuthenticatedUser)
	if err != nil {
		JSON(w, nil, err)
		return
	}
	var req model.SaveImageRequest
	if err := readRequestBody(r, &req); err != nil {
		JSON(w, nil, err)
		return
	}
	res, err := controller.SaveImage(req, session)
	JSON(w, res, err)
}

// RemoveImage godoc
// @Summary remove a profile photo / blood bank logo / cover
// @Tags images
// @Produce  json
// @Security Bearer
// @Param Authorization header string true "Bearer Token" default("Bearer YOUR_ACCESS_TOKEN")
// @Param target query string true "user_avatar | bank_logo | bank_cover"
// @Param blood_bank_id query int false "blood bank id"
// @Success 200 {object} model.SavedImage
// @Router /image [delete]
func RemoveImage(w http.ResponseWriter, r *http.Request) {
	session, err := isAuthorized(r, anyAuthenticatedUser)
	if err != nil {
		JSON(w, nil, err)
		return
	}
	q := r.URL.Query()
	req := model.ImageTargetRequest{Target: q.Get("target")}
	req.BloodBankId, _ = strconv.Atoi(q.Get("blood_bank_id"))
	res, err := controller.RemoveImage(req, session)
	JSON(w, res, err)
}
