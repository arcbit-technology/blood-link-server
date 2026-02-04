package controller

import (
	"fmt"
	"strings"

	"github.com/mdhasib01/go-rest-starter/pkg/data"

	config "github.com/mdhasib01/go-rest-starter/config"
	dao "github.com/mdhasib01/go-rest-starter/dao"
	itn "github.com/mdhasib01/go-rest-starter/itn"
	model "github.com/mdhasib01/go-rest-starter/model"
	"github.com/mdhasib01/go-rest-starter/pkg/logger"
	notifier "github.com/mdhasib01/go-rest-starter/pkg/notifications"
	security "github.com/mdhasib01/go-rest-starter/security"
	utils "github.com/mdhasib01/go-rest-starter/utils"
)

func GetUser(id int, session model.Session) (model.User, error) {
	if session.RoleId == model.USERROLE && session.UserId != id {
		return model.User{}, model.NewError(itn.ErrorUnauthorizedOperation, 403)
	}
	return dao.GetUser(id, session)
}

func UpdateUser(id int, user model.User, session model.Session) error {
	user.UpdateBy = session.UserId

	u, err := dao.GetUser(id, session)
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return err
	}

	if u.IdRole <= session.RoleId && u.Id != session.UserId {
		return model.NewError(itn.ErrorUnauthorizedOperation, 403)
	}

	err = dao.UpdateUser(user, id, session)
	if err != nil {
		return err
	}

	return nil
}

func CreateUser(user model.User, session model.Session) (model.User, error) {
	// id, err := dao.IsNewUser(user, session)
	// if err != nil {
	// 	return model.User{}, err
	// }
	// if id != 0 {
	// 	return dao.GetUser(id, session)
	// }

	err := prepareUserInsertion(&user, session)
	if err != nil {
		return model.User{}, err
	}

	user.CreatedBy = session.UserId
	user.IsVerified = true

	err = dao.CreateUser(&user)
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return model.User{}, err
	}
	return user, nil
}

func prepareUserInsertion(user *model.User, session model.Session) error {
	if err := validateUser(user); err != nil {
		return err
	}

	if err := validateUserPassword(user); err != nil {
		return err
	}

	if err := canGiveUserRole(user.IdRole, session.RoleId); err != nil {
		return err
	}

	if _, err := dao.GetRole(user.IdRole); err != nil {
		return model.NewError(itn.ErrorRoleNotFound, 400)
	}

	if _, isExist := dao.IsExistInDb(user.Username, "users", "username"); isExist {
		return model.NewError(itn.ErrorUsernameTaken, 409)
	}

	if _, isExist := dao.IsExistInDb(user.Email, "users", "email"); isExist {
		return model.NewError(itn.ErrorEmailExists, 409)
	}

	userPassword, _ := utils.BeforeSave(string(user.Password))
	user.Password = data.HiddenJsonString(userPassword)

	return nil
}

func canGiveUserRole(targetRole, userRole int) error {
	// lower level users cannot give higher level roles
	if targetRole < userRole {
		return model.NewError(itn.ErrorUnauthorizedOperation, 403)
	}

	return nil
}

func validateUser(user *model.User) error {
	if user.Firstname == "" {
		return model.NewError(itn.ErrorFirstnameRequired, 400)
	}
	if user.Lastname == "" {
		return model.NewError(itn.ErrorLastnameRequired, 400)
	}
	if user.Username == "" {
		return model.NewError(itn.ErrorUsernameRequired, 400)
	}
	if len(user.Username) < 3 {
		return model.NewError(itn.ErrorTooShortUsername, 400)
	}
	if user.Email == "" {
		return model.NewError(itn.ErrorEmailRequired, 400)
	}
	if user.IdRole < 1 {
		return model.NewError(itn.ErrorRoleRequired, 400)
	}

	return nil
}

func validateUserPassword(user *model.User) error {
	pass := strings.TrimSpace(string(user.Password))
	confirmPass := strings.TrimSpace(string(user.ConfirmPassword))
	if utils.IsEmpty(pass) {
		return model.NewError(itn.ErrorPasswordRequired, 400)
	}

	if utils.IsEmpty(confirmPass) {
		return model.NewError(itn.ErrorPasswordRequired, 400)
	}

	if pass != confirmPass {
		return model.NewError(itn.ErrorPasswordNotMatch, 400)
	}

	if !utils.IsPasswordStrong(pass) {
		return model.NewError(itn.ErrorPasswordNotStrong, 400)
	}
	return nil
}

func DeleteUser(idUser int) error {
	return dao.DeleteUser(idUser)
}

func ToggleUserStatus(idUser int) error {
	return dao.ToggleUserStatus(idUser)
}

func GetAllUsers(session model.Session, fr model.FilteredRequest) (model.PaginatedResponse, error) {
	return dao.GetAllUsers(fr)
}

func GetAllUsersWithPlan(session model.Session, fr model.FilteredRequest) (model.PaginatedResponse, error) {
	return dao.GetAllUsersWithPlan(session, fr)
}

func ChangePassword(userid int, oldpassword, newpassword string, sess model.Session) (bool, error) {
	if userid != sess.UserId {
		return false, model.NewError(itn.ErrorUnauthorizedOperation, 403)
	}
	user, _ := dao.GetCredendials(userid)
	fmt.Println(oldpassword)
	err := security.VerifyPassword(user.Password, oldpassword)

	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return false, model.NewError(itn.ErrorOldPassIncorrect, 400)
	}

	if !utils.IsPasswordStrong(newpassword) {
		return false, model.NewError(itn.ErrorPasswordNotStrong, 400)
	} else if newpassword == "" {
		return false, model.NewError(itn.ErrorInvalidPassword, 400)
	}

	newpassword, _ = utils.BeforeSave(newpassword)

	_, err = dao.ChangePassword(userid, newpassword)
	if err != nil {
		return false, err
	}

	return true, nil
}

func ResetPassword(userid int, sess model.Session) (interface{}, error) {

	user, err := dao.GetUser(userid, sess)
	if err != nil {
		return nil, err
	}
	if user.IdRole <= sess.RoleId {
		return nil, model.NewError(itn.ErrorUnauthorizedOperation, 403)
	}
	randpass := utils.GenerateRandomString(15)
	newpassword, _ := utils.BeforeSave(randpass)

	_, err = dao.ResetPassword(userid, newpassword)
	if err != nil {
		return false, err
	}
	// send email with the password
	body := fmt.Sprintf(`Hello %s, <br> <br>
			We've reset your password. <br>
			Your new password is: <strong>%s</strong> <br><br>
			
			Thank you, <br>
			GoPgDB`, user.Firstname, randpass)
	mail := model.Mail{
		ReceiverEmail: user.Email,
		ReceiverName:  user.Firstname,
		SUBJECT:       "GoPgDB Password Reset",
		BODY:          body,
		SENDER:        config.Param.Account.MAIL_SENDER,
		SENDER_NAME:   config.Param.Account.MAIL_SENDER_NAME,
	}

	err = notifier.ES.SendEmail(mail)
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return false, err
	}

	return randpass, nil
}

func ChangeTempPassword(pass model.ChangeTempPassword, session model.Session, IP string) (model.ProfileLogin, error) {
	// validate password
	if utils.IsEmpty(string(pass.Password)) {
		return model.ProfileLogin{}, model.NewError(itn.ErrorPasswordRequired, 400)
	}

	if string(pass.Password) != pass.ConfirmPassword {
		return model.ProfileLogin{}, model.NewError(itn.ErrorPasswordNotMatch, 400)
	}

	if !utils.IsPasswordStrong(string(pass.Password)) {
		return model.ProfileLogin{}, model.NewError(itn.ErrorPasswordNotStrong, 400)
	}
	newpassword, _ := utils.BeforeSave(string(pass.Password))

	_, err := dao.ChangePassword(session.UserId, newpassword)
	if err != nil {
		return model.ProfileLogin{}, err
	}
	user, err := GetUser(session.UserId, session)
	if err != nil {
		return model.ProfileLogin{}, err
	}
	_, err = dao.DisableSession(session.Token)
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return model.ProfileLogin{}, model.NewError(itn.ErrorUnknown, 500)
	}
	return Login(model.LoginRequest{
		Username: user.Username,
		Password: string(pass.Password),
	}, IP)
}

func GetUsersByLastLogin(monthsInterval int, includeOneDay bool) ([]model.User, error) {
	return dao.GetUsersByLastLogin(monthsInterval, includeOneDay)
}
