package model

import (
	"time"

	"github.com/mdhasib01/go-rest-starter/pkg/data"
)

type Permissione struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}
type Session struct {
	Id                int       `json:"code"`
	Token             string    `json:"token"`
	RoleId            int       `json:"idrole"`
	UserId            int       `json:"iduser"`
	BuyerProfileId    int       `json:"buyerprofileid"`
	SellerProfileId   int       `json:"sellerprofileid"`
	LoggedInProfileId int       `json:"loggedinprofileid"`
	IsValid           bool      `json:"isvalid"`
	LastUsed          string    `json:"lastused"`
	CreatedAt         time.Time `json:"createdat"`
	Currency          string    `json:"currency"`
	ExpirationDate    time.Time `json:"expireat"`
	ISOCountry        string    `json:"isocountry"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Login struct {
	User           User      `json:"user"`
	Role           Role      `json:"role"`
	LastLogin      time.Time `json:"lastlogin"`
	Plan           string    `json:"plan"`
	EvaluationUsed int       `json:"evaluation_used"`

	Token string `json:"token"`
}

type ProfileLogin struct {
	Login
	LoggedInProfileType string `json:"loggedinprofiletype,omitempty"`
	LoggedInProfileId   int    `json:"loggedinprofileid,omitempty"`
}

type RegisterRequest struct {
	Firstname       string                `json:"firstname"`
	Lastname        string                `json:"lastname"`
	Password        data.HiddenJsonString `json:"password"`
	ConfirmPassword string                `json:"confirmpassword"`
	Username        string                `json:"username"`
	Email           string                `json:"email"`
	Phone           string                `json:"phone"`
	City            string                `json:"city"`
	Country         string                `json:"country"`
	Address         string                `json:"address"`
	Photo           string                `json:"photo"`
}

type RegisterResponse struct {
	Message string `json:"message"`
	User    User   `json:"user"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

type ForgotPasswordResponse struct {
	Message     string                `json:"message"`
	NewPassword data.HiddenJsonString `json:"newpassword"`
}
