package dao

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	itn "github.com/mdhasib01/go-rest-starter/itn"
	model "github.com/mdhasib01/go-rest-starter/model"
	"github.com/mdhasib01/go-rest-starter/pkg/logger"
)

// awardPoints adds a ledger row and updates the cached users.points.
func awardPoints(tx *sql.Tx, userID, points int, reason, refType string, refID int) error {
	if _, err := tx.Exec(`INSERT INTO point_transactions (user_id, points, reason, ref_type, ref_id)
		VALUES ($1, $2, $3, $4, $5)`, userID, points, reason, refType, refID); err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return model.UnknownError
	}
	if _, err := tx.Exec(`UPDATE users SET points = points + $2 WHERE id = $1`, userID, points); err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return model.UnknownError
	}
	return nil
}

// awardBadge gives the badge to the user unless they already have it
// (for the period). It returns the badge when it was newly awarded.
func awardBadge(tx *sql.Tx, userID int, code string, bankID int, period string) (*model.Badge, error) {
	var b model.Badge
	err := tx.QueryRow(`INSERT INTO user_badges (user_id, badge_id, blood_bank_id, period)
		SELECT $1, b.id, NULLIF($3, 0), $4 FROM badges b WHERE b.code = $2
		ON CONFLICT (user_id, badge_id, period) DO NOTHING
		RETURNING badge_id`, userID, code, bankID, period).Scan(&b.Id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil // already had it
	}
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return nil, model.UnknownError
	}
	if err := tx.QueryRow(`SELECT code, name, description, icon, kind, threshold FROM badges WHERE id = $1`, b.Id).
		Scan(&b.Code, &b.Name, &b.Description, &b.Icon, &b.Kind, &b.Threshold); err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return nil, model.UnknownError
	}
	return &b, nil
}

// GetInterestedDonors lists the users interested in a post, for its author.
func GetInterestedDonors(postID int) ([]model.InterestedDonor, error) {
	rows, err := DB.Query(`SELECT u.id, COALESCE(u.first_name, ''), COALESCE(u.last_name, ''),
			COALESCE(u.blood_group, ''), COALESCE(u.phone, ''), COALESCE(u.district, ''),
			COALESCE(u.upazila, ''), u.donated_at, u.points, u.donation_count,
			COALESCE(b.name, ''),
			EXISTS (SELECT 1 FROM donation_history h WHERE h.post_id = i.post_id AND h.donor_id = u.id),
			i.created_at, COALESCE(u.photo, '')
		FROM donation_post_interests i
		JOIN users u ON u.id = i.user_id AND u.is_available = true
		LEFT JOIN blood_bank_members m ON m.user_id = u.id AND m.status = 'active'
		LEFT JOIN blood_banks b ON b.id = m.blood_bank_id AND b.status = 'approved'
		WHERE i.post_id = $1
		ORDER BY i.created_at`, postID)
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return nil, model.UnknownError
	}
	defer rows.Close()

	list := []model.InterestedDonor{}
	for rows.Next() {
		var d model.InterestedDonor
		var donated, interested sql.NullTime
		if err := rows.Scan(&d.UserId, &d.FirstName, &d.LastName, &d.BloodGroup, &d.Phone,
			&d.District, &d.Upazila, &donated, &d.Points, &d.DonationCount, &d.BloodBankName,
			&d.Confirmed, &interested, &d.Photo); err != nil {
			logger.GetLogger().LogErrors(err, nil)
			return nil, model.UnknownError
		}
		d.Eligible = true
		if donated.Valid {
			d.DonatedAt = donated.Time.Format("2006-01-02")
			d.Eligible = time.Since(donated.Time) >= model.MinDaysBetweenDonation*24*time.Hour
		}
		if interested.Valid {
			d.InterestedAt = interested.Time
		}
		list = append(list, d)
	}
	return list, rows.Err()
}

// ConfirmDonation records that donorID donated for the post, awards
// points and milestone badges and, once enough units are confirmed,
// marks the post fulfilled. authorID must be the post's author.
func ConfirmDonation(postID, authorID int, req model.ConfirmDonation, donationDate time.Time) (model.DonationConfirmation, error) {
	var res model.DonationConfirmation

	tx, err := DB.Begin()
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return res, model.UnknownError
	}
	defer tx.Rollback()

	var postAuthor, unitsNeeded int
	var bloodGroup, hospital, district, division, urgency, authorName string
	err = tx.QueryRow(`SELECT p.author_id, p.units_needed, p.blood_group, COALESCE(p.hospital_name, ''),
			COALESCE(p.district, ''), COALESCE(p.division, ''), p.urgency,
			TRIM(COALESCE(a.first_name, '') || ' ' || COALESCE(a.last_name, ''))
		FROM donation_posts p JOIN users a ON a.id = p.author_id
		WHERE p.id = $1 AND `+postBaseWhere+` FOR UPDATE OF p`, postID).
		Scan(&postAuthor, &unitsNeeded, &bloodGroup, &hospital, &district, &division, &urgency, &authorName)
	if errors.Is(err, sql.ErrNoRows) {
		return res, model.NewError(itn.ErrorPostNotFound, 404)
	}
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return res, model.UnknownError
	}
	if postAuthor != authorID {
		return res, model.NewError(itn.ErrorNotPostAuthor, 403)
	}

	var interested bool
	if err = tx.QueryRow(`SELECT EXISTS (SELECT 1 FROM donation_post_interests WHERE post_id = $1 AND user_id = $2)`,
		postID, req.DonorId).Scan(&interested); err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return res, model.UnknownError
	}
	if !interested {
		return res, model.NewError(itn.ErrorDonationNotInterested, 400)
	}

	var donatedAt sql.NullTime
	var donationCount int
	err = tx.QueryRow(`SELECT donated_at, donation_count FROM users WHERE id = $1 AND is_available = true FOR UPDATE`,
		req.DonorId).Scan(&donatedAt, &donationCount)
	if errors.Is(err, sql.ErrNoRows) {
		return res, model.NewError(itn.ErrorUserNotFound, 404)
	}
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return res, model.UnknownError
	}

	var already bool
	if err = tx.QueryRow(`SELECT EXISTS (SELECT 1 FROM donation_history WHERE post_id = $1 AND donor_id = $2)`,
		postID, req.DonorId).Scan(&already); err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return res, model.UnknownError
	}
	if already {
		return res, model.NewError(itn.ErrorDonationAlreadyConfirmed, 409)
	}

	if donatedAt.Valid {
		gap := donationDate.Sub(donatedAt.Time.Truncate(24 * time.Hour))
		if gap < 0 {
			gap = -gap
		}
		if gap < model.MinDaysBetweenDonation*24*time.Hour {
			return res, model.NewError(itn.ErrorDonationTooSoon, 409)
		}
	}

	var bankID sql.NullInt64
	err = tx.QueryRow(`SELECT m.blood_bank_id FROM blood_bank_members m
		JOIN blood_banks b ON b.id = m.blood_bank_id AND b.status = 'approved'
		WHERE m.user_id = $1 AND m.status = 'active'`, req.DonorId).Scan(&bankID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		logger.GetLogger().LogErrors(err, nil)
		return res, model.UnknownError
	}

	err = tx.QueryRow(`INSERT INTO donation_history (donor_id, recipient_id, recipient_name, post_id,
			blood_group, units, hospital_name, district, division, donation_date, blood_bank_id, confirmed_by)
		VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''), NULLIF($8, ''), NULLIF($9, ''), $10, $11, $2)
		RETURNING id`,
		req.DonorId, authorID, authorName, postID, bloodGroup, req.Units, hospital, district, division,
		donationDate, bankID).Scan(&res.HistoryId)
	if isUniqueViolation(err, "ux_donation_history_post_donor") {
		return res, model.NewError(itn.ErrorDonationAlreadyConfirmed, 409)
	}
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return res, model.UnknownError
	}

	type award struct {
		points int
		reason string
	}
	awards := []award{{model.PointsPerDonation, model.PointReasonDonation}}
	if donationCount == 0 {
		awards = append(awards, award{model.PointsFirstDonation, model.PointReasonFirstDonation})
	}
	if urgency == "emergency" {
		awards = append(awards, award{model.PointsEmergencyBonus, model.PointReasonEmergencyBonus})
	}
	for _, a := range awards {
		if err = awardPoints(tx, req.DonorId, a.points, a.reason, "donation_history", res.HistoryId); err != nil {
			return res, err
		}
		res.PointsAwarded += a.points
	}

	if err = tx.QueryRow(`UPDATE users SET donation_count = donation_count + 1,
			donated_at = GREATEST(COALESCE(donated_at, $2::timestamp), $2::timestamp)
		WHERE id = $1 RETURNING points, donation_count`, req.DonorId, donationDate).
		Scan(&res.TotalPoints, &donationCount); err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return res, model.UnknownError
	}
	if _, err = tx.Exec(`UPDATE donation_history SET points_awarded = $2 WHERE id = $1`,
		res.HistoryId, res.PointsAwarded); err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return res, model.UnknownError
	}

	// milestone badges reached with this donation
	codes, err := txStrings(tx, `SELECT code FROM badges WHERE kind = 'milestone' AND threshold <= $1 ORDER BY threshold`, donationCount)
	if err != nil {
		return res, err
	}
	if urgency == "emergency" {
		codes = append(codes, model.BadgeEmergencyResponder)
	}
	res.NewBadges = []model.Badge{}
	for _, code := range codes {
		b, err := awardBadge(tx, req.DonorId, code, 0, "")
		if err != nil {
			return res, err
		}
		if b != nil {
			res.NewBadges = append(res.NewBadges, *b)
		}
	}

	// enough units confirmed: the request is fulfilled
	if _, err = tx.Exec(`UPDATE donation_posts SET status = 'fulfilled', updated_at = now()
		WHERE id = $1 AND status = 'open'
		  AND (SELECT COALESCE(SUM(units), 0) FROM donation_history WHERE post_id = $1) >= units_needed`, postID); err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return res, model.UnknownError
	}

	if err = tx.Commit(); err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return res, model.UnknownError
	}
	res.DonorId = req.DonorId
	return res, nil
}

func txStrings(tx *sql.Tx, query string, args ...interface{}) ([]string, error) {
	rows, err := tx.Query(query, args...)
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return nil, model.UnknownError
	}
	defer rows.Close()
	list := []string{}
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			logger.GetLogger().LogErrors(err, nil)
			return nil, model.UnknownError
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

// GetGamificationProfile returns points, badges, rank and recent points.
func GetGamificationProfile(userID int) (model.GamificationProfile, error) {
	p := model.GamificationProfile{UserId: userID}

	err := DB.QueryRow(`SELECT u.points, u.donation_count,
			1 + (SELECT COUNT(*) FROM users o WHERE o.is_available = true AND o.points > u.points)
		FROM users u WHERE u.id = $1 AND u.is_available = true`, userID).Scan(&p.Points, &p.DonationCount, &p.Rank)
	if errors.Is(err, sql.ErrNoRows) {
		return p, model.NewError(itn.ErrorUserNotFound, 404)
	}
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return p, model.UnknownError
	}
	p.Level, p.NextLevel = model.LevelFor(p.Points)

	if p.AllBadges, err = GetBadges(); err != nil {
		return p, err
	}

	rows, err := DB.Query(`SELECT b.id, b.code, b.name, b.description, b.icon, b.kind, b.threshold,
			COALESCE(ub.blood_bank_id, 0), ub.period, ub.awarded_at
		FROM user_badges ub JOIN badges b ON b.id = ub.badge_id
		WHERE ub.user_id = $1 ORDER BY ub.awarded_at DESC, b.sort_order`, userID)
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return p, model.UnknownError
	}
	defer rows.Close()
	p.Badges = []model.UserBadge{}
	for rows.Next() {
		var ub model.UserBadge
		if err := rows.Scan(&ub.Id, &ub.Code, &ub.Name, &ub.Description, &ub.Icon, &ub.Kind, &ub.Threshold,
			&ub.BloodBankId, &ub.Period, &ub.AwardedAt); err != nil {
			logger.GetLogger().LogErrors(err, nil)
			return p, model.UnknownError
		}
		p.Badges = append(p.Badges, ub)
	}
	if err := rows.Err(); err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return p, model.UnknownError
	}

	trows, err := DB.Query(`SELECT id, points, reason, COALESCE(ref_type, ''), COALESCE(ref_id, 0), created_at
		FROM point_transactions WHERE user_id = $1 ORDER BY created_at DESC, id DESC LIMIT 10`, userID)
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return p, model.UnknownError
	}
	defer trows.Close()
	p.Recent = []model.PointTransaction{}
	for trows.Next() {
		var t model.PointTransaction
		if err := trows.Scan(&t.Id, &t.Points, &t.Reason, &t.RefType, &t.RefId, &t.CreatedAt); err != nil {
			logger.GetLogger().LogErrors(err, nil)
			return p, model.UnknownError
		}
		p.Recent = append(p.Recent, t)
	}
	return p, trows.Err()
}

func GetBadges() ([]model.Badge, error) {
	rows, err := DB.Query(`SELECT id, code, name, description, icon, kind, threshold FROM badges ORDER BY sort_order, id`)
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return nil, model.UnknownError
	}
	defer rows.Close()
	list := []model.Badge{}
	for rows.Next() {
		var b model.Badge
		if err := rows.Scan(&b.Id, &b.Code, &b.Name, &b.Description, &b.Icon, &b.Kind, &b.Threshold); err != nil {
			logger.GetLogger().LogErrors(err, nil)
			return nil, model.UnknownError
		}
		list = append(list, b)
	}
	return list, rows.Err()
}

// GetLeaderboard ranks users by points earned (all time or this month)
// within a location or blood bank. Names of users who hide their
// personal info are blanked.
func GetLeaderboard(f model.LeaderboardFilter) ([]model.LeaderboardEntry, error) {
	where := []string{"u.is_available = true"}
	args := []interface{}{}
	arg := func(v interface{}) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}

	if f.Period == "month" {
		where = append(where, "t.created_at >= date_trunc('month', now())")
	}
	switch f.Scope {
	case "division":
		where = append(where, "LOWER(u.division) = LOWER("+arg(f.Value)+")")
	case "district":
		where = append(where, "LOWER(u.district) = LOWER("+arg(f.Value)+")")
	case "upazila":
		where = append(where, "LOWER(u.upazila) = LOWER("+arg(f.Value)+")")
	case "bank":
		where = append(where, "m.blood_bank_id = "+arg(f.BankId))
	}

	query := `SELECT RANK() OVER (ORDER BY SUM(t.points) DESC)::int, u.id,
			CASE WHEN COALESCE(u.hide_personal_info, false) THEN '' ELSE COALESCE(u.first_name, '') END,
			CASE WHEN COALESCE(u.hide_personal_info, false) THEN '' ELSE COALESCE(u.last_name, '') END,
			COALESCE(u.blood_group, ''), COALESCE(u.district, ''), COALESCE(u.upazila, ''),
			SUM(t.points)::int, (COUNT(*) FILTER (WHERE t.reason = 'donation'))::int,
			COALESCE(b.id, 0), COALESCE(b.name, ''),
			CASE WHEN COALESCE(u.hide_personal_info, false) THEN '' ELSE COALESCE(u.photo, '') END
		FROM point_transactions t
		JOIN users u ON u.id = t.user_id
		LEFT JOIN blood_bank_members m ON m.user_id = u.id AND m.status = 'active'
		LEFT JOIN blood_banks b ON b.id = m.blood_bank_id AND b.status = 'approved'
		WHERE ` + strings.Join(where, " AND ") + `
		GROUP BY u.id, b.id
		HAVING SUM(t.points) > 0
		ORDER BY SUM(t.points) DESC, u.id
		LIMIT ` + arg(f.Size)

	rows, err := DB.Query(query, args...)
	if err != nil {
		logger.GetLogger().LogErrors(err, map[string]interface{}{"query": query})
		return nil, model.UnknownError
	}
	defer rows.Close()

	list := []model.LeaderboardEntry{}
	for rows.Next() {
		var e model.LeaderboardEntry
		if err := rows.Scan(&e.Rank, &e.UserId, &e.FirstName, &e.LastName, &e.BloodGroup, &e.District,
			&e.Upazila, &e.Points, &e.Donations, &e.BloodBankId, &e.BloodBankName, &e.Photo); err != nil {
			logger.GetLogger().LogErrors(err, nil)
			return nil, model.UnknownError
		}
		list = append(list, e)
	}
	return list, rows.Err()
}

// TopDonorWinner is a user who won the monthly top-donor badge.
type TopDonorWinner struct {
	UserId int
	BankId int
	Badge  model.Badge
}

// AwardTopDonorsOfMonth gives the top_donor_month badge for the month
// that starts at monthStart to the donor with the most donation points
// in each approved blood bank. It is idempotent and returns only the
// newly awarded winners.
func AwardTopDonorsOfMonth(monthStart time.Time) ([]TopDonorWinner, error) {
	period := monthStart.Format("2006-01")
	monthEnd := monthStart.AddDate(0, 1, 0)

	rows, err := DB.Query(`SELECT DISTINCT ON (h.blood_bank_id) h.blood_bank_id, h.donor_id
		FROM donation_history h
		JOIN blood_banks b ON b.id = h.blood_bank_id AND b.status = 'approved'
		WHERE h.donation_date >= $1 AND h.donation_date < $2
		GROUP BY h.blood_bank_id, h.donor_id
		ORDER BY h.blood_bank_id, SUM(h.points_awarded) DESC, COUNT(*) DESC, MIN(h.created_at) ASC`,
		monthStart, monthEnd)
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return nil, model.UnknownError
	}
	type pair struct{ bank, user int }
	pairs := []pair{}
	for rows.Next() {
		var p pair
		if err := rows.Scan(&p.bank, &p.user); err != nil {
			rows.Close()
			logger.GetLogger().LogErrors(err, nil)
			return nil, model.UnknownError
		}
		pairs = append(pairs, p)
	}
	rows.Close()

	winners := []TopDonorWinner{}
	for _, p := range pairs {
		tx, err := DB.Begin()
		if err != nil {
			logger.GetLogger().LogErrors(err, nil)
			return winners, model.UnknownError
		}
		b, err := awardBadge(tx, p.user, model.BadgeTopDonorMonth, p.bank, period)
		if err != nil {
			tx.Rollback()
			return winners, err
		}
		if err := tx.Commit(); err != nil {
			logger.GetLogger().LogErrors(err, nil)
			return winners, model.UnknownError
		}
		if b != nil {
			winners = append(winners, TopDonorWinner{UserId: p.user, BankId: p.bank, Badge: *b})
		}
	}
	return winners, nil
}
