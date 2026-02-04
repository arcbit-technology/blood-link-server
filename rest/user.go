package rest

import (
	"net/http"
	"strconv"
	"strings"

	controller "github.com/mdhasib01/go-rest-starter/controller"
	itn "github.com/mdhasib01/go-rest-starter/itn"
	model "github.com/mdhasib01/go-rest-starter/model"
	"github.com/mdhasib01/go-rest-starter/pkg/logger"

	"github.com/gorilla/mux"
)

// CreateUsers godoc
// @Summary create user
// @Description this route is used to create a new user
// @Tags users
// @Accept  json
// @Produce  json
// @Security Bearer
// @Param Authorization header string true "Bearer Token" default("Bearer YOUR_ACCESS_TOKEN")
// @Param user body model.User true "user"
// @Success 200 {object}   model.User
// @Failure 400 {string} string
// @Failure 500 {string} string
// @Router /user [post]
func CreateUser(w http.ResponseWriter, r *http.Request) {
	session, err := isAuthorized(r, CreateUserPermission)
	if err != nil {
		JSON(w, nil, err)
		return
	}

	var user model.User
	err = readRequestBody(r, &user)
	if err != nil {
		JSON(w, nil, err)
		return
	}

	user.CreatedBy = session.UserId

	set, err := controller.CreateUser(user, session)
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		JSON(w, nil, err)
		return
	}

	JSON(w, set, nil)
}

// UpdateUsers godoc
// @Summary update user
// @Description this route is used to update an existing user
// @Tags users
// @Accept  json
// @Produce  json
// @Security Bearer
// @Param Authorization header string true "Bearer Token" default("Bearer YOUR_ACCESS_TOKEN")
// @Param id path string true "id"
// @Param user body model.User true "user"
// @Success 200 {boolean}   boolean
// @Failure 400 {string} string
// @Failure 500 {string} string
// @Router /user/{id} [put]
func UpdateUser(w http.ResponseWriter, r *http.Request) {
	session, err := isAuthorized(r, UpdateUserPermission)
	if err != nil {
		JSON(w, nil, err)
		return
	}

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		JSON(w, nil, invalidDataError)
		return
	}

	var user model.User

	err = readRequestBody(r, &user)
	if err != nil {
		ERROR(w, http.StatusOK, err)
		return
	}

	// not allowed for pormotions from same level of authorization or lower
	if session.RoleId > user.IdRole && user.IdRole != 0 {
		JSON(w, nil, model.NewError(itn.ErrorUnauthorizedOperation, 403))
		return
	}

	err = controller.UpdateUser(id, user, session)
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		JSON(w, nil, err)
		return
	}

	JSON(w, map[string]string{"message": "User updated successfully"}, nil)
}

// DeleteUsers godoc
// @Summary delete user
// @Description this route is used to delete an existing user
// @Tags users
// @Accept  json
// @Produce  json
// @Security Bearer
// @Param Authorization header string true "Bearer Token" default("Bearer YOUR_ACCESS_TOKEN")
// @Param id path string true "id"
// @Success 200 {boolean}   boolean
// @Failure 400 {string} string
// @Failure 500 {string} string
// @Router /user/{id} [delete]
func DeleteUser(w http.ResponseWriter, r *http.Request) {
	session, err := isAuthorized(r, DeleteUserPermission)
	if err != nil {
		JSON(w, nil, err)
		return
	}

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		JSON(w, nil, invalidDataError)
		return
	}

	user, err := controller.GetUser(id, session)
	if err != nil {
		JSON(w, nil, err)
		return
	}

	// not allowed for user in lower level of authorization to delete user in higher or same level
	if user.IdRole <= session.RoleId {
		// if user is deleting himself
		if session.UserId != id {
			JSON(w, nil, model.NewError(itn.ErrorUnauthorizedOperation, 403))
			return
		}
	}

	err = controller.DeleteUser(id)
	JSON(w, nil, err)
}

// ToggleUserStatus godoc
// @Summary delete user
// @Description this route is used to delete an existing user
// @Tags users
// @Accept  json
// @Produce  json
// @Security Bearer
// @Param Authorization header string true "Bearer Token" default("Bearer YOUR_ACCESS_TOKEN")
// @Param id path string true "id"
// @Success 200 {boolean}   boolean
// @Failure 400 {string} string
// @Failure 500 {string} string
// @Router /user/{id}/disable [put]
// @Router /user/{id}/enable [put]
func ToggleUserStatus(w http.ResponseWriter, r *http.Request) {
	session, err := isAuthorized(r, ToggleUserStatusPermission)
	if err != nil {
		JSON(w, nil, err)
		return
	}

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		JSON(w, nil, invalidDataError)
		return
	}

	user, err := controller.GetUser(id, session)
	if err != nil {
		JSON(w, nil, err)
		return
	}

	// not allowed for user in lower level of authorization to disable user in higher or same level
	if user.IdRole <= session.RoleId && session.UserId != id {
		JSON(w, nil, model.NewError(itn.ErrorUnauthorizedOperation, 403))
		return
	}

	path := r.URL.Path
	if strings.Contains(path, "disable") {
		if !user.IsActive {
			JSON(w, nil, model.NewError(itn.ErrorAlreadyDisabled, 400))
			return
		}
	}

	if strings.Contains(path, "enable") {
		if user.IsActive {
			JSON(w, nil, model.NewError(itn.ErrorAlreadyEnabled, 400))
			return
		}
	}

	err = controller.ToggleUserStatus(id)
	JSON(w, nil, err)
}

// GetUser godoc
// @Summary get user
// @Description this route is used to get a user by id
// @Tags users
// @Accept  json
// @Produce  json
// @Security Bearer
// @Param Authorization header string true "Bearer Token" default("Bearer YOUR_ACCESS_TOKEN")
// @Param id path string true "id"
// @Success 200 {object}   model.User
// @Failure 400 {string} string
// @Failure 500 {string} string
// @Router /user/{id} [get]
func GetUser(w http.ResponseWriter, r *http.Request) {
	session, err := isAuthorized(r, ViewUserPermission)
	if err != nil {
		JSON(w, nil, err)
		return
	}
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		JSON(w, nil, invalidDataError)
		return
	}
	set, err := controller.GetUser(id, session)
	JSON(w, set, err)
}

// GetAllUsers godoc
// @Summary get all users
// @Description this route is used to get all the users
// @Tags users
// @Accept  json
// @Produce  json
// @Security Bearer
// @Param Authorization header string true "Bearer Token" default("Bearer YOUR_ACCESS_TOKEN")
// @Param page query int false "Page number" default(1)
// @Param size query int false "Page size" default(10)
// @Success 200 {object} model.PaginatedResponse
// @Failure 400 {string} string
// @Failure 500 {string} string
// @Router /users [get]
func GetAllUsers(w http.ResponseWriter, r *http.Request) {
	session, err := isAuthorized(r, ListUserPermission)
	if err != nil {
		JSON(w, nil, err)
		return
	}

	var fr model.FilteredRequest
	fr.Page, _ = strconv.Atoi(r.URL.Query().Get("page"))
	fr.Size, _ = strconv.Atoi(r.URL.Query().Get("size"))

	fr.Filters = r.URL.Query().Get("filters")

	fr.AllowedFilters = []string{
		"role",
	}

	set, err := controller.GetAllUsers(session, fr)
	JSON(w, set, err)
}

// GetAllUsersPlans godoc
// @Summary get all users with plans
// @Description this route is used to get all the users with their plans
// @Tags users
// @Accept  json
// @Produce  json
// @Security Bearer
// @Param Authorization header string true "Bearer Token" default("Bearer YOUR_ACCESS_TOKEN")
// @Param page query int false "Page number" default(1)
// @Param size query int false "Page size" default(10)
// @Param filters query string false "Filters"
// @Param search query string false "Search"
// @Param plan query string false "Plan"
// @Success 200 {object} model.PaginatedResponse
// @Failure 400 {string} string
// @Failure 500 {string} string
// @Router /users/plans [get]
func GetAllUsersPlans(w http.ResponseWriter, r *http.Request) {
	session, err := isAuthorized(r, 0)
	if err != nil {
		JSON(w, nil, err)
		return
	}

	var fr model.FilteredRequest
	fr.Page, _ = strconv.Atoi(r.URL.Query().Get("page"))
	fr.Size, _ = strconv.Atoi(r.URL.Query().Get("size"))
	fr.Filters = r.URL.Query().Get("filters")
	fr.Search = r.URL.Query().Get("search")
	fr.Plan = r.URL.Query().Get("plan")

	fr.AllowedFilters = []string{
		"firstname",
		"lastname",
		"email",
	}

	set, err := controller.GetAllUsersWithPlan(session, fr)
	JSON(w, set, err)
}

// ChangePassword godoc
// @Summary update user
// @Description this route is used to change or reset password of the user
// @Tags users
// @Accept  json
// @Produce  json
// @Security Bearer
// @Param Authorization header string true "Bearer Token" default("Bearer YOUR_ACCESS_TOKEN")
// @Param id path string true "id"
// @Param user body model.ChangePassword true "change password request"
// @Success 200 {string}   string
// @Failure 400 {string} string
// @Failure 500 {string} string
// @Router /changepassword/{id} [put]
// @Router /resetpassword/{id} [put]
func ChangeOrResetPassword(w http.ResponseWriter, r *http.Request) {
	sess, err := isAuthorized(r, ChangePasswordPermission)
	if err != nil {
		JSON(w, nil, err)
		return
	}

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		JSON(w, nil, invalidDataError)
		return
	}
	routePath := r.URL.Path

	var set interface{}
	if strings.Contains(routePath, "changepassword") {

		var password model.ChangePassword
		err = readRequestBody(r, &password)
		if err != nil {
			JSON(w, nil, err)
			return
		}

		set, err = controller.ChangePassword(id, password.OldPasword, password.NewPassword, sess)
	} else {
		set, err = controller.ResetPassword(id, sess)
	}

	JSON(w, set, err)
}

// ChangeTempPassword godoc
// @Summary update user password
// @Description this route is used to change the temporary password of the user
// @Tags users
// @Accept  json
// @Produce  json
// @Security Bearer
// @Param Authorization header string true "Bearer Token" default("Bearer YOUR_ACCESS_TOKEN")
// @Param user body model.ChangeTempPassword true "change password request"
// @Success 200 {object}   model.Login
// @Failure 400 {string} string
// @Failure 403 {string} string
// @Failure 500 {string} string
// @Router /changetemppassword [put]
func ChangeTempPassword(w http.ResponseWriter, r *http.Request) {
	session, err := isAuthorized(r, ChangePasswordPermission)
	if err != nil {
		JSON(w, nil, err)
		return
	}
	var pass model.ChangeTempPassword
	err = readRequestBody(r, &pass)
	if err != nil {
		JSON(w, nil, err)
		return
	}

	set, err := controller.ChangeTempPassword(pass, session, r.Header.Get("X-Forwarded-For"))
	JSON(w, set, err)
}

// GetProfileCompletePercentage godoc
// @Summary get profile complete percentage
// @Description this route is used to get the profile complete percentage of the user
// @Tags users
// @Accept  json
// @Produce  json
// @Security Bearer
// @Param Authorization header string true "Bearer Token" default("Bearer YOUR_ACCESS_TOKEN")
// @Success 200 {object}   int
// @Failure 400 {string} string
// @Failure 500 {string} string
// @Router /profilecomplete [get]
func GetProfileCompletePercentage(w http.ResponseWriter, r *http.Request) {
	session, err := isAuthorized(r, 0)
	if err != nil {
		JSON(w, nil, err)
		return
	}

	percentage, err := controller.GetProfileCompletePercentage(session)
	if err != nil {
		JSON(w, nil, err)
		return
	}

	JSON(w, percentage, nil)
}
