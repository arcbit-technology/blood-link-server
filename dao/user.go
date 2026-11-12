package dao

import (
	"database/sql"
	"errors"
	"fmt"

	itn "github.com/mdhasib01/go-rest-starter/itn"
	model "github.com/mdhasib01/go-rest-starter/model"
	"github.com/mdhasib01/go-rest-starter/pkg/logger"
	"github.com/mdhasib01/go-rest-starter/utils"
)

func CreateUser(user *model.User) error {

	tx, err := DB.Begin()
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return model.UnknownError
	}
	defer tx.Rollback()

	err = createUser(user, tx)
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return err
	}

	tx.Commit()

	return nil
}

func createRepresentative(tx *sql.Tx, user model.User) error {
	query := `INSERT INTO representative (user_id, created_by) VALUES ($1, $2);`
	// TODO: check representative extra data
	_, err := tx.Exec(query, user.Id, user.CreatedBy)
	if err != nil {
		logger.GetLogger().LogErrors(err, map[string]interface{}{
			"query": query,
		})
		return err
	}

	return nil

}

func createProfessional(tx *sql.Tx, user model.User) error {
	query := `INSERT INTO professional (user_id, profession, created_by) VALUES ($1, $2, $3);`

	_, err := tx.Exec(query, user.Id, user.Profession, user.CreatedBy)
	if err != nil {
		logger.GetLogger().LogErrors(err, map[string]interface{}{
			"query": query,
		})
		return err
	}

	return nil
}

func createUser(user *model.User, tx *sql.Tx) error {
	query := `Insert into users (
		first_name, last_name, password, phone, address,
		email, city, role_id, country, created_by, verification_code, is_password_temporary, is_verified, hide_personal_info,
		gender, district, division, blood_group, profession, date_of_birth, donated_at, upazila)
		Values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14,
		NULLIF($15, ''), NULLIF($16, ''), NULLIF($17, ''), NULLIF($18, ''), NULLIF($19, ''),
		CASE WHEN $20::int > 0 THEN (current_date - make_interval(years => $20::int))::date END,
		NULLIF($21, '')::date, NULLIF($22, '')) RETURNING id`

	err := tx.QueryRow(query, user.Firstname, user.Lastname, user.Password, user.Phone, user.Address,
		user.Email, user.City, user.IdRole, user.Country, user.CreatedBy,
		user.VerificationCode, user.IsPasswordTemporary, user.IsVerified, user.HidePersonalInfo,
		user.Gender, user.District, user.Division, user.BloodGroup, user.Profession, user.Age, user.DonatedAt,
		user.Upazila,
	).Scan(&user.Id)

	if err != nil {
		logger.GetLogger().LogErrors(err, map[string]interface{}{
			"query": query,
		})
		return model.UnknownError
	}

	return nil
}

func UpdateUserProfilePicture(userProfileKey string, id int) error {

	query := `UPDATE  users SET photo = $1, updated_at=CURRENT_TIMESTAMP where id = $2 AND is_available=TRUE`

	_, err := DB.Exec(query, userProfileKey, id)

	if err != nil {
		logger.GetLogger().LogErrors(err, map[string]interface{}{
			"query": query,
		})
		return model.UnknownError
	}

	return nil
}

func UpdateUser(user model.User, id int, s model.Session) error {
	args := []interface{}{s.UserId, id}
	// location fields are validated by the controller and always sent
	// together; they are bound as parameters
	location := ""
	if user.Division != "" && user.District != "" {
		args = append(args, user.Division, user.District, user.Upazila)
		location = "division = $3, district = $4, upazila = NULLIF($5, ''),"
	}

	query := fmt.Sprintf(`UPDATE  users SET %s %s updated_at=CURRENT_TIMESTAMP, updated_by=$1
					WHERE 
						id=$2
						AND is_available=TRUE
						`, prepareUserFieldsToUpdate(user), location,
	)

	_, err := DB.Exec(query, args...)

	if err != nil {
		logger.GetLogger().LogErrors(err, map[string]interface{}{
			"query": query,
		})
		return model.UnknownError
	}

	return nil
}

func GetAllUsers(fr model.FilteredRequest) (model.PaginatedResponse, error) {
	var list []model.User
	var lastlogin sql.NullTime
	var query = searchUserQuery()

	args := []interface{}{}
	var err error
	query, args, err = buildFilteredQuery(query, fr, args, "u")
	if err != nil {
		return model.PaginatedResponse{}, err
	}

	rows, err := DB.Query(query, args...)
	if err != nil {
		logger.GetLogger().LogErrors(err, map[string]interface{}{
			"query": query,
		})
		return model.PaginatedResponse{}, model.UnknownError
	}
	defer rows.Close()

	for rows.Next() {
		var user model.User

		var donatedAt sql.NullTime

		err = rows.Scan(&user.Id, &user.Firstname, &user.Lastname, &user.Gender, &user.Phone, &user.City, &user.Address,
			&user.Email, &user.Country, &user.RoleName, &user.IsAvailable, &user.CreatedBy, &user.CreatedAt, &lastlogin, &donatedAt, &user.IdRole, &user.HidePersonalInfo)
		if err != nil {
			logger.GetLogger().LogErrors(err, nil)
			return model.PaginatedResponse{}, model.UnknownError
		}
		user.LastLogin = utils.ConvertNullTime(lastlogin)
		if donatedAt.Valid {
			user.DonatedAt = donatedAt.Time.Format("2006-01-02")
		}

		list = append(list, user)

	}

	count, err := GetTableCount(query, "id", fr, args, "u")
	if err != nil {
		return model.PaginatedResponse{}, err
	}

	return model.PaginatedResponse{Data: list, Count: count}, nil
}

func searchUserQuery() string {
	// COALESCE: nullable columns are scanned into plain Go strings / ints
	var query = `SELECT
						u.id, COALESCE(u.first_name, ''), COALESCE(u.last_name, ''), COALESCE(u.gender, ''),
						COALESCE(u.phone, ''), COALESCE(u.city, ''), COALESCE(u.address, ''), u.email,
						COALESCE(u.country, ''), r.name, u.is_available, COALESCE(u.created_by, 0), u.created_at,
						u.last_login, u.donated_at, u.role_id, COALESCE(u.hide_personal_info, false)
						FROM users u
						JOIN role r ON r.id=u.role_id 
						WHERE
							u.is_available=TRUE
							`

	return query
}

func GetRepresentativeProfileID(userId int) (int, error) {
	var id int
	query := `SELECT id FROM representative WHERE user_id=$1 AND is_deleted=false`
	err := DB.QueryRow(query, userId).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, model.NewError(itn.ErrorProfileNotFound, 404)
	}
	if err != nil {
		logger.GetLogger().LogErrors(err, map[string]interface{}{
			"query": query,
		})
		return 0, model.UnknownError
	}
	return id, nil
}

func GetProfessionalProfileID(userId int) (int, error) {
	var id int
	query := `SELECT id FROM professional WHERE user_id=$1 AND is_deleted=false`
	err := DB.QueryRow(query, userId).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, model.NewError(itn.ErrorProfileNotFound, 404)
	}
	if err != nil {
		logger.GetLogger().LogErrors(err, map[string]interface{}{
			"query": query,
		})
		return 0, model.UnknownError
	}
	return id, nil
}

func GetBuyerProfileCompletePercentage(session model.Session) (int, error) {
	return 100, nil
}

func GetSellerProfileCompletePercentage(session model.Session) (int, error) {
	return 100, nil
}

func GetUser(id int, s model.Session) (model.User, error) {

	query := `SELECT
	u.id, COALESCE(u.first_name, ''), COALESCE(u.last_name, ''), COALESCE(u.gender, ''),
	COALESCE(u.phone, ''), COALESCE(u.city, ''), COALESCE(u.address, ''), u.email,
	COALESCE(u.country, ''), r.name, u.is_available, COALESCE(u.created_by, 0), u.created_at,
	u.last_login, u.donated_at, u.role_id, COALESCE(u.is_verified, false), COALESCE(u.hide_personal_info, false),
	COALESCE(u.division, ''), COALESCE(u.district, ''), COALESCE(u.upazila, ''), COALESCE(u.blood_group, '')
	FROM users u
	JOIN role r ON r.id=u.role_id 
	WHERE
		u.id=$1
		AND u.is_available=TRUE`

	var user model.User
	var lastlogin sql.NullTime

	var donatedAt sql.NullTime

	err := DB.QueryRow(query, id).Scan(
		&user.Id, &user.Firstname, &user.Lastname, &user.Gender, &user.Phone, &user.City, &user.Address,
		&user.Email, &user.Country, &user.RoleName, &user.IsAvailable, &user.CreatedBy, &user.CreatedAt, &lastlogin, &donatedAt, &user.IdRole, &user.IsVerified, &user.HidePersonalInfo,
		&user.Division, &user.District, &user.Upazila, &user.BloodGroup,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return model.User{}, model.NewError(itn.ErrorUserNotFound, 404)
	}
	if err != nil {
		logger.GetLogger().LogErrors(err, map[string]interface{}{
			"query": query,
		})
		return model.User{}, model.UnknownError
	}

	// lower level users cannot get higher or same level users
	if s.RoleId != model.SUPERADMIN && s.RoleId != model.ADMIN && user.Id != s.UserId {
		if user.IdRole <= s.RoleId {
			return model.User{}, model.NewError(itn.ErrorUnauthorizedOperation, 403)
		}
	}

	user.LastLogin = utils.ConvertNullTime(lastlogin)
	if donatedAt.Valid {
		user.DonatedAt = donatedAt.Time.Format("2006-01-02")
	}

	return user, nil
}

func GetCredendials(id int) (model.LoginRequest, string) {

	var user model.LoginRequest

	var errString string
	query := "Select password from users where id=$1 AND is_available=TRUE ;"
	err := DB.QueryRow(query, id).Scan(&user.Password)
	if err != nil {
		logger.GetLogger().LogErrors(err, map[string]interface{}{
			"query": query,
		})
		errString = itn.ErrorUnknown
	}

	return user, errString
}

func DeleteUser(idUser int) error {
	var idreturn int

	query := "UPDATE users set updated_at=CURRENT_TIMESTAMP, is_available=FALSE WHERE (id=$1 AND is_available=TRUE)  RETURNING id;"

	err := DB.QueryRow(query, idUser).Scan(&idreturn)
	if errors.Is(err, sql.ErrTxDone) {
		return model.NewError(itn.ErrorUserNotFound, 404)
	}
	if err != nil {
		logger.GetLogger().LogErrors(err, map[string]interface{}{
			"query": query,
		})
		return model.UnknownError
	}
	return nil
}

func GetUsersByLastLogin(monthsInterval int, includeOneDay bool) ([]model.User, error) {
	query := `SELECT id, email, last_login FROM users WHERE 
	last_login <= NOW() - INTERVAL '%d month' AND is_available = true AND role_id = 3 `
	query = fmt.Sprintf(query, monthsInterval)
	if includeOneDay {
		query += fmt.Sprintf(` AND last_login >= NOW() - INTERVAL '%d month' - INTERVAL '1 day'`, monthsInterval)
	}

	var list []model.User

	rows, err := DB.Query(query)
	if err != nil {
		logger.GetLogger().LogErrors(err, map[string]interface{}{
			"query": query,
		})
		return nil, model.UnknownError
	}

	defer rows.Close()
	for rows.Next() {
		var lastlogin sql.NullTime
		var user model.User
		err = rows.Scan(&user.Id, &user.Email, &lastlogin)
		if err != nil {
			logger.GetLogger().LogErrors(err, nil)
			return nil, model.UnknownError
		}
		user.LastLogin = utils.ConvertNullTime(lastlogin)
		list = append(list, user)
	}

	return list, nil
}

func ToggleUserStatus(idUser int) error {
	var idreturn int

	query := "UPDATE users set updated_at=CURRENT_TIMESTAMP, is_available=NOT is_available WHERE id=$1  RETURNING id;"

	err := DB.QueryRow(query, idUser).Scan(&idreturn)
	if errors.Is(err, sql.ErrNoRows) {
		return model.NewError(itn.ErrorUserNotFound, 404)
	}
	if err != nil {
		logger.GetLogger().LogErrors(err, map[string]interface{}{
			"query": query,
		})
		return model.UnknownError
	}

	return nil
}

func IsNewUser(user model.User, session model.Session) (int, error) {
	query := `UPDATE users SET is_verified = false, is_available = true WHERE email=$1 AND is_available=FALSE RETURNING id`
	var id int
	err := DB.QueryRow(query, user.Email).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		logger.GetLogger().LogErrors(err, map[string]interface{}{
			"query": query,
		})
		return 0, model.UnknownError
	}
	return id, nil
}

func IsExistInDb(value string, table string, field string) (int, bool) {
	var id int

	requestStatement := `SELECT id FROM ` + table + ` WHERE ` + field + `= $1 AND is_available=TRUE`

	err := DB.QueryRow(requestStatement, value).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return id, false
	}
	if err != nil {
		logger.GetLogger().LogErrors(err, map[string]interface{}{
			"query": requestStatement,
		})
	}

	if id > 0 {
		return id, true
	}
	return id, false
}

func ChangePassword(userid int, newpassword string) (int, error) {
	var id int
	query := `UPDATE users SET password=$1, is_password_temporary = false WHERE id=$2 RETURNING id`
	err := DB.QueryRow(query, newpassword, userid).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, model.NewError(itn.ErrorUserNotFound, 404)
	}
	if err != nil {
		logger.GetLogger().LogErrors(err, map[string]interface{}{
			"query": query,
		})
		return 0, model.NewError(itn.ErrorUnknown, 500)
	}
	return id, nil
}

func ResetPassword(userid int, newpassword string) (int, error) {
	var id int
	query := `UPDATE users SET password=$1, is_password_temporary = true  WHERE id=$2 RETURNING id`
	err := DB.QueryRow(query, newpassword, userid).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, model.NewError(itn.ErrorUserNotFound, 404)
	}
	if err != nil {
		logger.GetLogger().LogErrors(err, map[string]interface{}{
			"query": query,
		})
		return 0, model.NewError(itn.ErrorUnknown, 500)
	}
	return id, nil
}

func GetUserByEmail(email string) (model.User, error) {
	query := `SELECT
	id FROM users 
	WHERE email=$1 AND is_available=TRUE`
	var user model.User
	err := DB.QueryRow(query, email).Scan(&user.Id)
	if errors.Is(err, sql.ErrNoRows) {
		return model.User{}, model.NewError(itn.ErrorUserNotFound, 404)
	}
	if err != nil {
		logger.GetLogger().LogErrors(err, map[string]interface{}{
			"query": query,
		})
		return model.User{}, model.NewError(itn.ErrorUnknown, 500)
	}

	return user, nil
}

func VerifyUser(code string) (string, error) {
	var firstname string
	query := `UPDATE users SET is_verified=true WHERE verification_code=$1 RETURNING first_name`
	err := DB.QueryRow(query, code).Scan(&firstname)
	if errors.Is(err, sql.ErrNoRows) {
		return "", model.NewError("User Not Found", 404)
	}
	if err != nil {
		logger.GetLogger().LogErrors(err, map[string]interface{}{
			"query": query,
		})
		return "", model.NewError("Internal Server Error", 500)
	}

	return firstname, nil
}

func GetUserBasicData(id int) (name, email string, err error) {

	query := `SELECT first_name, email FROM users WHERE id=$1`
	err = DB.QueryRow(query, id).Scan(&name, &email)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", model.NewError(itn.ErrorUserNotFound, 404)
	}

	if err != nil {
		logger.GetLogger().LogErrors(err, map[string]interface{}{
			"query": query,
		})
		return "", "", model.UnknownError
	}

	return name, email, nil
}

func prepareUserFieldsToUpdate(user model.User) string {
	query := " "
	if !utils.IsEmpty(user.Firstname) {
		query += fmt.Sprintf("first_name = '%s',", user.Firstname)
	}
	if !utils.IsEmpty(user.Lastname) {
		query += fmt.Sprintf("last_name = '%s',", user.Lastname)
	}
	if !utils.IsEmpty(user.Phone) {
		query += fmt.Sprintf("phone = '%s',", user.Phone)
	}
	if !utils.IsEmpty(user.Address) {
		query += fmt.Sprintf("address = '%s',", user.Address)
	}
	if !utils.IsEmpty(user.Email) {
		query += fmt.Sprintf("email = '%s',", user.Email)
	}
	if !utils.IsEmpty(user.City) {
		query += fmt.Sprintf("city = '%s',", user.City)
	}
	if !utils.IsEmpty(user.Country) {
		query += fmt.Sprintf("country = '%s',", user.Country)
	}
	if user.IdRole > 0 {
		query += fmt.Sprintf("role_id = '%d',", user.IdRole)
	}

	return query
}
