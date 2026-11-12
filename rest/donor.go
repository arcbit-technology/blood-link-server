package rest

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	controller "github.com/mdhasib01/go-rest-starter/controller"
	"github.com/mdhasib01/go-rest-starter/model"
)

// GetDonors godoc
// @Summary list donors
// @Description public donor directory with filters and pagination. Personal details are hidden for donors who opted out.
// @Tags donors
// @Produce  json
// @Param blood_group query string false "Blood group, e.g. A+ (URL-encoded as A%2B)"
// @Param division query string false "Division"
// @Param district query string false "District"
// @Param gender query string false "male | female | others"
// @Param search query string false "Search by name, district, city, profession"
// @Param page query int false "Page number" default(1)
// @Param size query int false "Page size (max 100)" default(12)
// @Success 200 {object} model.PaginatedResponse
// @Failure 400 {string} string
// @Failure 500 {string} string
// @Router /doners [get]
func GetDonors(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	f := model.DonorFilter{
		BloodGroup: normalizeBloodGroup(q.Get("blood_group")),
		Division:   strings.TrimSpace(q.Get("division")),
		District:   strings.TrimSpace(q.Get("district")),
		Upazila:    strings.TrimSpace(q.Get("upazila")),
		Gender:     strings.TrimSpace(q.Get("gender")),
		Search:     q.Get("search"),
	}
	f.Page, _ = strconv.Atoi(q.Get("page"))
	f.Size, _ = strconv.Atoi(q.Get("size"))

	if f.BloodGroup != "" && !model.IsValidBloodGroup(f.BloodGroup) {
		JSON(w, nil, invalidDataError)
		return
	}

	set, err := controller.GetDonors(f)
	JSON(w, set, err)
}

// GetDonor godoc
// @Summary get donor
// @Description public profile of a single donor
// @Tags donors
// @Produce  json
// @Param id path int true "donor id"
// @Success 200 {object} model.Donor
// @Failure 404 {string} string
// @Failure 500 {string} string
// @Router /doner/{id} [get]
func GetDonor(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		JSON(w, nil, invalidDataError)
		return
	}

	donor, err := controller.GetDonor(id)
	JSON(w, donor, err)
}

// normalizeBloodGroup accepts "a+", "A+" and the unencoded "A " form
// ("+" turns into a space when it is not URL-encoded).
func normalizeBloodGroup(bg string) string {
	bg = strings.ToUpper(strings.TrimLeft(bg, " "))
	bg = strings.ReplaceAll(bg, " ", "+")
	return strings.TrimSpace(bg)
}
