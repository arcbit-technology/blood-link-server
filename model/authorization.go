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
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	CreatedBy   int          `json:"created_by"`
	UpdatedBy   int          `json:"updated_by"`
	IsDeleted   bool         `json:"is_deleted"`
	Permissions []Permission `json:"permissions"`
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
