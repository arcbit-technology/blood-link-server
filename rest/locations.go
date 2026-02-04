package rest

import (
	"net/http"

	model "github.com/mdhasib01/go-rest-starter/model"
)

func GetDivisions(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	JSON(w, model.DIVISIONS, nil)
}

func GetDistricts(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	JSON(w, model.DISTRICTS, nil)
}
