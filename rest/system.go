package rest

import (
	"encoding/json"
	"net/http"
)

// CheckApiStatus godoc
// @Summary check api status
// @Description this route is used to check if the api is running
// @Tags system
// @Produce  plain
// @Success 200 {string} string "GoPg Server API is listening"
// @Router /ping [get]
func CheckApiStatus(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	w.Write([]byte("GoPg Server API is listening"))
}

// GetMyIP godoc
// @Summary get requester ip
// @Description this route is used to get the ip address of the requester
// @Tags system
// @Produce  json
// @Success 200 {object} map[string]string
// @Router /myip [get]
func GetMyIP(w http.ResponseWriter, r *http.Request) {
	ip := r.Header.Get("X-Real-IP")
	remoteAddr := r.RemoteAddr
	var response = make(map[string]string)
	forwardedIP := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		response["realip"] = "No real IP found"
	} else {
		response["realip"] = ip
	}

	if forwardedIP == "" {
		response["forwardedip"] = "No forwarded IP found"
	} else {
		response["forwardedip"] = forwardedIP
	}

	response["remoteaddr"] = remoteAddr

	json.NewEncoder(w).Encode(response)
}
