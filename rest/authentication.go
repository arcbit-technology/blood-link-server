package rest

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	config "github.com/mdhasib01/go-rest-starter/config"
	controller "github.com/mdhasib01/go-rest-starter/controller"
	model "github.com/mdhasib01/go-rest-starter/model"

	"github.com/gorilla/mux"
	"github.com/markbates/goth/gothic"
)

// Login godoc
// @Summary log into the app
// @Description that route is use to log into the app
// @Tags authentication
// @Accept  json
// @Produce  json
// @Param auth body model.LoginRequest true "auth"
// @Success 200 {object}   model.Login
// @Failure 400 {string} string
// @Failure 500 {string} string
// @Router /login [post]
func Login(w http.ResponseWriter, r *http.Request) {
	var user model.LoginRequest
	err := readRequestBody(r, &user)
	if err != nil {
		JSON(w, nil, err)
		return
	}

	set, err := controller.Login(user, r.Header.Get("X-Forwarded-For"))
	if err != nil {
		JSON(w, nil, err)
		return
	}

	JSON(w, set, nil)
}

// LoginProfile godoc
// @Summary log into profile
// @Description that route is use to log into profile
// @Tags authentication
// @Accept  json
// @Produce  json
// @Security Bearer
// @Param Authorization header string true "Bearer Token" default("Bearer YOUR_ACCESS_TOKEN")
// @Param profiletype path string true "profiletype"
// @Success 200 {object}   model.ProfileLogin
// @Failure 403 {string} string
// @Failure 400 {string} string
// @Failure 500 {string} string
// @Router /login/{profiletype} [get]
func LoginProfile(w http.ResponseWriter, r *http.Request) {
	session, err := isAuthorized(r, ProfileLoginPermission)
	if err != nil {
		JSON(w, nil, err)
		return
	}

	vars := mux.Vars(r)
	profileType := vars["profiletype"]
	profileType = strings.ToLower(profileType)

	login, err := controller.LoginProfile(profileType, session)
	JSON(w, login, err)

}

// Logout godoc
// @Summary log out the app
// @Description that route is use to log out the app
// @Tags authentication
// @Accept  json
// @Produce  json
// @Security Bearer
// @Param Authorization header string true "Bearer Token" default("Bearer YOUR_ACCESS_TOKEN")
// @Success 200 {string}   string
// @Failure 400 {string} string
// @Failure 500 {string} string
// @Router /logout [put]
func Logout(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	token = strings.Replace(token, "Bearer ", "", 1)
	if token == "" {
		ERROR(w, http.StatusBadRequest, errors.New("can not find token"))
		return
	}

	set, err := controller.Logout(token)
	JSON(w, set, err)

}

// IfTokenValid godoc
// @Summary check if token is valid
// @Description this route is used to check if the token is valid
// @Tags authentication
// @Produce  json
// @Security Bearer
// @Param Authorization header string true "Bearer Token" default("Bearer YOUR_ACCESS_TOKEN")
// @Success 200 {boolean} boolean
// @Failure 400 {string} string
// @Failure 500 {string} string
// @Router /iftokenvalid [get]
func IfTokenValid(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	token = strings.Replace(token, "Bearer ", "", 1)

	_, err := controller.IsAuthorized(token, 0)
	if err != nil {
		JSON(w, false, err)
		return
	}

	JSON(w, true, nil)
}

// SocialAuth godoc
// @Summary social auth
// @Description that route is use to authenticate with social media
// @Tags authentication
// @Produce  json
// @Success 200 {string}   string
// @Failure 403 {string} string
// @Failure 500 {string} string
// @Router /auth/{provider} [get]
func SocialAuth(w http.ResponseWriter, r *http.Request) {
	gothic.BeginAuthHandler(w, r)
}

// SocialAuthCallback godoc
// @Summary social auth callback
// @Description that route is use to authenticate with social media
// @Tags authentication
// @Produce  json
// @Success 200 {object}   model.Login
// @Failure 403 {string} string
// @Failure 500 {string} string
// @Router /auth/{provider}/callback [get]
func SocialAuthCallback(w http.ResponseWriter, r *http.Request) {

	user, err := gothic.CompleteUserAuth(w, r)
	if err != nil {
		JSON(w, nil, err)
		return
	}

	login, err := controller.SocialAuth(user)

	if err != nil {
		JSON(w, nil, err)
		return
	}

	http.Redirect(w, r, fmt.Sprintf(config.Param.ClientBaseURL+"/business?token=%s", login.Token), http.StatusFound)
}

// FetchLoginData godoc
// @Summary fetch login data
// @Description that route is use to fetch login data
// @Tags authentication
// @Accept  json
// @Produce  json
// @Security Bearer
// @Param Authorization header string true "Bearer Token" default("Bearer YOUR_ACCESS_TOKEN")
// @Success 200 {object}   model.Login
// @Failure 403 {string} string
// @Failure 400 {string} string
// @Failure 500 {string} string
// @Router /auth/fetchlogindata [get]
func FetchLoginData(w http.ResponseWriter, r *http.Request) {
	session, err := isAuthorized(r, 0)
	if err != nil {
		JSON(w, nil, err)
		return
	}

	login, err := controller.FetchLoginData(session)
	if err != nil {
		JSON(w, nil, err)
		return
	}

	JSON(w, login, nil)
}

// Register godoc
// @Summary register a new user
// @Description that route is use to register a new user
// @Tags authentication
// @Accept  json
// @Produce  json
// @Param body body model.User true "Request body containing user registration information"
// @Success 200 {object}   model.RegisterResponse
// @Failure 400 {string} string
// @Failure 403 {string} string
// @Failure 409 {string} string
// @Failure 500 {string} string
// @Router /register [post]
func Register(w http.ResponseWriter, r *http.Request) {

	var user model.User
	err := readRequestBody(r, &user)
	if err != nil {
		JSON(w, nil, err)
		return
	}

	user.IdRole = model.USERROLE
	user.CreatedBy = model.OUTBOUND_USER

	var fr model.FileRequest

	set, err := controller.Register(user, fr, r.Header.Get("X-Forwarded-For"))
	JSON(w, set, err)
}

// VerifyAccount godoc
// @Summary verify account
// @Description this route is used to verify account
// @Tags authentication
// @Produce  html
// @Param code path string true "verification code"
// @Success 200 {string} string "html content"
// @Router /verify/{code} [get]
func VerifyAccount(w http.ResponseWriter, r *http.Request) {

	code := mux.Vars(r)["code"]

	w.Header().Set("Content-Type", "text/html")
	firstName, err := controller.VerifyAccount(code)
	if err != nil {
		fmt.Fprintf(w, model.HtmlErrorVerificationPage, err.Error())
		return
	}

	fmt.Fprintf(w, model.HtmlVerifiedEmailPage, firstName, "http://localhost:3000"+"/login")
}

// ForgotPassword godoc
// @Summary forgot password
// @Description that route is use to send an email to the user with a temporary password
// @Tags authentication
// @Accept  json
// @Produce  json
// @Param body body model.ForgotPasswordRequest true "Request body containing user email"
// @Success 200 {object}   model.ForgotPasswordResponse
// @Failure 400 {string} string
// @Failure 500 {string} string
// @Router /forgotpassword [post]
func ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var body model.ForgotPasswordRequest
	err := readRequestBody(r, &body)
	if err != nil {
		JSON(w, nil, err)
		return
	}

	res, err := controller.ForgotPassword(body)
	if err != nil {
		JSON(w, nil, err)
		return
	}

	JSON(w, res, nil)
}
