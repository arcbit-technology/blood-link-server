package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	security "github.com/mdhasib01/go-rest-starter/security"

	"github.com/gorilla/sessions"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/facebook"
	"github.com/markbates/goth/providers/google"
)

var rand uint32
var randmu sync.Mutex

var Pathconfigparam string
var PathMedia string
var ApiHost string
var LinkUserCreate string
var LinkStripe string
var LinkEmailer string
var Param appConfig
var ImagesPath string

type appConfig struct {
	ServerPort           string             `json:"serverPort"`
	ServerBaseURL        string             `json:"serverBaseURL"`
	ClientBaseURL        string             `json:"clientBaseURL"`
	ApiKey               string             `json:"apiKey"`
	BaseHost             string             `json:"baseHost"`
	BasePort             string             `json:"basePort"`
	BaseName             string             `json:"baseName"`
	BaseUser             string             `json:"baseUser"`
	BasePassword         string             `json:"basePassword"`
	BaseDriver           string             `json:"baseDriver"`
	ConnString           string             `json:"connString"`
	GoogleClientID       string             `json:"googleClientID"`
	GoogleClientSecret   string             `json:"googleClientSecret"`
	LinkedinClientID     string             `json:"linkedinClientID"`
	LinkedinClientSecret string             `json:"linkedinClientSecret"`
	FacebookClientID     string             `json:"facebookClientID"`
	FacebookClientSecret string             `json:"facebookClientSecret"`
	SecretKey            string             `json:"secretKey"`
	MaxAgeInHours        string             `json:"maxAgeInHours"`
	Protocol             string             `json:"protocol"`
	SessionSecret        string             `json:"sessionSecret"`
	Services             map[string]Service `json:"services"`
	Geoip2UserID         string             `json:"geoip2UserID"`
	Geoip2LicenseKey     string             `json:"geoip2LicenseKey"`

	StripeSecretKey      string `json:"stripeSecretKey"`
	StripePublishableKey string `json:"stripePublishableKey"`
	StripeWebhookSecret  string `json:"stripeWebhookSecret"`

	SMTP_DETAILS SMTP_DETAILS `json:"SMTP_DETAILS"`

	Account Account `json:"ACCOUNT"`

	VOIP_DETAIL VOIP_CREDENTIALS `json:"VOIP_CREDENTIALS"`
}

type Account struct {
	Name             string `json:"NAME"`
	MAIL_SENDER      string `json:"MAIL_SENDER"`
	MAIL_SENDER_NAME string `json:"MAIL_SENDER_NAME"`
}

type SMTP_DETAILS struct {
	SMTP_SERVER      string `json:"SMTP_SERVER"`
	SMTP_SERVER_PORT string `json:"SMTP_SERVER_PORT"`
	SMTP_USER        string `json:"SMTP_USER"`
	SMTP_PASS        string `json:"SMTP_PASS"`
}

type VOIP_CREDENTIALS struct {
	USER_NAME     string `json:"USERNAME"`
	PASSWORD      string `json:"PASSWORD"`
	VOIP_DID      string `json:"VOIP_DID"`
	KEY_ENCRYPTED bool   `json:"KEY_ENCRYPTED"`
}

type Service struct {
	Hostname string `json:"hostname"`
	Protocol string `json:"protocol"`
	BaseURL  string `json:"baseURL"`
}

func readConfigParam() (appConfig, error) {
	var param appConfig

	plan, err := ioutil.ReadFile(filepath.Join(Pathconfigparam, "config.json"))
	if err != nil {
		Display(err.Error())
		return appConfig{}, err
	}
	err = json.Unmarshal(plan, &param)
	if err != nil {
		Display(err.Error())
		return appConfig{}, err
	}

	return param, nil
}

func Display(str string) {
	fmt.Println(str)
}

func DefaultPassword() string {
	return "azouken"
}

func nextRandom() string {
	randmu.Lock()
	r := rand
	if r == 0 {
		r = reseed()
	}
	r = r*1664525 + 1013904223 // constants from Numerical Recipes
	rand = r
	randmu.Unlock()
	return strconv.Itoa(int(1e9 + r%1e9))[1:]
}

func reseed() uint32 {
	return uint32(time.Now().UnixNano() + int64(os.Getpid()))
}

func TempFile(dir, pattern string) (f *os.File, err error) {
	if dir == "" {
		dir = os.TempDir()
	}

	prefix, suffix := prefixAndSuffix(pattern)

	nconflict := 0
	for i := 0; i < 10000; i++ {
		name := filepath.Join(dir, prefix+nextRandom()+suffix)
		f, err = os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
		if os.IsExist(err) {
			if nconflict++; nconflict > 10 {
				randmu.Lock()
				rand = reseed()
				randmu.Unlock()
			}
			continue
		}
		break
	}
	return
}

func prefixAndSuffix(pattern string) (prefix, suffix string) {
	if pos := strings.LastIndex(pattern, "*"); pos != -1 {
		prefix, suffix = pattern[:pos], pattern[pos+1:]
	} else {
		prefix = pattern
	}
	return
}

func TempDir(dir, pattern string) (name, pathname string, err error) {
	if dir == "" {
		dir = os.TempDir()
	}

	prefix, suffix := prefixAndSuffix(pattern)

	nconflict := 0
	pathname = prefix + nextRandom() + suffix
	fmt.Println(pathname)
	for i := 0; i < 10000; i++ {
		try := filepath.Join(dir, pathname)
		err = os.Mkdir(try, 0700)
		if os.IsExist(err) {
			if nconflict++; nconflict > 10 {
				randmu.Lock()
				rand = reseed()
				randmu.Unlock()
			}
			continue
		}
		if os.IsNotExist(err) {
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				return "", "", err
			}
		}
		if err == nil {
			name = try
		}
		break
	}
	return
}

func CreateFileName(pref, path string) (string, string) {
	fmt.Println(nextRandom())
	prefix, suffix := prefixAndSuffix(pref)
	filename := prefix + nextRandom() + suffix
	name := filepath.Join(path, filename)

	return name, filename
}

//***********************************************************

func CreateFolderIfNotExists(path string) string {

	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.Mkdir(path, 0755)
		if err != nil {
			return err.Error()
		}
	}
	return ""
}

func RemoveAllFile(path string) string {

	err := os.RemoveAll(path)
	if err != nil {
		return err.Error()
	}
	return ""
}

func CreateFile(filename string) error {
	var path = filepath.Join(Pathconfigparam, filename)

	var _, err = os.Stat(path)

	// create file if not exists
	if os.IsNotExist(err) {
		var file, err = os.Create(path)
		if err != nil {
			return err
		}
		defer file.Close()
	}
	return nil
}

func decryptParam(param appConfig) appConfig {

	param.ServerPort = security.Decrypt(param.ServerPort)
	param.ApiKey = security.Decrypt(param.ApiKey)
	param.BaseDriver = security.Decrypt(param.BaseDriver)
	param.BaseHost = security.Decrypt(param.BaseHost)
	param.BaseName = security.Decrypt(param.BaseName)
	param.BasePassword = security.Decrypt(param.BasePassword)
	param.BasePort = security.Decrypt(param.BasePort)
	param.BaseUser = security.Decrypt(param.BaseUser)
	param.ConnString = security.Decrypt(param.ConnString)
	param.GoogleClientID = security.Decrypt(param.GoogleClientID)
	param.GoogleClientSecret = security.Decrypt(param.GoogleClientSecret)
	param.SecretKey = security.Decrypt(param.SecretKey)
	param.MaxAgeInHours = security.Decrypt(param.MaxAgeInHours)
	param.Protocol = security.Decrypt(param.Protocol)
	param.LinkedinClientID = security.Decrypt(param.LinkedinClientID)
	param.LinkedinClientSecret = security.Decrypt(param.LinkedinClientSecret)
	param.FacebookClientID = security.Decrypt(param.FacebookClientID)
	param.FacebookClientSecret = security.Decrypt(param.FacebookClientSecret)
	param.SessionSecret = security.Decrypt(param.SessionSecret)
	param.SMTP_DETAILS.SMTP_SERVER = security.Decrypt(param.SMTP_DETAILS.SMTP_SERVER)
	param.SMTP_DETAILS.SMTP_SERVER_PORT = security.Decrypt(param.SMTP_DETAILS.SMTP_SERVER_PORT)
	param.SMTP_DETAILS.SMTP_USER = security.Decrypt(param.SMTP_DETAILS.SMTP_USER)
	param.SMTP_DETAILS.SMTP_PASS = security.Decrypt(param.SMTP_DETAILS.SMTP_PASS)
	param.ServerBaseURL = security.Decrypt(param.ServerBaseURL)
	param.ClientBaseURL = security.Decrypt(param.ClientBaseURL)
	param.Account.Name = security.Decrypt(param.Account.Name)
	param.Account.MAIL_SENDER = security.Decrypt(param.Account.MAIL_SENDER)
	param.Account.MAIL_SENDER_NAME = security.Decrypt(param.Account.MAIL_SENDER_NAME)
	param.Geoip2UserID = security.Decrypt(param.Geoip2UserID)
	param.Geoip2LicenseKey = security.Decrypt(param.Geoip2LicenseKey)
	param.StripeSecretKey = security.Decrypt(param.StripeSecretKey)
	param.StripePublishableKey = security.Decrypt(param.StripePublishableKey)
	param.StripeWebhookSecret = security.Decrypt(param.StripeWebhookSecret)

	for key, service := range param.Services {
		service.Hostname = security.Decrypt(service.Hostname)
		service.Protocol = security.Decrypt(service.Protocol)
		service.BaseURL = fmt.Sprintf("%s://%s", service.Protocol, service.Hostname)
		param.Services[key] = service
	}

	if param.VOIP_DETAIL.KEY_ENCRYPTED {
		param.VOIP_DETAIL.USER_NAME = security.Decrypt(param.VOIP_DETAIL.USER_NAME)
		param.VOIP_DETAIL.PASSWORD = security.Decrypt(param.VOIP_DETAIL.PASSWORD)
		param.VOIP_DETAIL.VOIP_DID = security.Decrypt(param.VOIP_DETAIL.VOIP_DID)
	}

	//model.Param.URL_EMAILER = security.Decrypt(model.Param.URL_EMAILER)

	// model.Param.SMTP_DETAILS.SMTP_SERVER = security.Decrypt(model.Param.SMTP_DETAILS.SMTP_SERVER)
	// model.Param.SMTP_DETAILS.SMTP_SERVER_PORT = security.Decrypt(model.Param.SMTP_DETAILS.SMTP_SERVER_PORT)
	// model.Param.SMTP_DETAILS.SMTP_USER = security.Decrypt(model.Param.SMTP_DETAILS.SMTP_USER)
	// model.Param.SMTP_DETAILS.SMTP_PASS = security.Decrypt(model.Param.SMTP_DETAILS.SMTP_PASS)
	// model.Param.Account.Name = security.Decrypt(model.Param.Account.Name)
	// model.Param.Account.AppUrl = security.Decrypt(model.Param.Account.AppUrl)
	// model.Param.Account.MAIL_SENDER = security.Decrypt(model.Param.Account.MAIL_SENDER)
	// model.Param.Account.MAIL_SENDER_NAME = security.Decrypt(model.Param.Account.MAIL_SENDER_NAME)
	// model.Param.Account.DESTINATION_RECEIPT.DESTINATIONS_NAME = security.Decrypt(model.Param.Account.DESTINATION_RECEIPT.DESTINATIONS_NAME)
	// model.Param.Account.DESTINATION_RECEIPT.DESTINATION_ADR = security.Decrypt(model.Param.Account.DESTINATION_RECEIPT.DESTINATION_ADR)

	// model.Param.Account.DESTINATION_RECEIPT.CC_Tab.CC_NAME = security.Decrypt(model.Param.Account.DESTINATION_RECEIPT.CC_Tab.CC_NAME)
	// model.Param.Account.DESTINATION_RECEIPT.CC_Tab.CC_ADR = security.Decrypt(model.Param.Account.DESTINATION_RECEIPT.CC_Tab.CC_ADR)

	// model.Param.Account.DESTINATION_RECEIPT.BCC_Tab.BCC_NAME = security.Decrypt(model.Param.Account.DESTINATION_RECEIPT.BCC_Tab.BCC_NAME)
	// model.Param.Account.DESTINATION_RECEIPT.BCC_Tab.BCC_ADR = security.Decrypt(model.Param.Account.DESTINATION_RECEIPT.BCC_Tab.BCC_ADR)
	return param
}

func InitConfig(path string) error {
	PathMedia = path
	Pathconfigparam = path
	CreateAssetsFolders(PathMedia, "assets")

	param, err := readConfigParam()
	if err != nil {
		return err
	}

	param = decryptParam(param)
	Param = replaceParam(param)

	return nil
}

func PingServices() error {
	for key := range Param.Services {
		url := Param.Services[key].BaseURL + "/ping"
		err := PingService(url)
		if err != nil {
			return err
		}
	}
	return nil
}

func PingService(url string) error {
	log.Println("Pinging service", url)
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	// Create an HTTP client
	client := &http.Client{}

	// Make the request
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("failed to ping service %s, err: %s", url, err.Error())
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("Service %s is not available", url)
	}

	log.Println("Successfully pinged service", url)

	return nil
}

func CreateDocsFolders(basePath string, baseFolderName string) string {
	err := CreateFolderIfNotExists(filepath.Join(basePath, baseFolderName))
	if err != "" {
		return err
	}
	internalFolders := []string{"buyers", "sellers", "professionals", "businesses", "contracts", "jobs"}

	for _, folder := range internalFolders {
		err := CreateFolderIfNotExists(filepath.Join(basePath, baseFolderName, folder))
		if err != "" {
			return err
		}
	}

	return ""
}

func CreateAssetsFolders(basePath string, baseFolderName string) string {
	err := CreateFolderIfNotExists(filepath.Join(basePath, baseFolderName))
	if err != "" {
		return err
	}
	internalFolders := []string{"excel"}

	for _, folder := range internalFolders {
		err := CreateFolderIfNotExists(filepath.Join(basePath, baseFolderName, folder))
		if err != "" {
			return err
		}
	}

	return ""
}

func replaceParam(param appConfig) appConfig {
	param.ConnString = strings.Replace(param.ConnString,
		"$HOST", param.BaseHost, -1)
	param.ConnString = strings.Replace(param.ConnString,
		"$PORT", param.BasePort, -1)
	param.ConnString = strings.Replace(param.ConnString,
		"$USER", param.BaseUser, -1)
	param.ConnString = strings.Replace(param.ConnString,
		"$PSWD", param.BasePassword, -1)
	param.ConnString = strings.Replace(param.ConnString,
		"$DBNAME", param.BaseName, -1)

	return param
}

func InitAuthProviders() error {

	key := Param.SessionSecret
	maxAge, err := strconv.Atoi(Param.MaxAgeInHours)

	if err != nil {
		return err
	}
	maxAge *= 60 * 60
	store := sessions.NewCookieStore([]byte(key))
	store.MaxAge(maxAge)
	store.Options.Path = "/"
	store.Options.HttpOnly = true
	store.Options.Secure = true

	gothic.Store = store

	redirectUrl := fmt.Sprintf("%s/auth", Param.ServerBaseURL)

	googleRedirectUrl := fmt.Sprintf("%s%s", redirectUrl, "/google/callback")
	// linkedinRedirectUrl := fmt.Sprintf("%s%s", redirectUrl, "/linkedin/callback")
	facebookRedirectUrl := fmt.Sprintf("%s%s", redirectUrl, "/facebook/callback")
	goth.UseProviders(
		google.New(Param.GoogleClientID, Param.GoogleClientSecret, googleRedirectUrl, "email", "profile"),
		// linkedin.New(config.Param.LinkedinClientID, config.Param.LinkedinClientSecret, linkedinRedirectUrl, "email", "profile"),
		facebook.New(Param.FacebookClientID, Param.FacebookClientSecret, facebookRedirectUrl, "email", "public_profile"),
	)
	return nil
}
