package dao

import (
	"database/sql"
	"errors"

	itn "github.com/mdhasib01/go-rest-starter/itn"
	model "github.com/mdhasib01/go-rest-starter/model"
	"github.com/mdhasib01/go-rest-starter/pkg/logger"
	"github.com/mdhasib01/go-rest-starter/utils"
)

func UpdateSession(token string) error {
	const query = `UPDATE sessions
						SET
							last_used = now(),
							expire_at = now() + interval '1 hour'
					WHERE
						token = $1
						AND expire_at > now()
					RETURNING token`

	err := DB.QueryRow(query, token).Scan(&token)
	if errors.Is(err, sql.ErrNoRows) {
		logger.GetLogger().LogErrors(err, nil)
		return model.NewError(itn.ErrorTokenExpired, 403)
	}
	if err != nil {
		logger.GetLogger().LogErrors(err, map[string]interface{}{
			"query": query,
		})
		return model.NewError(itn.ErrorUnknown, 400)
	}
	return nil
}

func GetRole(id int) (model.Role, error) {
	var role model.Role
	query := `SELECT id, name FROM role WHERE id=$1`
	err := DB.QueryRow(query, id).Scan(&role.Id, &role.Name)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Role{}, model.NewError(itn.ErrorRoleNotFound, 404)
	}
	if err != nil {
		logger.GetLogger().LogErrors(err, map[string]interface{}{
			"query": query,
		})
		return model.Role{}, model.UnknownError
	}
	return role, nil
}

func GetRolePermissions(idRole int) ([]model.Permission, error) {
	var permissions []model.Permission
	query := `SELECT p.id, p.name, p.description 
			  FROM permission p 
			  JOIN role_permission rp ON p.id = rp.permission_id 
			  WHERE rp.role_id = $1`

	rows, err := DB.Query(query, idRole)
	if err != nil {
		logger.GetLogger().LogErrors(err, map[string]interface{}{
			"query": query,
		})
		return nil, model.UnknownError
	}
	defer rows.Close()

	for rows.Next() {
		var p model.Permission
		if err := rows.Scan(&p.Id, &p.Name, &p.Description); err != nil {
			logger.GetLogger().LogErrors(err, nil)
			return nil, model.UnknownError
		}
		permissions = append(permissions, p)
	}
	return permissions, nil
}

func IsTrustedLocation(cityGeoID, countryGeoID, userID int) (bool, error) {
	var id int
	query := `SELECT id FROM user_locations 
			  WHERE city_geo_id = $1 AND country_geo_id = $2 AND user_id = $3 AND is_trusted = true`

	err := DB.QueryRow(query, cityGeoID, countryGeoID, userID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		logger.GetLogger().LogErrors(err, map[string]interface{}{
			"query": query,
		})
		return false, model.UnknownError
	}
	return true, nil
}

func InsertLocation(location model.Location) error {
	query := `INSERT INTO user_locations (country, country_geo_id, city, city_geo_id, is_trusted, user_id, trust_code)
			  VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := DB.Exec(query, location.Country, location.CountryGeoID, location.City, location.CityGeoID, location.IsTrusted, location.UserID, location.TrustCode)
	if err != nil {
		logger.GetLogger().LogErrors(err, map[string]interface{}{
			"query": query,
		})
		return model.UnknownError
	}
	return nil
}

func Login(email string) (model.User, error) {
	var user model.User
	var lastlogin sql.NullTime
	const query = `SELECT
					 	id, first_name, last_name, gender, password, phone, city, address,
					 	email, country, role_id, is_available, COALESCE(created_by, 0), created_at,
						last_login, donated_at, is_verified, verification_code, is_password_temporary, hide_personal_info
					FROM
					  	users 
					WHERE
						email = $1`

	var donatedAt sql.NullTime

	err := DB.QueryRow(query, email).Scan(&user.Id, &user.Firstname, &user.Lastname, &user.Gender,
		&user.Password, &user.Phone, &user.City, &user.Address, &user.Email,
		&user.Country, &user.IdRole, &user.IsAvailable, &user.CreatedBy, &user.CreatedAt,
		&lastlogin, &donatedAt, &user.IsVerified, &user.VerificationCode, &user.IsPasswordTemporary, &user.HidePersonalInfo,
	)

	if errors.Is(err, sql.ErrNoRows) {
		logger.GetLogger().LogErrors(err, nil)
		return model.User{}, model.NewError(itn.ErrorLoginFailed, 403)
	}
	if err != nil {
		logger.GetLogger().LogErrors(err, map[string]interface{}{
			"query": query,
		})

		return model.User{}, model.NewError(itn.ErrorLoginFailed, 500)
	}

	user.LastLogin = utils.ConvertNullTime(lastlogin)
	if donatedAt.Valid {
		user.DonatedAt = donatedAt.Time.Format("2006-01-02")
	}

	return user, nil
}

func UpdateLastLogin(id int) error {
	const query = `UPDATE users
						SET
							last_login = now()
					WHERE
						id=$1`
	_, err := DB.Exec(query, id)
	if err != nil {
		logger.GetLogger().LogErrors(err, map[string]interface{}{
			"query": query,
		})
		return model.UnknownError
	}

	return nil
}

func CreateSession(session model.Session) error {
	const query = `INSERT INTO sessions
						(role_id, user_id, token)
					VALUES(
						$1, $2, $3
					)`

	_, err := DB.Exec(query, session.RoleId, session.UserId, session.Token)
	if err != nil {
		logger.GetLogger().LogErrors(err, map[string]interface{}{
			"query": query,
		})
		return model.UnknownError
	}

	return nil
}

func DisableSession(token string) (int, error) {
	var idreturn int
	errorStr := ""

	query := "DELETE FROM sessions WHERE token=$1  RETURNING id;"

	err := DB.QueryRow(query, token).Scan(&idreturn)

	if errors.Is(err, sql.ErrNoRows) {
		errorStr = itn.ErrorTokenInvalid
		return 0, model.NewError(errorStr, 403)
	}
	if err != nil {
		logger.GetLogger().LogErrors(err, map[string]interface{}{
			"query": query,
		})
		return 0, model.NewError(itn.ErrorUnknown, 500)
	}

	return idreturn, nil

}

func GetUserPlan(userId int) (string, int, error) {
	return "free", 0, nil
}

func IncrementEvaluationUsed(userId int) error {
	return nil
}
