package model

import "time"

type FilteredRequest struct {
	Search             string   `json:"search"`
	Page               int      `json:"page"`
	Size               int      `json:"size"`
	Plan               string   `json:"plan"`
	OrderParams        string   `json:"order_params"`
	AllowedOrderParams []string `json:"allowed_order_params"`
	Filters            string   `json:"filters"`
	AllowedFilters     []string `json:"allowed_filters"`
}

type PaginatedResponse struct {
	Data  interface{} `json:"data"`
	Count int         `json:"total_count"`
	Page  int         `json:"page"`
	Size  int         `json:"size"`
}

type Location struct {
	ID           int       `json:"id,omitempty"`
	Country      string    `json:"country,omitempty"`
	CountryGeoID int       `json:"country_geo_id,omitempty"`
	City         string    `json:"city,omitempty"`
	CityGeoID    int       `json:"city_geo_id,omitempty"`
	IsTrusted    bool      `json:"is_trusted"`
	TrustCode    string    `json:"trust_code,omitempty"`
	UserID       int       `json:"user_id,omitempty"`
	CreatedAt    time.Time `json:"created_at,omitempty"`
	UpdatedAt    time.Time `json:"updated_at,omitempty"`
	UpdatedBy    int       `json:"updated_by,omitempty"`
}

type BloodGropu string

var BLOOD_GROUPS = []BloodGropu{
	"O+",
	"O-",
	"A+",
	"A-",
	"B+",
	"B-",
	"AB+",
	"AB-",
}

type Division string

var DIVISIONS = []Division{
	"Dhaka",
	"Chattogram",
	"Khulna",
	"Rajshahi",
	"Sylhet",
	"Rangpur",
	"Barisal",
	"Mymensingh",
}

type District string

var DISTRICTS = map[Division][]District{
	"Dhaka": {
		"Dhaka", "Gazipur", "Narayanganj", "Tangail", "Manikganj",
		"Munshiganj", "Narsingdi", "Faridpur", "Gopalganj",
		"Kishoreganj", "Madaripur", "Rajbari", "Shariatpur",
	},
	"Chattogram": {
		"Chattogram", "Cox's Bazar", "Comilla", "Feni", "Brahmanbaria",
		"Noakhali", "Lakshmipur", "Chandpur", "Rangamati",
		"Khagrachari", "Bandarban",
	},
	"Khulna": {
		"Khulna", "Jessore", "Satkhira", "Bagerhat", "Narail", "Magura",
		"Jhenaidah", "Kushtia", "Chuadanga", "Meherpur",
	},
	"Rajshahi": {
		"Rajshahi", "Bogra", "Pabna", "Sirajganj", "Natore",
		"Naogaon", "Chapainawabganj", "Joypurhat",
	},
	"Sylhet": {
		"Sylhet", "Moulvibazar", "Habiganj", "Sunamganj",
	},
	"Rangpur": {
		"Rangpur", "Dinajpur", "Gaibandha", "Kurigram", "Lalmonirhat",
		"Nilphamari", "Panchagarh", "Thakurgaon",
	},
	"Barisal": {
		"Barisal", "Bhola", "Jhalokati", "Patuakhali",
		"Pirojpur", "Barguna",
	},
	"Mymensingh": {
		"Mymensingh", "Jamalpur", "Netrokona", "Sherpur",
	},
}
