package controller

import (
	"encoding/json"
	"fmt"

	"github.com/mdhasib01/go-rest-starter/pkg/data"

	config "github.com/mdhasib01/go-rest-starter/config"
	dao "github.com/mdhasib01/go-rest-starter/dao"
	itn "github.com/mdhasib01/go-rest-starter/itn"
	model "github.com/mdhasib01/go-rest-starter/model"
	"github.com/mdhasib01/go-rest-starter/pkg/geoip"
	"github.com/mdhasib01/go-rest-starter/pkg/logger"
	notifier "github.com/mdhasib01/go-rest-starter/pkg/notifications"
	security "github.com/mdhasib01/go-rest-starter/security"
	utils "github.com/mdhasib01/go-rest-starter/utils"

	"strings"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/markbates/goth"
)

const (
	internal_key = "64FEC87F72EBA6B0BC915344D8A9EBC1"
)

func IsAuthorized(token string, idPermission int) (model.Session, error) {

	if len(token) < 2 {
		logger.GetLogger().LogErrors(fmt.Errorf(itn.ErrorTokenRequired), nil)
		return model.Session{}, model.NewError(itn.ErrorTokenRequired, 403)
	}

	ok, claims := validateToken(token)
	if !ok {
		logger.GetLogger().LogErrors(fmt.Errorf(itn.ErrorTokenInvalid), map[string]interface{}{"token": token})
		return model.Session{}, model.NewError(itn.ErrorTokenInvalid, 403)
	}

	session, _ := readToken(claims)

	err := dao.UpdateSession(token)
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return model.Session{}, err
	}
	session.Token = token

	if session.RoleId == model.SUPERADMIN || idPermission == 0 {
		return session, nil
	}

	if isAuthorized, err := hasPermission(session.RoleId, idPermission); !isAuthorized {
		logger.GetLogger().LogErrors(err, nil)
		return model.Session{}, err
	}

	return session, nil
}

func CreateNewSession(s model.Session) (model.Session, error) {
	_, err := dao.DisableSession(s.Token)
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return model.Session{}, err
	}

	token, s := createToken(s)
	s.Token = token

	err = dao.CreateSession(s)
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return model.Session{}, err
	}
	return s, nil
}

func hasPermission(idRole, idPermission int) (bool, error) {
	permissions, err := dao.GetRolePermissions(idRole)
	if err != nil {
		return false, err
	}

	for _, permission := range permissions {
		if permission.Id == idPermission {
			return true, nil
		}
	}

	return false, model.NewError(itn.ErrorNotAuthorized, 403)
}

func validateToken(token string) (bool, jwt.MapClaims) {
	claims := jwt.MapClaims{}
	tkn, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {

		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf(itn.ErrorTokenInvalid)
		}
		return []byte(internal_key), nil
	})

	if err != nil || !tkn.Valid {
		return false, jwt.MapClaims{}
	}

	return true, claims
}

func readToken(claims jwt.Claims) (model.Session, error) {

	var session model.Session
	b, err := json.Marshal(claims)
	if err != nil {
		return model.Session{}, fmt.Errorf(itn.ErrorTokenInvalid)
	}
	err = json.Unmarshal(b, &session)
	if err != nil {
		return model.Session{}, fmt.Errorf(itn.ErrorTokenInvalid)
	}
	return session, nil
}

func createToken(session model.Session) (string, model.Session) {
	claims := jwt.MapClaims{}
	claims["authorized"] = true
	claims["iduser"] = session.UserId
	claims["idrole"] = session.RoleId
	claims["createdat"] = time.Now()
	claims["expireat"] = time.Now().Add(time.Hour * 1)
	claims["isocountry"] = session.ISOCountry
	claims["buyerprofileid"] = session.BuyerProfileId
	claims["sellerprofileid"] = session.SellerProfileId
	claims["loggedinprofileid"] = session.LoggedInProfileId
	session.CreatedAt = time.Now()
	session.ExpirationDate = time.Now().Add(time.Hour)
	session.IsValid = true

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tkn, _ := token.SignedString([]byte(internal_key))
	return tkn, session
}

func Login(req model.LoginRequest, IP string) (model.ProfileLogin, error) {
	var login model.ProfileLogin
	var err error

	if err = validateLoginRequest(req); err != nil {
		return login, err
	}

	login.User, err = dao.Login(req.Username)
	if err != nil {
		return login, err
	}

	if err = validateLogin(login.User, req); err != nil {
		return login, err
	}

	err = dao.UpdateLastLogin(login.User.Id)
	if err != nil {
		return model.ProfileLogin{}, err
	}

	login.Role, err = dao.GetRole(login.User.IdRole)
	if err != nil {
		return model.ProfileLogin{}, err
	}

	login.Plan, login.EvaluationUsed, _ = dao.GetUserPlan(login.User.Id)

	session := model.Session{
		RoleId:  login.User.IdRole,
		UserId:  login.User.Id,
		IsValid: true,
	}

	login.Token, session = createToken(session)
	session.Token = login.Token

	err = dao.CreateSession(session)
	if err != nil {
		return model.ProfileLogin{}, err
	}

	// this to prevent him from login with the temporary password more than once
	if login.User.IsPasswordTemporary {
		randpass := utils.GenerateRandomString(15)
		newpassword, _ := utils.BeforeSave(randpass)

		_, err = dao.ResetPassword(login.User.Id, newpassword)
		if err != nil {
			return model.ProfileLogin{}, err
		}
		return model.ProfileLogin{Login: model.Login{User: model.User{IsPasswordTemporary: true}, Token: login.Token}}, nil
	}

	go func() {
		// get the user location
		loc, err := geoip.GetLocation(IP)
		if err != nil {
			logger.GetLogger().LogErrors(err, nil)
			return
		}

		isTrusted, err := dao.IsTrustedLocation(loc.City.GeoNameId, loc.Country.GeoNameId, login.User.Id)
		if err != nil {
			logger.GetLogger().LogErrors(err, nil)
			return
		}

		if !isTrusted {
			trustCode := uuid.NewString()
			// insert location
			location := model.Location{
				Country:      loc.Country.Names["en"],
				CountryGeoID: loc.Country.GeoNameId,
				City:         loc.City.Names["en"],
				CityGeoID:    loc.City.GeoNameId,
				IsTrusted:    false,
				UserID:       login.User.Id,
				TrustCode:    trustCode,
			}

			err = dao.InsertLocation(location)
			if err != nil {
				logger.GetLogger().LogErrors(err, nil)
				return
			}

			// send email verification
			body := fmt.Sprintf(`Hello %s, <br> <br>
			We have detected a login attempt from a new location. <br>
			Please verify your account by clicking the link below: <br>
			<a href="%s/location/trust/%s?token=%s">Verify Location</a> <br> <br>
			Thank you, <br>
			GOPG Server `, login.User.Firstname, config.Param.ServerBaseURL, trustCode, login.Token)
			mail := model.Mail{
				ReceiverEmail: login.User.Email,
				ReceiverName:  login.User.Firstname,
				SUBJECT:       "GoPG Server Location Verification",
				BODY:          body,
				SENDER:        config.Param.Account.MAIL_SENDER,
				SENDER_NAME:   config.Param.Account.MAIL_SENDER_NAME,
			}

			err = notifier.ES.SendEmail(mail)
			if err != nil {
				logger.GetLogger().LogErrors(err, nil)
				return
			}
		}

	}()

	return login, nil
}

func FetchLoginData(session model.Session) (model.Login, error) {
	var login model.Login
	var err error
	login.User, err = dao.GetUser(session.UserId, session)
	if err != nil {
		return login, err
	}

	login.Role, err = dao.GetRole(login.User.IdRole)
	if err != nil {
		return model.Login{}, err
	}

	if err != nil {
		return model.Login{}, err
	}

	login.Token = session.Token
	return login, nil

}

func LoginProfile(profileType string, session model.Session) (model.ProfileLogin, error) {

	// deactivate 3 profiles
	DeactivateProfiles(&session)

	var login model.ProfileLogin
	var err error

	login.User, err = dao.GetUser(session.UserId, session)
	if err != nil {
		return model.ProfileLogin{}, err
	}

	login.Role, err = dao.GetRole(login.User.IdRole)
	if err != nil {
		return model.ProfileLogin{}, err
	}

	login.Plan, login.EvaluationUsed, _ = dao.GetUserPlan(login.User.Id)

	if err != nil {
		return model.ProfileLogin{}, err
	}

	session, err = CreateNewSession(session)
	if err != nil {
		return model.ProfileLogin{}, err
	}

	login.LoggedInProfileType = profileType

	login.Token = session.Token

	return login, nil
}

func ForgotPassword(body model.ForgotPasswordRequest) (model.ForgotPasswordResponse, error) {
	if utils.IsEmpty(body.Email) {
		return model.ForgotPasswordResponse{}, model.NewError(itn.ErrorEmailRequired, 400)
	}

	user, err := dao.GetUserByEmail(body.Email)
	if err != nil {
		return model.ForgotPasswordResponse{}, err
	}

	randpass := utils.GenerateRandomString(15)
	newpassword, _ := utils.BeforeSave(randpass)

	_, err = dao.ResetPassword(user.Id, newpassword)
	if err != nil {
		return model.ForgotPasswordResponse{}, err
	}

	return model.ForgotPasswordResponse{Message: "A new password has been sent to your email", NewPassword: data.HiddenJsonString(randpass)}, nil
}

func validateLogin(user model.User, req model.LoginRequest) error {
	if user.Id == 0 {
		logger.GetLogger().LogErrors(fmt.Errorf("user not found"), nil)
		return model.NewError(itn.ErrorLoginFailed, 403)
	}

	if !user.IsVerified {
		logger.GetLogger().LogErrors(fmt.Errorf("account not verified"), nil)
		return model.NewError(itn.ErrorAccountNotVerified, 403)
	}

	if security.VerifyPassword(string(user.Password), req.Password) != nil {
		logger.GetLogger().LogErrors(fmt.Errorf("wrong password"), nil)
		return model.NewError(itn.ErrorLoginFailed, 403)
	}
	return nil
}

func validateLoginRequest(req model.LoginRequest) error {
	if utils.IsEmpty(req.Username) {
		return model.NewError(itn.ErrorUsernameRequired, 400)
	}

	if utils.IsEmpty(req.Password) {
		return model.NewError(itn.ErrorPasswordRequired, 400)
	}
	return nil
}

func Logout(token string) (int, error) {
	return dao.DisableSession(token)
}

func SocialAuth(user goth.User) (model.Login, error) {

	_, isExist := dao.IsExistInDb(user.Email, "users", "email")

	if isExist {
		return ThirdPartyLogin(user.Email)
	}

	newUser, err := prepareAccountInsertion(user)
	if err != nil {
		return model.Login{}, err
	}

	newUser.IsVerified = true

	err = dao.CreateUser(&newUser)

	if err != nil {
		return model.Login{}, err
	}
	return ThirdPartyLogin(user.Email)

}

func prepareAccountInsertion(user goth.User) (model.User, error) {
	// var userImageURL string
	// if user.Provider == "google" {
	// 	userImageURL = user.RawData["picture"].(string)
	// } else if user.Provider == "facebook" {
	// 	userImageURL = "https://graph.facebook.com/" + user.UserID + "/picture?type=large"
	// }

	newUser := model.User{Firstname: user.FirstName, Lastname: user.LastName, Email: user.Email,
		IdRole: model.USERROLE, CreatedBy: model.OUTBOUND_USER,
		Username: user.Email}

	err := validateSocialAccountData(newUser)

	if err != nil {
		return model.User{}, err
	}

	return newUser, nil

}

func ThirdPartyLogin(email string) (model.Login, error) {
	var login model.Login
	var err error

	login.User, err = dao.Login(email)
	if err != nil {
		return login, err
	}

	login.Role, err = dao.GetRole(login.User.IdRole)
	if err != nil {
		return model.Login{}, err
	}

	session := model.Session{
		RoleId:  login.User.IdRole,
		UserId:  login.User.Id,
		IsValid: true,
	}

	login.Token, session = createToken(session)

	session.Token = login.Token

	err = dao.CreateSession(session)
	if err != nil {
		return model.Login{}, err
	}
	return login, nil
}

func validateSocialAccountData(user model.User) error {
	if utils.IsEmpty(user.Firstname) {
		return model.NewError(itn.ErrorFirstnameRequired, 400)
	}

	if utils.IsEmpty(user.Lastname) {
		return model.NewError(itn.ErrorLastnameRequired, 400)
	}

	if utils.IsEmpty(user.Email) {
		return model.NewError(itn.ErrorEmailRequired, 400)
	}

	if user.IdRole < 1 {
		return model.NewError(itn.ErrorRoleRequired, 400)
	}

	return nil
}

func Register(user model.User, file model.FileRequest, IP string) (model.RegisterResponse, error) {

	// id, err := dao.IsNewUser(user, model.Session{})
	// if err != nil {
	// 	return model.RegisterResponse{}, err
	// }
	// if id != 0 {
	// 	user, err := dao.GetUserByEmail(user.Email)
	// 	if err != nil {
	// 		return model.RegisterResponse{}, err
	// 	}
	// 	return model.RegisterResponse{Message: "An Activation link has been sent to your email", User: user}, nil
	// }
	// parse base64 to file

	// remove prefix of the base 64

	err := prepareUser(&user)
	if err != nil {
		return model.RegisterResponse{}, err
	}

	err = dao.CreateUser(&user)

	if err != nil {
		return model.RegisterResponse{}, err
	}

	go func() {
		loc, err := geoip.GetLocation(IP)
		if err != nil {
			logger.GetLogger().LogErrors(err, nil)
			return
		}

		location := model.Location{
			Country:      loc.Country.Names["en"],
			CountryGeoID: loc.Country.GeoNameId,
			City:         loc.City.Names["en"],
			CityGeoID:    loc.City.GeoNameId,
			IsTrusted:    true,
			UserID:       user.Id,
		}

		// insert trusted location
		err = dao.InsertLocation(location)
		if err != nil {
			logger.GetLogger().LogErrors(err, nil)
			return
		}
	}()
	role, err := dao.GetRole(user.IdRole)
	if err != nil {
		return model.RegisterResponse{}, err
	}
	user.RoleName = role.Name

	tempVerificationCode := user.VerificationCode

	go func() {
		body := fmt.Sprintf(model.ConfirmationEmailBody, user.Firstname,
			config.Param.ServerBaseURL+"/verify/"+tempVerificationCode)

		mail := model.Mail{
			ReceiverEmail: user.Email,
			ReceiverName:  user.Firstname,
			SUBJECT:       "GoPgDB Account Verification",
			BODY:          body,
			SENDER:        config.Param.Account.MAIL_SENDER,
			SENDER_NAME:   config.Param.Account.MAIL_SENDER_NAME,
		}

		// send email verification
		err = notifier.ES.SendEmail(mail)
		if err != nil {
			logger.GetLogger().LogErrors(err, nil)
			return
		}
	}()

	user.VerificationCode = ""

	return model.RegisterResponse{Message: "An Activation link has been sent to your email", User: user}, nil

}

func VerifyAccount(code string) (string, error) {
	if utils.IsEmpty(code) {
		return "", model.NewError("Wrong Verification Code", 400)
	}

	firstName, err := dao.VerifyUser(code)
	if err != nil {
		return "", err
	}
	return firstName, err
}

func prepareUser(user *model.User) error {
	user.Email = strings.ToLower(user.Email)

	err := validateRegisterationBody(user)
	if err != nil {
		return err
	}

	if _, isExist := dao.IsExistInDb(user.Email, "users", "email"); isExist {
		return model.NewError(itn.ErrorEmailExists, 409)
	}

	if _, isExist := dao.IsExistInDb(user.Username, "users", "username"); isExist {
		return model.NewError(itn.ErrorUsernameTaken, 409)
	}

	hashedPassword, err := utils.BeforeSave(string(user.Password))
	if err != nil {
		return err
	}

	user.Password = data.HiddenJsonString(hashedPassword)

	user.VerificationCode = uuid.NewString()

	return nil
}

func validateRegisterationBody(user *model.User) error {
	if utils.IsEmpty(user.Firstname) {
		return model.NewError(itn.ErrorFirstnameRequired, 400)
	}

	if utils.IsEmpty(user.Lastname) {
		return model.NewError(itn.ErrorLastnameRequired, 400)
	}

	if utils.IsEmpty(user.Email) {
		return model.NewError(itn.ErrorEmailRequired, 400)
	}

	if !utils.IsEmail(user.Email) {
		return model.NewError(itn.ErrorEmailInvalid, 400)
	}

	if utils.IsEmpty(user.Username) {
		return model.NewError(itn.ErrorUsernameRequired, 400)
	}

	if utils.IsEmpty(string(user.Password)) {
		return model.NewError(itn.ErrorPasswordRequired, 400)
	}

	if len(user.Password) < 8 {
		return model.NewError(itn.ErrorPasswordTooShort, 400)
	}

	if string(user.Password) != string(user.ConfirmPassword) {
		return model.NewError(itn.ErrorPasswordNotMatch, 400)
	}

	if !utils.IsPasswordStrong(string(user.Password)) {
		return model.NewError(itn.ErrorPasswordNotStrong, 400)
	}

	if utils.IsEmpty(user.Country) {
		return model.NewError(itn.ErrorCountryRequired, 400)
	}

	return nil
}

func GetProfileCompletePercentage(session model.Session) (int, error) {
	if session.LoggedInProfileId == 0 {
		return 0, model.NewError(itn.ErrorUnauthorizedOperation, 401)
	}

	if session.LoggedInProfileId == session.BuyerProfileId {
		return dao.GetBuyerProfileCompletePercentage(session)
	}

	if session.LoggedInProfileId == session.SellerProfileId {
		return dao.GetSellerProfileCompletePercentage(session)
	}

	return 0, model.NewError(itn.ErrorUnauthorizedOperation, 401)
}

func DeactivateProfiles(session *model.Session) {
	session.BuyerProfileId = 0
	session.SellerProfileId = 0
	session.LoggedInProfileId = 0
}
