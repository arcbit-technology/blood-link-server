package model

import "time"

type FilteredRequest struct {
	Search             string   `json:"search"`
	Page               int      `json:"page"`
	Size               int      `json:"size"`
	Plan               string   `json:"plan"`
	OrderParams        string   `json:"orderparams"`
	AllowedOrderParams []string `json:"allowedorderparams"`
	Filters            string   `json:"filters"`
	AllowedFilters     []string `json:"allowedfilters"`
}

type PaginatedResponse struct {
	Data  interface{} `json:"data"`
	Count int         `json:"total_count"`
	Page  int         `json:"page"`
	Size  int         `json:"size"`
}

type UserWithPlan struct {
	UserId    int    `json:"user_id"`
	Firstname string `json:"firstname"`
	Lastname  string `json:"lastname"`
	Email     string `json:"email"`
	Plan      string `json:"plan"`
}

type Location struct {
	ID           int       `json:"id,omitempty"`
	Country      string    `json:"country,omitempty"`
	CountryGeoID int       `json:"countryGeoID,omitempty"`
	City         string    `json:"city,omitempty"`
	CityGeoID    int       `json:"cityGeoID,omitempty"`
	IsTrusted    bool      `json:"isTrusted"`
	TrustCode    string    `json:"trustCode,omitempty"`
	UserID       int       `json:"userID,omitempty"`
	CreatedAt    time.Time `json:"createdAt,omitempty"`
	UpdatedAt    time.Time `json:"updatedAt,omitempty"`
	UpdatedBy    int       `json:"updatedBy,omitempty"`
}
