package controller

import (
	"fmt"
	"strings"
	"time"

	dao "github.com/mdhasib01/go-rest-starter/dao"
	itn "github.com/mdhasib01/go-rest-starter/itn"
	model "github.com/mdhasib01/go-rest-starter/model"
)

const (
	defaultLeaderboardSize = 20
	maxLeaderboardSize     = 100
)

// GetInterestedDonors lists the donors interested in a post; only the
// post author can see it (it includes phone numbers).
func GetInterestedDonors(postID int, session model.Session) ([]model.InterestedDonor, error) {
	post, err := dao.GetPost(postID)
	if err != nil {
		return nil, err
	}
	if post.AuthorId != session.UserId {
		return nil, model.NewError(itn.ErrorNotPostAuthor, 403)
	}
	return dao.GetInterestedDonors(postID)
}

// ConfirmDonation lets a post author confirm that an interested donor
// donated. The donor earns points and badges.
func ConfirmDonation(postID int, req model.ConfirmDonation, session model.Session) (model.DonationConfirmation, error) {
	if req.DonorId < 1 || req.DonorId == session.UserId {
		return model.DonationConfirmation{}, model.NewError(itn.ErrorDonationNotInterested, 400)
	}
	if req.Units < 1 {
		req.Units = 1
	}
	if req.Units > maxUnitsPerPost {
		return model.DonationConfirmation{}, model.NewError(itn.ErrorInvalidData, 400)
	}

	date, err := parseDonationDate(req.DonationDate, time.Now())
	if err != nil {
		return model.DonationConfirmation{}, err
	}

	res, err := dao.ConfirmDonation(postID, session.UserId, req, date)
	if err != nil {
		return res, err
	}

	notify([]int{req.DonorId}, model.Notification{
		Kind:  model.NotifyDonationConfirmed,
		Title: "Donation confirmed — thank you, hero!",
		Body:  fmt.Sprintf("You earned %d points. Total: %d.", res.PointsAwarded, res.TotalPoints),
		Data:  map[string]any{"post_id": postID, "points": res.PointsAwarded, "total_points": res.TotalPoints},
	})
	for _, b := range res.NewBadges {
		notifyBadge(req.DonorId, b)
	}
	return res, nil
}

// parseDonationDate accepts YYYY-MM-DD (default today) up to a year back
// and not in the future.
func parseDonationDate(s string, now time.Time) (time.Time, error) {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	s = strings.TrimSpace(s)
	if s == "" {
		return today, nil
	}
	d, err := time.Parse("2006-01-02", s)
	if err != nil || d.After(today) || d.Before(today.AddDate(-1, 0, 0)) {
		return time.Time{}, model.NewError(itn.ErrorDonationDateInvalid, 400)
	}
	return d, nil
}

func notifyBadge(userID int, b model.Badge) {
	notify([]int{userID}, model.Notification{
		Kind:  model.NotifyBadgeEarned,
		Title: "Badge unlocked: " + b.Name,
		Body:  b.Description,
		Data:  map[string]any{"badge": b},
	})
}

func GetGamificationProfile(userID int) (model.GamificationProfile, error) {
	return dao.GetGamificationProfile(userID)
}

func GetBadges() ([]model.Badge, error) {
	return dao.GetBadges()
}

func GetLeaderboard(f model.LeaderboardFilter) ([]model.LeaderboardEntry, error) {
	if err := normalizeLeaderboardFilter(&f); err != nil {
		return nil, err
	}
	return dao.GetLeaderboard(f)
}

func normalizeLeaderboardFilter(f *model.LeaderboardFilter) error {
	f.Scope = strings.ToLower(strings.TrimSpace(f.Scope))
	f.Value = strings.TrimSpace(f.Value)
	if f.Scope == "" {
		f.Scope = "global"
	}
	if !contains(model.LeaderboardScopes, f.Scope) {
		return model.NewError(itn.ErrorInvalidData, 400)
	}
	switch f.Scope {
	case "division", "district", "upazila":
		if f.Value == "" {
			return model.NewError(itn.ErrorInvalidData, 400)
		}
	case "bank":
		if f.BankId < 1 {
			return model.NewError(itn.ErrorInvalidData, 400)
		}
	}
	if f.Period != "month" {
		f.Period = "all"
	}
	if f.Size < 1 {
		f.Size = defaultLeaderboardSize
	}
	if f.Size > maxLeaderboardSize {
		f.Size = maxLeaderboardSize
	}
	return nil
}
