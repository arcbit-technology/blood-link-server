package model

import (
	"database/sql"
	"time"

	"github.com/lib/pq"
)

type Role struct {
	Id          int          `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	CreatedAt   time.Time    `json:"createdAt"`
	UpdatedAt   time.Time    `json:"updatedAt"`
	CreatedBy   int          `json:"createdBy"`
	UpdatedBy   int          `json:"updatedBy"`
	IsDeleted   bool         `json:"isdeleted"`
	Permissions []Permission `json:"permission"`
	UserCreate  string       `json:"-"`
}

type NullRole struct {
	Id          sql.NullInt64
	Name        sql.NullString
	Description sql.NullString
	CreatedAt   pq.NullTime
	UpdatedAt   pq.NullTime
	CreatedBy   sql.NullInt64
	UpdatedBy   sql.NullInt64
	UserCreate  sql.NullString
}

type Permission struct {
	Id             int    `json:"id"`
	Name           string `json:"name"`
	SuperAdminOnly bool   `json:"-"`
	Description    string `json:"description"`
}
