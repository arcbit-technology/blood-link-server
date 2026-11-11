package rest

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"runtime/debug"
	"time"

	itn "github.com/mdhasib01/go-rest-starter/itn"
	"github.com/mdhasib01/go-rest-starter/pkg/logger"

	"github.com/gorilla/mux"
)

func router() *mux.Router {
	router := mux.NewRouter().StrictSlash(true)
	router.Methods("GET").Path("/ping").Name("CheckApiStatus").HandlerFunc(CheckApiStatus)

	//auth
	router.Methods("POST").Path("/login").Name("login").HandlerFunc(Login)
	router.Methods("GET").Path("/login/{profiletype}").Name("loginProfile").HandlerFunc(LoginProfile)
	router.Methods("POST").Path("/register").Name("register").HandlerFunc(Register)
	router.Methods("GET").Path("/auth/fetchlogindata").Name("fetchdata").HandlerFunc(FetchLoginData)
	router.Methods("GET").Path("/auth/{provider}").Name("auth").HandlerFunc(SocialAuth)
	router.Methods("GET").Path("/auth/{provider}/callback").Name("authCallback").HandlerFunc(SocialAuthCallback)
	router.Methods("PUT").Path("/logout").Name("logout").HandlerFunc(Logout)
	router.Methods("GET").Path("/iftokenvalid").Name("iftokenvalid").HandlerFunc(IfTokenValid)
	router.Methods("PUT").Path("/changepassword/{id:[0-9]+}").Name("changePassword").HandlerFunc(ChangeOrResetPassword)
	router.Methods("PUT").Path("/resetpassword/{id:[0-9]+}").Name("resetPassword").HandlerFunc(ChangeOrResetPassword)
	router.Methods("GET").Path("/verify/{code}").Name("verify").HandlerFunc(VerifyAccount)
	router.Methods("POST").Path("/forgotpassword").Name("forgotPassword").HandlerFunc(ForgotPassword)
	router.Methods("PUT").Path("/changetemppassword").Name("changetemppassword").HandlerFunc(ChangeTempPassword)
	router.Methods("GET").Path("/profilecomplete").Name("profilecomplete").HandlerFunc(GetProfileCompletePercentage)

	// serve static files
	router.Methods("GET").PathPrefix("/static/").Name("static").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./images"))))

	router.Methods("GET").Path("/myip").Name("myip").HandlerFunc(GetMyIP)

	//user
	router.Methods("POST").Path("/user").Name("createUser").HandlerFunc(CreateUser)
	router.Methods("PUT").Path("/user/{id:[0-9]+}").Name("updateUser").HandlerFunc(UpdateUser)
	router.Methods("GET").Path("/users").Name("getUsers").HandlerFunc(GetAllUsers)
	router.Methods("GET").Path("/user/{id:[0-9]+}").Name("getAllUsers").HandlerFunc(GetUser)
	router.Methods("DELETE").Path("/user/{id:[0-9]+}").Name("deleteUSer").HandlerFunc(DeleteUser)
	router.Methods("PUT").Path("/user/{id:[0-9]+}/disable").Name("disable user").HandlerFunc(ToggleUserStatus)
	router.Methods("PUT").Path("/user/{id:[0-9]+}/enable").Name("enable user").HandlerFunc(ToggleUserStatus)

	// Donors (public). "/doners" is the spelling the UI uses; "/donors" is an alias.
	router.Methods("GET").Path("/doners").Name("getDonors").HandlerFunc(GetDonors)
	router.Methods("GET").Path("/donors").Name("getDonorsAlias").HandlerFunc(GetDonors)
	router.Methods("GET").Path("/doner/{id:[0-9]+}").Name("getDonor").HandlerFunc(GetDonor)
	router.Methods("GET").Path("/donor/{id:[0-9]+}").Name("getDonorAlias").HandlerFunc(GetDonor)

	// Blood request posts: listing is public, creating / interest needs login
	router.Methods("GET").Path("/posts").Name("getPosts").HandlerFunc(GetPosts)
	router.Methods("GET").Path("/post/{id:[0-9]+}").Name("getPost").HandlerFunc(GetPost)
	router.Methods("POST").Path("/post").Name("createPost").HandlerFunc(CreatePost)
	router.Methods("PUT").Path("/post/{id:[0-9]+}/interest").Name("togglePostInterest").HandlerFunc(TogglePostInterest)

	// Locations
	router.Methods("GET").Path("/divisions").Name("getDivisions").HandlerFunc(GetDivisions)
	router.Methods("GET").Path("/districts").Name("getDistricts").HandlerFunc(GetDistricts)

	//role
	// router.Methods("POST").Path("/role").Name("CreateRole").HandlerFunc(CreateRole)
	// router.Methods("GET").Path("/roles").Name("GetAllRole").HandlerFunc(GetAllRole)
	// router.Methods("GET").Path("/role/{id:[0-9]+}").Name("GetRole").HandlerFunc(GetRole)
	// router.Methods("PUT").Path("/role/{id:[0-9]+}").Name("updateRole").HandlerFunc(UpdateRole)
	// router.Methods("DELETE").Path("/role/{id:[0-9]+}").Name("DeleteRole").HandlerFunc(DeleteRole)
	// router.Methods("GET").Path("/permissions").Name("GetPermissions").HandlerFunc(GetPermissions)
	// router.Methods("PUT").Path("/role/{roleid:[0-9]+}/{permissionid:[0-9]+}").Name("associateRoleToPermission").HandlerFunc(AddPermisionToRole)
	// router.Methods("DELETE").Path("/role/{roleid:[0-9]+}/{permissionid:[0-9+]}").Name("DesassociatePermission").HandlerFunc(DesassociatePermission)

	return router
}

// responseWriter wraps http.ResponseWriter to capture the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := rw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("underlying ResponseWriter does not support hijacking")
	}
	return hijacker.Hijack()
}

// LoggingMiddleware logs request details and response status
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		start := time.Now()

		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rw, r)

		logger.GetLogger().LogInfo(fmt.Sprintf(
			"method=%s, url=%s, status=%d, duration=%s IP=%s",
			r.Method,
			r.URL.Path,
			rw.statusCode,
			time.Since(start),
			r.Header.Get("X-Fowarded-For"),
		), nil)

	})
}
func InitializeRouter() handler {
	router := router()
	h := handler{Router: router}
	router.Use(LoggingMiddleware)
	return h
}

// WriteHeader captures the status code
func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

// Write captures the status code if WriteHeader hasn't been called yet
func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.statusCode == http.StatusOK {
		rw.statusCode = http.StatusOK
	}
	return rw.ResponseWriter.Write(b)
}

type handler struct {
	Router *mux.Router
}

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		err := recover()
		if err != nil {
			//e := err.(error)
			stack := debug.Stack()
			log.Println(err, string(stack))
			w.WriteHeader(http.StatusInternalServerError)
			res := make(map[string]interface{})
			res["error"] = itn.ErrorUnknown
			res["result"] = nil
			json.NewEncoder(w).Encode(res)
		}
	}()

	h.Router.ServeHTTP(w, r)
}
