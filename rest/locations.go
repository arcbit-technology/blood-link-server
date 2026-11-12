package rest

import (
	"net/http"
	"strings"

	model "github.com/mdhasib01/go-rest-starter/model"
)

func GetDivisions(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	JSON(w, model.DIVISIONS, nil)
}

// GetUpazilas returns the upazilas / thanas of ?district=, or the whole
// district -> upazilas map when no district is given.
func GetUpazilas(w http.ResponseWriter, r *http.Request) {
	district := strings.TrimSpace(r.URL.Query().Get("district"))
	if district == "" {
		JSON(w, model.UPAZILAS, nil)
		return
	}
	list, ok := model.UPAZILAS[model.District(district)]
	if !ok {
		JSON(w, nil, invalidDataError)
		return
	}
	JSON(w, list, nil)
}

func GetDistricts(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	JSON(w, model.DISTRICTS, nil)
}
