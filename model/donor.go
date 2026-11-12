package model

import "time"

// Donor is the public view of a user shown in the donor directory.
// It never carries email, password or verification data.
type Donor struct {
	Id               int       `json:"id"`
	Firstname        string    `json:"first_name,omitempty"`
	Lastname         string    `json:"last_name,omitempty"`
	Gender           string    `json:"gender,omitempty"`
	BloodGroup       string    `json:"blood_group,omitempty"`
	Age              int       `json:"age,omitempty"`
	Phone            string    `json:"phone,omitempty"`
	Address          string    `json:"address,omitempty"`
	City             string    `json:"city,omitempty"`
	District         string    `json:"district,omitempty"`
	Division         string    `json:"division,omitempty"`
	Upazila          string    `json:"upazila,omitempty"`
	Photo            string    `json:"photo,omitempty"`
	Points           int       `json:"points"`
	DonationCount    int       `json:"donation_count"`
	Profession       string    `json:"profession,omitempty"`
	DonatedAt        string    `json:"donated_at,omitempty"`
	HidePersonalInfo bool      `json:"hide_personal_info"`
	IsVerified       bool      `json:"is_verified"`
	CreatedAt        time.Time `json:"created_at"`
}

// DonorFilter holds the query parameters of the donor directory.
type DonorFilter struct {
	BloodGroup string
	Division   string
	District   string
	Upazila    string
	Gender     string // "male", "female", "other"/"others"
	Search     string
	Page       int
	Size       int
}

// IsValidBloodGroup reports whether bg is one of the 8 ABO/Rh groups.
func IsValidBloodGroup(bg string) bool {
	for _, g := range BLOOD_GROUPS {
		if string(g) == bg {
			return true
		}
	}
	return false
}
