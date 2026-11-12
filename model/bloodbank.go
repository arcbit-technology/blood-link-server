package model

import "time"

// Blood bank lifecycle
const (
	BankStatusPending   = "pending"
	BankStatusApproved  = "approved"
	BankStatusRejected  = "rejected"
	BankStatusSuspended = "suspended"
)

var BankStatuses = []string{BankStatusPending, BankStatusApproved, BankStatusRejected, BankStatusSuspended}

// Roles inside a blood bank. They are separate from the platform roles
// (SUPERADMIN, ADMIN, ...): any donor can be the admin of a blood bank.
const (
	BankRoleAdmin   = "admin"
	BankRoleOfficer = "officer"
	BankRoleMember  = "member"
)

// Membership lifecycle
const (
	MemberStatusPending  = "pending"
	MemberStatusActive   = "active"
	MemberStatusRejected = "rejected"
	MemberStatusLeft     = "left"
	MemberStatusRemoved  = "removed"
)

var MemberStatuses = []string{MemberStatusPending, MemberStatusActive, MemberStatusRejected, MemberStatusLeft, MemberStatusRemoved}

// BloodBank is a local, upazila-level group of donors.
type BloodBank struct {
	Id              int        `json:"id"`
	Name            string     `json:"name"`
	Description     string     `json:"description"`
	Division        string     `json:"division"`
	District        string     `json:"district"`
	Upazila         string     `json:"upazila"`
	Address         string     `json:"address"`
	ContactPhone    string     `json:"contact_phone"`
	Logo            string     `json:"logo"`
	Cover           string     `json:"cover"`
	Status          string     `json:"status"`
	AdminId         int        `json:"admin_id"`
	AdminName       string     `json:"admin_name"`
	CreatedBy       int        `json:"created_by"`
	CreatedByName   string     `json:"created_by_name"`
	ReviewedAt      *time.Time `json:"reviewed_at,omitempty"`
	RejectionReason string     `json:"rejection_reason,omitempty"`
	MemberCount     int        `json:"member_count"`
	DonationCount   int        `json:"donation_count"`
	TotalPoints     int        `json:"total_points"`
	CreatedAt       time.Time  `json:"created_at"`
}

// BloodBankFilter holds the query parameters of GET /blood-banks.
type BloodBankFilter struct {
	Division string
	District string
	Upazila  string
	Search   string
	Status   string
	Page     int
	Size     int
}

// BankMember is a donor's membership in a blood bank.
type BankMember struct {
	Id            int        `json:"id"`
	BloodBankId   int        `json:"blood_bank_id"`
	UserId        int        `json:"user_id"`
	FirstName     string     `json:"first_name"`
	LastName      string     `json:"last_name"`
	BloodGroup    string     `json:"blood_group"`
	Phone         string     `json:"phone,omitempty"`
	Upazila       string     `json:"upazila"`
	Role          string     `json:"role"`
	Position      string     `json:"position"`
	Status        string     `json:"status"`
	Points        int        `json:"points"`
	DonationCount int        `json:"donation_count"`
	DonatedAt     string     `json:"donated_at,omitempty"`
	Photo         string     `json:"photo,omitempty"`
	JoinedAt      *time.Time `json:"joined_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`

	HidePersonalInfo bool `json:"-"`
}

// MyBloodBank is the caller's relation to blood banks: their membership
// (pending or active) and/or a creation request waiting for review.
type MyBloodBank struct {
	Membership     *BankMember `json:"membership"`
	Bank           *BloodBank  `json:"bank"`
	PendingRequest *BloodBank  `json:"pending_request"`
}

// BankMemberRoleUpdate is the body of PUT /blood-bank/{id}/member/{userId}/role.
type BankMemberRoleUpdate struct {
	Role     string `json:"role"`
	Position string `json:"position"`
}

// BankReview is the body of the admin reject / suspend endpoints.
type BankReview struct {
	Reason string `json:"reason"`
}

// TransferAdmin is the body of PUT /blood-bank/{id}/transfer-admin.
type TransferAdmin struct {
	UserId int `json:"user_id"`
}
