package controller

import (
	"testing"
	"time"

	model "github.com/mdhasib01/go-rest-starter/model"
	"github.com/stretchr/testify/assert"
)

func TestLevelFor(t *testing.T) {
	cases := []struct {
		points   int
		level    string
		nextName string
	}{
		{0, "bronze", "silver"},
		{299, "bronze", "silver"},
		{300, "silver", "gold"},
		{799, "silver", "gold"},
		{800, "gold", "platinum"},
		{2000, "platinum", ""},
		{9999, "platinum", ""},
	}
	for _, c := range cases {
		level, next := model.LevelFor(c.points)
		assert.Equal(t, c.level, level.Name, "points %d", c.points)
		if c.nextName == "" {
			assert.Nil(t, next, "points %d", c.points)
		} else if assert.NotNil(t, next, "points %d", c.points) {
			assert.Equal(t, c.nextName, next.Name, "points %d", c.points)
		}
	}
}

func TestParseDonationDate(t *testing.T) {
	now := time.Date(2026, 11, 12, 15, 0, 0, 0, time.UTC)
	today := time.Date(2026, 11, 12, 0, 0, 0, 0, time.UTC)

	d, err := parseDonationDate("", now)
	assert.NoError(t, err)
	assert.Equal(t, today, d)

	d, err = parseDonationDate("2026-10-01", now)
	assert.NoError(t, err)
	assert.Equal(t, time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), d)

	for _, bad := range []string{"2026-11-13", "2025-11-11", "12/11/2026", "nope"} {
		_, err = parseDonationDate(bad, now)
		assert.Error(t, err, bad)
	}
}

func TestNormalizeLeaderboardFilter(t *testing.T) {
	f := model.LeaderboardFilter{}
	assert.NoError(t, normalizeLeaderboardFilter(&f))
	assert.Equal(t, "global", f.Scope)
	assert.Equal(t, "all", f.Period)
	assert.Equal(t, defaultLeaderboardSize, f.Size)

	f = model.LeaderboardFilter{Scope: "District", Value: "Dhaka", Period: "month", Size: 1000}
	assert.NoError(t, normalizeLeaderboardFilter(&f))
	assert.Equal(t, "district", f.Scope)
	assert.Equal(t, maxLeaderboardSize, f.Size)

	assert.Error(t, normalizeLeaderboardFilter(&model.LeaderboardFilter{Scope: "upazila"}))
	assert.Error(t, normalizeLeaderboardFilter(&model.LeaderboardFilter{Scope: "bank"}))
	assert.Error(t, normalizeLeaderboardFilter(&model.LeaderboardFilter{Scope: "planet"}))
}

func TestValidateBloodBank(t *testing.T) {
	ok := model.BloodBank{Name: "  Savar Heroes ", Division: "Dhaka", District: "Dhaka", Upazila: "Savar"}
	assert.NoError(t, validateBloodBank(&ok, true))
	assert.Equal(t, "Savar Heroes", ok.Name)

	bad := []model.BloodBank{
		{Name: "ab", Division: "Dhaka", District: "Dhaka", Upazila: "Savar"},              // name too short
		{Name: "Savar Heroes", Division: "Khulna", District: "Dhaka", Upazila: "Savar"},   // district not in division
		{Name: "Savar Heroes", Division: "Dhaka", District: "Dhaka", Upazila: "Kaliganj"}, // upazila not in district
		{Name: "Savar Heroes", Division: "Dhaka", District: "Dhaka"},                      // upazila missing
	}
	for i := range bad {
		assert.Error(t, validateBloodBank(&bad[i], true), "case %d", i)
	}

	// edits do not touch the location
	edit := model.BloodBank{Name: "Renamed"}
	assert.NoError(t, validateBloodBank(&edit, false))
}

func TestUpazilaData(t *testing.T) {
	assert.Len(t, model.UPAZILAS, 64, "every district has upazilas")
	for division, districts := range model.DISTRICTS {
		for _, d := range districts {
			assert.NotEmpty(t, model.UPAZILAS[d], "district %s (%s) has no upazilas", d, division)
		}
	}
	assert.True(t, model.IsValidUpazila("Gazipur", "Kaliganj"))
	assert.False(t, model.IsValidUpazila("Dhaka", "Kaliganj"))
}
