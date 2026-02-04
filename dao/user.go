package dao

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

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
	query := `INSERT INTO representative (user_id, createdby) VALUES ($1, $2);`
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
	query := `INSERT INTO professional (user_id, profession, createdby) VALUES ($1, $2, $3);`

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
		firstname, lastname, username, password, phone, address,
		email, city, id_role, country, created_by, verification_code, is_password_temporary, isverified)
		Values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14) RETURNING id`

	err := DB.QueryRow(query, user.Firstname, user.Lastname, user.Username, user.Password, user.Phone, user.Address,
		user.Email, user.City, user.IdRole, user.Country, user.CreatedBy,
		user.VerificationCode, user.IsPasswordTemporary, user.IsVerified,
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

	query := `UPDATE  users SET photo = $1, updated_at=CURRENT_TIMESTAMP where id = $2 AND isactive=TRUE`

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

	query := fmt.Sprintf(`UPDATE  users SET %s updated_at=CURRENT_TIMESTAMP, updated_by=$1
					WHERE 
						id=$2
						AND isactive=TRUE
						`, prepareUserFieldsToUpdate(user),
	)

	_, err := DB.Exec(query, s.UserId, id)

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

		err = rows.Scan(&user.Id, &user.Firstname, &user.Lastname, &user.Username, &user.Phone, &user.City, &user.Address,
			&user.Email, &user.Country, &user.RoleName, &user.IsActive, &user.CreatedBy, &user.CreatedAt, &lastlogin, &user.IdRole)
		if err != nil {
			logger.GetLogger().LogErrors(err, nil)
			return model.PaginatedResponse{}, model.UnknownError
		}
		user.LastLogin = utils.ConvertNullTime(lastlogin)

		list = append(list, user)

	}

	count, err := GetTableCount(query, "id", fr, args, "u")
	if err != nil {
		return model.PaginatedResponse{}, err
	}

	return model.PaginatedResponse{Data: list, Count: count}, nil
}

func searchUserQuery() string {
	var query = `SELECT
						u.id, u.firstname, u.lastname, u.username, u.phone, u.city, u.address, u.email,
						u.country, r.name, u.isactive, u.created_by, u.created_at, u.lastlogin,
						u.id_role 
						FROM users u
						JOIN role r ON r.id=u.id_role 
						WHERE
							u.isactive=TRUE
							`

	return query
}

func GetRepresentativeProfileID(userId int) (int, error) {
	var id int
	query := `SELECT id FROM representative WHERE user_id=$1 AND isdeleted=false`
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
	query := `SELECT id FROM professional WHERE user_id=$1 AND isdeleted=false`
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

func GetAllUsersWithPlan(session model.Session, fr model.FilteredRequest) (model.PaginatedResponse, error) {
	var list []model.UserWithPlan
	query := `SELECT
  u.id                            AS user_id,
  u.firstname,
  u.lastname,
  u.email                         AS email,
  COALESCE(p.name, 'free')        AS plan
FROM users u
LEFT JOIN LATERAL (
  SELECT *
  FROM subscriptions s
  WHERE s.user_id = u.id
  ORDER BY (s.is_active::int) DESC, s.start_date DESC, s.created_at DESC
  LIMIT 1
) AS sub ON true
LEFT JOIN plans p ON p.id = sub.plan_id WHERE 1=1`

	args := []interface{}{}

	if fr.Search != "" {
		args = append(args, "%"+fr.Search+"%")
		query += fmt.Sprintf(" AND (u.firstname ILIKE $%d OR u.lastname ILIKE $%d OR u.email ILIKE $%d)", len(args), len(args), len(args))
	}

	if fr.Plan != "" {
		if strings.ToLower(fr.Plan) == "free" {
			query += " AND p.name IS NULL"
		} else {
			args = append(args, fr.Plan)
			query += fmt.Sprintf(" AND p.name = $%d", len(args))
		}
	}

	if fr.OrderParams == "" {
		fr.OrderParams = "id:asc"
		fr.AllowedOrderParams = append(fr.AllowedOrderParams, "id")
	}

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
		var row model.UserWithPlan
		err = rows.Scan(&row.UserId, &row.Firstname, &row.Lastname, &row.Email, &row.Plan)
		if err != nil {
			logger.GetLogger().LogErrors(err, nil)
			return model.PaginatedResponse{}, model.UnknownError
		}
		list = append(list, row)
	}

	var count int
	countQuery := query
	upper := strings.ToUpper(countQuery)
	idx := strings.LastIndex(upper, " LIMIT ")
	if idx != -1 {
		countQuery = countQuery[:idx]
		upper = strings.ToUpper(countQuery)
	}
	idx = strings.LastIndex(upper, " OFFSET ")
	if idx != -1 {
		countQuery = countQuery[:idx]
	}

	countQuery = "SELECT COUNT(*) FROM (" + countQuery + ") AS t"
	argsCnt := strings.Count(countQuery, "$")
	err = DB.QueryRow(countQuery, args[:argsCnt]...).Scan(&count)
	if err != nil {
		logger.GetLogger().LogErrors(err, map[string]interface{}{
			"query": countQuery,
		})
		return model.PaginatedResponse{}, err
	}

	return model.PaginatedResponse{Data: list, Count: count}, nil
}

func GetUser(id int, s model.Session) (model.User, error) {

	query := `SELECT
	u.id, u.firstname, u.lastname, u.username, u.phone, u.city, u.address, u.email,
	u.country, r.name, u.isactive, u.created_by, u.created_at, u.lastlogin, u.id_role, u.isverified
	FROM users u
	JOIN role r ON r.id=u.id_role 
	WHERE
		u.id=$1
		AND u.isactive=TRUE`

	var user model.User
	var lastlogin sql.NullTime

	err := DB.QueryRow(query, id).Scan(
		&user.Id, &user.Firstname, &user.Lastname, &user.Username, &user.Phone, &user.City, &user.Address,
		&user.Email, &user.Country, &user.RoleName, &user.IsActive, &user.CreatedBy, &user.CreatedAt, &lastlogin, &user.IdRole, &user.IsVerified,
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

	return user, nil
}

func GetCredendials(id int) (model.LoginRequest, string) {

	var user model.LoginRequest

	var errString string
	query := "Select username,password from users where id=$1 AND isactive=TRUE ;"
	err := DB.QueryRow(query, id).Scan(&user.Username, &user.Password)
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

	query := "UPDATE users set updated_at=CURRENT_TIMESTAMP, isactive=FALSE WHERE (id=$1 AND isactive=TRUE)  RETURNING id;"

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
	query := `SELECT id, email, lastlogin FROM users WHERE 
	lastlogin <= NOW() - INTERVAL '%d month' AND isactive = true AND id_role = 3 `
	query = fmt.Sprintf(query, monthsInterval)
	if includeOneDay {
		query += fmt.Sprintf(` AND lastlogin >= NOW() - INTERVAL '%d month' - INTERVAL '1 day'`, monthsInterval)
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

	query := "UPDATE users set updated_at=CURRENT_TIMESTAMP, isactive=NOT isactive WHERE id=$1  RETURNING id;"

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
	query := `UPDATE users SET isverified = false, isactive = true WHERE email=$1 AND isactive=FALSE RETURNING id`
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

	requestStatement := `SELECT id FROM ` + table + ` WHERE ` + field + `= $1 AND isactive=TRUE`

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
	query := `UPDATE users SET password=$1, ispasswordtemp = false WHERE id=$2 RETURNING id`
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
	query := `UPDATE users SET password=$1, ispasswordtemp = true  WHERE id=$2 RETURNING id`
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
	WHERE email=$1 AND isactive=TRUE`
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
	query := `UPDATE users SET isverified=true WHERE verificationcode=$1 RETURNING firstname`
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

	query := `SELECT firstname, email FROM users WHERE id=$1`
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
		query += fmt.Sprintf("firstname = '%s',", user.Firstname)
	}
	if !utils.IsEmpty(user.Lastname) {
		query += fmt.Sprintf("lastname = '%s',", user.Lastname)
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
	if !utils.IsEmpty(user.Username) {
		query += fmt.Sprintf("username = '%s',", user.Username)
	}
	if user.IdRole > 0 {
		query += fmt.Sprintf("id_role = '%d',", user.IdRole)
	}

	return query
}
