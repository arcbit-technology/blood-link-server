package dao

import (
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Runs only against a disposable, migrated database:
//
//	BLOODLINK_TEST_DB=postgres://postgres:postgres@localhost:5432/bloodlink_scratch?sslmode=disable go test ./dao/
func openTestDB(t *testing.T) {
	conn := os.Getenv("BLOODLINK_TEST_DB")
	if conn == "" {
		t.Skip("BLOODLINK_TEST_DB not set")
	}
	db, err := sql.Open("postgres", conn)
	require.NoError(t, err)
	require.NoError(t, db.Ping())
	DB = db
	t.Cleanup(func() { db.Close() })
}

func TestAwardTopDonorsOfMonthIsIdempotent(t *testing.T) {
	openTestDB(t)

	// a bank with one donation this month
	var bankID, donorID int
	err := DB.QueryRow(`SELECT h.blood_bank_id, h.donor_id FROM donation_history h
		JOIN blood_banks b ON b.id = h.blood_bank_id AND b.status = 'approved'
		WHERE h.donation_date >= date_trunc('month', now()) LIMIT 1`).Scan(&bankID, &donorID)
	if err == sql.ErrNoRows {
		t.Skip("no bank donation this month in the test database")
	}
	require.NoError(t, err)

	now := time.Now().UTC()
	month := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	first, err := AwardTopDonorsOfMonth(month)
	require.NoError(t, err)
	second, err := AwardTopDonorsOfMonth(month)
	require.NoError(t, err)
	assert.Empty(t, second, "second run awards nothing new")

	var count int
	require.NoError(t, DB.QueryRow(`SELECT COUNT(*) FROM user_badges ub JOIN badges b ON b.id = ub.badge_id
		WHERE b.code = 'top_donor_month' AND ub.blood_bank_id = $1 AND ub.period = $2`,
		bankID, month.Format("2006-01")).Scan(&count))
	assert.Equal(t, 1, count, "exactly one top donor per bank per month")
	_ = first
}
