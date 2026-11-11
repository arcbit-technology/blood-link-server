package dao

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	itn "github.com/mdhasib01/go-rest-starter/itn"
	model "github.com/mdhasib01/go-rest-starter/model"
	"github.com/mdhasib01/go-rest-starter/pkg/logger"
)

const donorColumns = `
	u.id,
	COALESCE(u.first_name, ''), COALESCE(u.last_name, ''), COALESCE(u.gender, ''),
	COALESCE(u.blood_group, ''),
	COALESCE(date_part('year', age(u.date_of_birth))::int, 0),
	COALESCE(u.phone, ''), COALESCE(u.address, ''), COALESCE(u.city, ''),
	COALESCE(u.district, ''), COALESCE(u.division, ''), COALESCE(u.profession, ''),
	u.donated_at,
	COALESCE(u.hide_personal_info, false), COALESCE(u.is_verified, false),
	u.created_at`

// donors are active, non-admin accounts
var donorBaseWhere = fmt.Sprintf(`u.is_available = TRUE AND u.role_id NOT IN (%d, %d)`,
	model.SUPERADMIN, model.ADMIN)

func scanDonor(row interface{ Scan(...any) error }) (model.Donor, error) {
	var d model.Donor
	var donatedAt, createdAt sql.NullTime
	err := row.Scan(
		&d.Id, &d.Firstname, &d.Lastname, &d.Gender, &d.BloodGroup, &d.Age,
		&d.Phone, &d.Address, &d.City, &d.District, &d.Division, &d.Profession,
		&donatedAt, &d.HidePersonalInfo, &d.IsVerified, &createdAt,
	)
	if donatedAt.Valid {
		d.DonatedAt = donatedAt.Time.Format("2006-01-02")
	}
	if createdAt.Valid {
		d.CreatedAt = createdAt.Time
	}
	return d, err
}

func GetDonors(f model.DonorFilter) (model.PaginatedResponse, error) {
	where := []string{donorBaseWhere}
	args := []interface{}{}
	arg := func(v interface{}) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}

	if f.BloodGroup != "" {
		where = append(where, "u.blood_group = "+arg(f.BloodGroup))
	}
	if f.Division != "" {
		where = append(where, "LOWER(u.division) = LOWER("+arg(f.Division)+")")
	}
	if f.District != "" {
		where = append(where, "LOWER(u.district) = LOWER("+arg(f.District)+")")
	}
	switch strings.ToLower(f.Gender) {
	case "male", "m":
		where = append(where, "UPPER(u.gender) IN ('M', 'MALE')")
	case "female", "f":
		where = append(where, "UPPER(u.gender) IN ('F', 'FEMALE')")
	case "other", "others":
		where = append(where, "UPPER(COALESCE(u.gender, '')) NOT IN ('M', 'MALE', 'F', 'FEMALE')")
	}
	if s := strings.TrimSpace(f.Search); s != "" {
		p := arg("%" + escapeLike(s) + "%")
		// names of donors who hide their personal info are not searchable
		where = append(where, fmt.Sprintf(`(
			u.district ILIKE %[1]s OR u.city ILIKE %[1]s OR u.division ILIKE %[1]s
			OR u.blood_group ILIKE %[1]s OR u.profession ILIKE %[1]s
			OR (NOT COALESCE(u.hide_personal_info, false)
			    AND (COALESCE(u.first_name, '') || ' ' || COALESCE(u.last_name, '')) ILIKE %[1]s)
		)`, p))
	}

	whereSQL := strings.Join(where, " AND ")

	var count int
	countQuery := "SELECT COUNT(*) FROM users u WHERE " + whereSQL
	if err := DB.QueryRow(countQuery, args...).Scan(&count); err != nil {
		logger.GetLogger().LogErrors(err, map[string]interface{}{"query": countQuery})
		return model.PaginatedResponse{}, model.UnknownError
	}

	query := "SELECT " + donorColumns + " FROM users u WHERE " + whereSQL +
		" ORDER BY u.created_at DESC, u.id DESC" +
		" LIMIT " + arg(f.Size) + " OFFSET " + arg((f.Page-1)*f.Size)

	rows, err := DB.Query(query, args...)
	if err != nil {
		logger.GetLogger().LogErrors(err, map[string]interface{}{"query": query})
		return model.PaginatedResponse{}, model.UnknownError
	}
	defer rows.Close()

	list := []model.Donor{}
	for rows.Next() {
		d, err := scanDonor(rows)
		if err != nil {
			logger.GetLogger().LogErrors(err, nil)
			return model.PaginatedResponse{}, model.UnknownError
		}
		list = append(list, d)
	}
	if err := rows.Err(); err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return model.PaginatedResponse{}, model.UnknownError
	}

	return model.PaginatedResponse{Data: list, Count: count, Page: f.Page, Size: f.Size}, nil
}

func GetDonor(id int) (model.Donor, error) {
	query := "SELECT " + donorColumns + " FROM users u WHERE u.id = $1 AND " + donorBaseWhere

	d, err := scanDonor(DB.QueryRow(query, id))
	if errors.Is(err, sql.ErrNoRows) {
		return model.Donor{}, model.NewError(itn.ErrorUserNotFound, 404)
	}
	if err != nil {
		logger.GetLogger().LogErrors(err, map[string]interface{}{"query": query})
		return model.Donor{}, model.UnknownError
	}
	return d, nil
}

func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}
