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
	Id                int       `json:"id"`
	Token             string    `json:"token"`
	RoleId            int       `json:"role_id"`
	UserId            int       `json:"user_id"`
	BuyerProfileId    int       `json:"buyer_profile_id"`
	SellerProfileId   int       `json:"seller_profile_id"`
	LoggedInProfileId int       `json:"logged_in_profile_id"`
	IsValid           bool      `json:"is_valid"`
	LastUsed          string    `json:"last_used"`
	CreatedAt         time.Time `json:"created_at"`
	Currency          string    `json:"currency"`
	ExpirationDate    time.Time `json:"expire_at"`
	ISOCountry        string    `json:"iso_country"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type Login struct {
	User           User      `json:"user"`
	Role           Role      `json:"role"`
	LastLogin      time.Time `json:"last_login"`
	Plan           string    `json:"plan"`
	EvaluationUsed int       `json:"evaluation_used"`

	Token string `json:"token"`
}

type ProfileLogin struct {
	Login
	LoggedInProfileType string `json:"logged_in_profile_type,omitempty"`
	LoggedInProfileId   int    `json:"logged_in_profile_id,omitempty"`
}

type RegisterRequest struct {
	Firstname       string                `json:"first_name"`
	Lastname        string                `json:"last_name"`
	Password        data.HiddenJsonString `json:"password"`
	ConfirmPassword string                `json:"confirm_password"`
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
	NewPassword data.HiddenJsonString `json:"new_password"`
}
