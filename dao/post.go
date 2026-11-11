package dao

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/lib/pq"
	itn "github.com/mdhasib01/go-rest-starter/itn"
	model "github.com/mdhasib01/go-rest-starter/model"
	"github.com/mdhasib01/go-rest-starter/pkg/logger"
)

const postColumns = `
	p.id, p.author_id,
	CASE WHEN COALESCE(a.hide_personal_info, false) THEN ''
	     ELSE TRIM(COALESCE(a.first_name, '') || ' ' || COALESCE(a.last_name, '')) END,
	p.blood_group, p.title, COALESCE(p.description, ''), COALESCE(p.hospital_name, ''),
	COALESCE(p.address, ''), COALESCE(p.district, ''), COALESCE(p.division, ''),
	p.units_needed, COALESCE(p.contact_phone, ''), p.required_date, p.status,
	COALESCE((SELECT array_agg(i.user_id ORDER BY i.created_at)
	          FROM donation_post_interests i WHERE i.post_id = p.id), '{}'),
	p.created_at`

// posts of deleted / disabled authors are hidden
const postBaseWhere = `p.is_deleted = false AND a.is_available = true`

func scanPost(row interface{ Scan(...any) error }) (model.DonationPost, error) {
	var p model.DonationPost
	var required, created sql.NullTime
	var interested []int64
	err := row.Scan(
		&p.Id, &p.AuthorId, &p.AuthorName, &p.BloodGroup, &p.Title, &p.Description,
		&p.HospitalName, &p.Address, &p.District, &p.Division, &p.UnitsNeeded,
		&p.ContactPhone, &required, &p.Status, pq.Array(&interested), &created,
	)
	if required.Valid {
		p.RequiredDate = required.Time.Format("2006-01-02")
	}
	if created.Valid {
		p.CreatedAt = created.Time
	}
	p.InterestedUsers = make([]int, len(interested))
	for i, id := range interested {
		p.InterestedUsers[i] = int(id)
	}
	return p, err
}

func GetPosts(f model.PostFilter) (model.PaginatedResponse, error) {
	where := []string{postBaseWhere}
	args := []interface{}{}
	arg := func(v interface{}) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}

	if f.Status != "" {
		where = append(where, "p.status = "+arg(f.Status))
	}
	if f.BloodGroup != "" {
		where = append(where, "p.blood_group = "+arg(f.BloodGroup))
	}
	if f.Division != "" {
		where = append(where, "LOWER(p.division) = LOWER("+arg(f.Division)+")")
	}
	if f.District != "" {
		where = append(where, "LOWER(p.district) = LOWER("+arg(f.District)+")")
	}
	if s := strings.TrimSpace(f.Search); s != "" {
		k := arg("%" + escapeLike(s) + "%")
		where = append(where, fmt.Sprintf(
			"(p.title ILIKE %[1]s OR p.description ILIKE %[1]s OR p.hospital_name ILIKE %[1]s OR p.district ILIKE %[1]s)", k))
	}

	from := " FROM donation_posts p JOIN users a ON a.id = p.author_id WHERE " + strings.Join(where, " AND ")

	var count int
	if err := DB.QueryRow("SELECT COUNT(*)"+from, args...).Scan(&count); err != nil {
		logger.GetLogger().LogErrors(err, map[string]interface{}{"query": from})
		return model.PaginatedResponse{}, model.UnknownError
	}

	// open requests first, then newest
	query := "SELECT " + postColumns + from +
		" ORDER BY (p.status = 'open') DESC, p.created_at DESC, p.id DESC" +
		" LIMIT " + arg(f.Size) + " OFFSET " + arg((f.Page-1)*f.Size)

	rows, err := DB.Query(query, args...)
	if err != nil {
		logger.GetLogger().LogErrors(err, map[string]interface{}{"query": query})
		return model.PaginatedResponse{}, model.UnknownError
	}
	defer rows.Close()

	list := []model.DonationPost{}
	for rows.Next() {
		p, err := scanPost(rows)
		if err != nil {
			logger.GetLogger().LogErrors(err, nil)
			return model.PaginatedResponse{}, model.UnknownError
		}
		list = append(list, p)
	}
	if err := rows.Err(); err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return model.PaginatedResponse{}, model.UnknownError
	}

	return model.PaginatedResponse{Data: list, Count: count, Page: f.Page, Size: f.Size}, nil
}

func GetPost(id int) (model.DonationPost, error) {
	query := "SELECT " + postColumns +
		" FROM donation_posts p JOIN users a ON a.id = p.author_id WHERE p.id = $1 AND " + postBaseWhere

	p, err := scanPost(DB.QueryRow(query, id))
	if errors.Is(err, sql.ErrNoRows) {
		return model.DonationPost{}, model.NewError(itn.ErrorPostNotFound, 404)
	}
	if err != nil {
		logger.GetLogger().LogErrors(err, map[string]interface{}{"query": query})
		return model.DonationPost{}, model.UnknownError
	}
	return p, nil
}

// CreatePost inserts a post. When no contact phone is given, the
// author's profile phone is used.
func CreatePost(p model.DonationPost) (int, error) {
	query := `INSERT INTO donation_posts (
			author_id, blood_group, title, description, hospital_name, address,
			district, division, units_needed, contact_phone, required_date, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9,
			COALESCE(NULLIF($10, ''), (SELECT NULLIF(phone, '') FROM users WHERE id = $1)),
			$11::date, 'open')
		RETURNING id`

	var id int
	err := DB.QueryRow(query, p.AuthorId, p.BloodGroup, p.Title, p.Description, p.HospitalName,
		p.Address, p.District, p.Division, p.UnitsNeeded, p.ContactPhone, p.RequiredDate).Scan(&id)
	if err != nil {
		logger.GetLogger().LogErrors(err, map[string]interface{}{"query": query})
		return 0, model.UnknownError
	}
	return id, nil
}

// ToggleInterest adds the user's interest in an open post, or removes it
// when it already exists. It returns true when the user is now interested.
func ToggleInterest(postID, userID int) (bool, error) {
	tx, err := DB.Begin()
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return false, model.UnknownError
	}
	defer tx.Rollback()

	var authorID int
	var status string
	err = tx.QueryRow(`SELECT p.author_id, p.status
		FROM donation_posts p JOIN users a ON a.id = p.author_id
		WHERE p.id = $1 AND `+postBaseWhere+` FOR UPDATE OF p`, postID).Scan(&authorID, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return false, model.NewError(itn.ErrorPostNotFound, 404)
	}
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return false, model.UnknownError
	}
	if authorID == userID {
		return false, model.NewError(itn.ErrorPostOwnInterest, 400)
	}

	res, err := tx.Exec(`DELETE FROM donation_post_interests WHERE post_id = $1 AND user_id = $2`, postID, userID)
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return false, model.UnknownError
	}
	removed, _ := res.RowsAffected()

	interested := false
	if removed == 0 {
		// new interest is only allowed while the post is open;
		// withdrawing is always allowed
		if status != "open" {
			return false, model.NewError(itn.ErrorPostNotOpen, 409)
		}
		if _, err = tx.Exec(`INSERT INTO donation_post_interests (post_id, user_id) VALUES ($1, $2)
			ON CONFLICT (post_id, user_id) DO NOTHING`, postID, userID); err != nil {
			logger.GetLogger().LogErrors(err, nil)
			return false, model.UnknownError
		}
		interested = true
	}

	if err = tx.Commit(); err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return false, model.UnknownError
	}
	return interested, nil
}
