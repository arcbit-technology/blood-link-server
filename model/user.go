package model

import (
	"time"

	"github.com/mdhasib01/go-rest-starter/pkg/data"
)

type User struct {
	Id                  int                   `json:"id,omitempty"`
	Firstname           string                `json:"first_name,omitempty"`
	Lastname            string                `json:"last_name,omitempty"`
	Password            data.HiddenJsonString `json:"password,omitempty"`
	Gender              string                `json:"gender,omitempty"`
	BloodGroup          string                `json:"blood_group,omitempty"`
	Age                 int                   `json:"age,omitempty"`
	HidePersonalInfo    bool                  `json:"hide_personal_info,omitempty"`
	DonatedAt           string                `json:"donated_at,omitempty"`
	IsAvailable         bool                  `json:"is_available,omitempty"`
	Email               string                `json:"email,omitempty"`
	Phone               string                `json:"phone,omitempty"`
	Address             string                `json:"address,omitempty"`
	City                string                `json:"city,omitempty"`
	District            string                `json:"district,omitempty"`
	Division            string                `json:"division,omitempty"`
	Upazila             string                `json:"upazila,omitempty"`
	Photo               string                `json:"photo,omitempty"`
	Country             string                `json:"country,omitempty"`
	IsVerified          bool                  `json:"is_verified,omitempty"`
	LastLogin           string                `json:"last_login,omitempty"`
	CreatedAt           time.Time             `json:"created_at,omitempty"`
	UpdatedAt           time.Time             `json:"updated_at,omitempty"`
	CreatedBy           int                   `json:"created_by,omitempty"`
	UpdateBy            int                   `json:"updated_by,omitempty"`
	Profession          string                `json:"profession,omitempty"`
	IsPasswordTemporary bool                  `json:"is_password_temporary,omitempty"`
	RoleName            string                `json:"role,omitempty"`
	IdRole              int                   `json:"id_role,omitempty"`
	VerificationCode    string                `json:"verification_code,omitempty"`
	ConfirmPassword     data.HiddenJsonString `json:"confirm_password,omitempty"`
}

type ChangePassword struct {
	OldPasword  string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

type ChangeTempPassword struct {
	Password        data.HiddenJsonString `json:"password"`
	ConfirmPassword string                `json:"confirm_password"`
}

type Profession string
