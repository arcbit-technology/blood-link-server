package model

import "time"

// DonationPost is a public "blood needed" request shown on /posts.
type DonationPost struct {
	Id              int       `json:"id"`
	AuthorId        int       `json:"author_id"`
	AuthorName      string    `json:"author_name"` // empty when the author hides personal info
	BloodGroup      string    `json:"blood_group"`
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	HospitalName    string    `json:"hospital_name"`
	Address         string    `json:"address"`
	District        string    `json:"district"`
	Division        string    `json:"division"`
	UnitsNeeded     int       `json:"units_needed"`
	ContactPhone    string    `json:"contact_phone,omitempty"`
	RequiredDate    string    `json:"required_date"` // YYYY-MM-DD
	Status          string    `json:"status"`        // open | fulfilled | closed
	InterestedUsers []int     `json:"interested_users"`
	CreatedAt       time.Time `json:"created_at"`
}

// PostFilter holds the query parameters of GET /posts.
type PostFilter struct {
	Status     string
	BloodGroup string
	Division   string
	District   string
	Search     string
	Page       int
	Size       int
}

var PostStatuses = []string{"open", "fulfilled", "closed"}
