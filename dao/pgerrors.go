package dao

import (
	"errors"

	"github.com/lib/pq"
)

// isUniqueViolation reports whether err is a unique-constraint violation,
// optionally on the named constraint / index.
func isUniqueViolation(err error, constraint string) bool {
	var pqErr *pq.Error
	if !errors.As(err, &pqErr) || pqErr.Code != "23505" {
		return false
	}
	return constraint == "" || pqErr.Constraint == constraint
}

// rowScanner is satisfied by *sql.Row and *sql.Rows.
type rowScanner interface{ Scan(...any) error }

// stringArray binds a []string as a Postgres text[] parameter.
func stringArray(s []string) interface{} { return pq.Array(s) }
