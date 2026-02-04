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
