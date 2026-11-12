package model

import "time"

// Points awarded for a confirmed donation.
const (
	PointsPerDonation      = 100
	PointsFirstDonation    = 50
	PointsEmergencyBonus   = 50
	PointsBankFounder      = 200
	MinDaysBetweenDonation = 90
)

// Point ledger reasons
const (
	PointReasonDonation       = "donation"
	PointReasonFirstDonation  = "first_donation"
	PointReasonEmergencyBonus = "emergency_bonus"
	PointReasonBankFounder    = "bank_founder"
)

// Badge codes (seeded in migration 000003)
const (
	BadgeFirstDrop          = "first_drop"
	BadgeLifeSaver          = "life_saver"
	BadgeHero               = "hero"
	BadgeSuperHero          = "super_hero"
	BadgeLegend             = "legend"
	BadgeEmergencyResponder = "emergency_responder"
	BadgeBankFounder        = "bank_founder"
	BadgeTopDonorMonth      = "top_donor_month"
)

// Level is a points band.
type Level struct {
	Name      string `json:"name"`
	MinPoints int    `json:"min_points"`
}

// LEVELS is ordered from lowest to highest.
var LEVELS = []Level{
	{Name: "bronze", MinPoints: 0},
	{Name: "silver", MinPoints: 300},
	{Name: "gold", MinPoints: 800},
	{Name: "platinum", MinPoints: 2000},
}

// LevelFor returns the level for the points and the next level, if any.
func LevelFor(points int) (Level, *Level) {
	current := LEVELS[0]
	for i, l := range LEVELS {
		if points >= l.MinPoints {
			current = l
			continue
		}
		next := LEVELS[i]
		return current, &next
	}
	return current, nil
}

type Badge struct {
	Id          int    `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Kind        string `json:"kind"`
	Threshold   int    `json:"threshold"`
}

type UserBadge struct {
	Badge
	BloodBankId int       `json:"blood_bank_id,omitempty"`
	Period      string    `json:"period,omitempty"`
	AwardedAt   time.Time `json:"awarded_at"`
}

type PointTransaction struct {
	Id        int       `json:"id"`
	Points    int       `json:"points"`
	Reason    string    `json:"reason"`
	RefType   string    `json:"ref_type,omitempty"`
	RefId     int       `json:"ref_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// GamificationProfile is the response of GET /user/{id}/gamification.
type GamificationProfile struct {
	UserId        int                `json:"user_id"`
	Points        int                `json:"points"`
	DonationCount int                `json:"donation_count"`
	Level         Level              `json:"level"`
	NextLevel     *Level             `json:"next_level"`
	Rank          int                `json:"rank"`
	Badges        []UserBadge        `json:"badges"`
	AllBadges     []Badge            `json:"all_badges"`
	Recent        []PointTransaction `json:"recent"`
}

type LeaderboardEntry struct {
	Rank          int    `json:"rank"`
	UserId        int    `json:"user_id"`
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
	Photo         string `json:"photo,omitempty"`
	BloodGroup    string `json:"blood_group"`
	District      string `json:"district"`
	Upazila       string `json:"upazila"`
	Points        int    `json:"points"`
	Donations     int    `json:"donations"`
	BloodBankId   int    `json:"blood_bank_id,omitempty"`
	BloodBankName string `json:"blood_bank_name,omitempty"`
}

var LeaderboardScopes = []string{"global", "division", "district", "upazila", "bank"}

// LeaderboardFilter holds the query parameters of GET /leaderboard.
type LeaderboardFilter struct {
	Scope  string // global | division | district | upazila | bank
	Value  string // division / district / upazila name for those scopes
	BankId int
	Period string // all | month
	Size   int
}

// ConfirmDonation is the body of POST /post/{id}/confirm-donation.
type ConfirmDonation struct {
	DonorId      int    `json:"donor_id"`
	Units        int    `json:"units"`
	DonationDate string `json:"donation_date"` // YYYY-MM-DD, defaults to today
}

// DonationConfirmation is the result of a confirmed donation.
type DonationConfirmation struct {
	HistoryId     int     `json:"history_id"`
	DonorId       int     `json:"donor_id"`
	PointsAwarded int     `json:"points_awarded"`
	TotalPoints   int     `json:"total_points"`
	NewBadges     []Badge `json:"new_badges"`
}

// InterestedDonor is a donor who tapped "interested" on a post, as seen
// by the post author.
type InterestedDonor struct {
	UserId        int       `json:"user_id"`
	FirstName     string    `json:"first_name"`
	LastName      string    `json:"last_name"`
	BloodGroup    string    `json:"blood_group"`
	Phone         string    `json:"phone"`
	Photo         string    `json:"photo,omitempty"`
	District      string    `json:"district"`
	Upazila       string    `json:"upazila"`
	DonatedAt     string    `json:"donated_at,omitempty"`
	Points        int       `json:"points"`
	DonationCount int       `json:"donation_count"`
	BloodBankName string    `json:"blood_bank_name,omitempty"`
	Eligible      bool      `json:"eligible"`  // 90 days since last donation
	Confirmed     bool      `json:"confirmed"` // already confirmed for this post
	InterestedAt  time.Time `json:"interested_at"`
}
