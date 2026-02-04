package model

import (
	"time"

	"github.com/mdhasib01/go-rest-starter/pkg/data"
)

type User struct {
	Id                  int                   `json:"id,omitempty"`
	Firstname           string                `json:"firstname,omitempty"`
	Lastname            string                `json:"lastname,omitempty"`
	Password            data.HiddenJsonString `json:"password,omitempty"`
	Username            string                `json:"username,omitempty"`
	Email               string                `json:"email,omitempty"`
	Phone               string                `json:"phone,omitempty"`
	Address             string                `json:"address,omitempty"`
	City                string                `json:"city,omitempty"`
	District            string                `json:"district,omitempty"`
	Division            string                `json:"division,omitempty"`
	Country             string                `json:"country,omitempty"`
	IsActive            bool                  `json:"isactive,omitempty"`
	IsVerified          bool                  `json:"isverified,omitempty"`
	LastLogin           string                `json:"lastlogin,omitempty"`
	CreatedAt           time.Time             `json:"createdat,omitempty"`
	UpdatedAt           time.Time             `json:"updatedat,omitempty"`
	CreatedBy           int                   `json:"createdby,omitempty"`
	UpdateBy            int                   `json:"updatedby,omitempty"`
	Profession          string                `json:"profession,omitempty"`
	IsPasswordTemporary bool                  `json:"ispasswordtemporary,omitempty"`
	RoleName            string                `json:"role,omitempty"`
	IdRole              int                   `json:"idrole,omitempty"`
	VerificationCode    string                `json:"verificationcode,omitempty"`
	ConfirmPassword     data.HiddenJsonString `json:"confirmpassword,omitempty"`
}

type ChangePassword struct {
	OldPasword  string `json:"oldpassword"`
	NewPassword string `json:"newpassword"`
}

type ChangeTempPassword struct {
	Password        data.HiddenJsonString `json:"password"`
	ConfirmPassword string                `json:"confirmpassword"`
}

type Profession string
