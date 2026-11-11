package rest

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	controller "github.com/mdhasib01/go-rest-starter/controller"
	"github.com/mdhasib01/go-rest-starter/model"
)

// any logged-in user may post a blood request or express interest
const anyAuthenticatedUser = 0

// GetPosts godoc
// @Summary list blood request posts
// @Description public list of blood request posts, open ones first
// @Tags posts
// @Produce  json
// @Param status query string false "open | fulfilled | closed"
// @Param blood_group query string false "Blood group, e.g. A+ (URL-encoded as A%2B)"
// @Param division query string false "Division"
// @Param district query string false "District"
// @Param search query string false "Search title, description, hospital, district"
// @Param page query int false "Page number" default(1)
// @Param size query int false "Page size (max 100)" default(30)
// @Success 200 {object} model.PaginatedResponse
// @Failure 400 {string} string
// @Router /posts [get]
func GetPosts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := model.PostFilter{
		Status:     strings.ToLower(strings.TrimSpace(q.Get("status"))),
		BloodGroup: normalizeBloodGroup(q.Get("blood_group")),
		Division:   strings.TrimSpace(q.Get("division")),
		District:   strings.TrimSpace(q.Get("district")),
		Search:     q.Get("search"),
	}
	f.Page, _ = strconv.Atoi(q.Get("page"))
	f.Size, _ = strconv.Atoi(q.Get("size"))

	if f.BloodGroup != "" && !model.IsValidBloodGroup(f.BloodGroup) {
		JSON(w, nil, invalidDataError)
		return
	}
	if f.Status != "" && !contains(model.PostStatuses, f.Status) {
		JSON(w, nil, invalidDataError)
		return
	}

	set, err := controller.GetPosts(f)
	JSON(w, set, err)
}

// GetPost godoc
// @Summary get a blood request post
// @Tags posts
// @Produce  json
// @Param id path int true "post id"
// @Success 200 {object} model.DonationPost
// @Failure 404 {string} string
// @Router /post/{id} [get]
func GetPost(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		JSON(w, nil, invalidDataError)
		return
	}
	post, err := controller.GetPost(id)
	JSON(w, post, err)
}

// CreatePost godoc
// @Summary create a blood request post
// @Tags posts
// @Accept  json
// @Produce  json
// @Security Bearer
// @Param Authorization header string true "Bearer Token" default("Bearer YOUR_ACCESS_TOKEN")
// @Param post body model.DonationPost true "title, blood_group, description, hospital_name, address, division, district, required_date (YYYY-MM-DD), units_needed, contact_phone"
// @Success 200 {object} model.DonationPost
// @Failure 400 {string} string
// @Failure 403 {string} string
// @Router /post [post]
func CreatePost(w http.ResponseWriter, r *http.Request) {
	session, err := isAuthorized(r, anyAuthenticatedUser)
	if err != nil {
		JSON(w, nil, err)
		return
	}

	var post model.DonationPost
	if err := readRequestBody(r, &post); err != nil {
		JSON(w, nil, err)
		return
	}

	created, err := controller.CreatePost(post, session)
	JSON(w, created, err)
}

// TogglePostInterest godoc
// @Summary express or withdraw interest in a post
// @Description toggles the logged-in user's interest; returns the updated post
// @Tags posts
// @Produce  json
// @Security Bearer
// @Param Authorization header string true "Bearer Token" default("Bearer YOUR_ACCESS_TOKEN")
// @Param id path int true "post id"
// @Success 200 {object} model.DonationPost
// @Failure 400 {string} string
// @Failure 404 {string} string
// @Failure 409 {string} string
// @Router /post/{id}/interest [put]
func TogglePostInterest(w http.ResponseWriter, r *http.Request) {
	session, err := isAuthorized(r, anyAuthenticatedUser)
	if err != nil {
		JSON(w, nil, err)
		return
	}

	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		JSON(w, nil, invalidDataError)
		return
	}

	post, err := controller.ToggleInterest(id, session)
	JSON(w, post, err)
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}
