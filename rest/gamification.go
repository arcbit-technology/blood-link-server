package rest

import (
	"net/http"
	"strconv"

	controller "github.com/mdhasib01/go-rest-starter/controller"
	"github.com/mdhasib01/go-rest-starter/model"
)

// GetPostInterests godoc
// @Summary donors interested in a post (post author only)
// @Tags posts
// @Produce  json
// @Security Bearer
// @Param Authorization header string true "Bearer Token" default("Bearer YOUR_ACCESS_TOKEN")
// @Param id path int true "post id"
// @Success 200 {array} model.InterestedDonor
// @Failure 403 {string} string
// @Router /post/{id}/interests [get]
func GetPostInterests(w http.ResponseWriter, r *http.Request) {
	session, err := isAuthorized(r, anyAuthenticatedUser)
	if err != nil {
		JSON(w, nil, err)
		return
	}
	id, err := pathInt(r, "id")
	if err != nil {
		JSON(w, nil, err)
		return
	}
	list, err := controller.GetInterestedDonors(id, session)
	JSON(w, list, err)
}

// ConfirmDonation godoc
// @Summary confirm that an interested donor donated (post author only)
// @Description awards the donor points and badges; marks the post fulfilled once enough units are confirmed
// @Tags posts
// @Accept  json
// @Produce  json
// @Security Bearer
// @Param Authorization header string true "Bearer Token" default("Bearer YOUR_ACCESS_TOKEN")
// @Param id path int true "post id"
// @Param body body model.ConfirmDonation true "donor_id, units, donation_date (YYYY-MM-DD)"
// @Success 200 {object} model.DonationConfirmation
// @Failure 400 {string} string
// @Failure 403 {string} string
// @Failure 409 {string} string
// @Router /post/{id}/confirm-donation [post]
func ConfirmDonation(w http.ResponseWriter, r *http.Request) {
	session, err := isAuthorized(r, anyAuthenticatedUser)
	if err != nil {
		JSON(w, nil, err)
		return
	}
	id, err := pathInt(r, "id")
	if err != nil {
		JSON(w, nil, err)
		return
	}
	var req model.ConfirmDonation
	if err := readRequestBody(r, &req); err != nil {
		JSON(w, nil, err)
		return
	}
	res, err := controller.ConfirmDonation(id, req, session)
	JSON(w, res, err)
}

// GetGamificationProfile godoc
// @Summary a user's points, level, rank and badges
// @Tags gamification
// @Produce  json
// @Param id path int true "user id"
// @Success 200 {object} model.GamificationProfile
// @Router /user/{id}/gamification [get]
func GetGamificationProfile(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		JSON(w, nil, err)
		return
	}
	p, err := controller.GetGamificationProfile(id)
	JSON(w, p, err)
}

// GetBadges godoc
// @Summary all badges that can be earned
// @Tags gamification
// @Produce  json
// @Success 200 {array} model.Badge
// @Router /badges [get]
func GetBadges(w http.ResponseWriter, r *http.Request) {
	list, err := controller.GetBadges()
	JSON(w, list, err)
}

// GetLeaderboard godoc
// @Summary top donors by points
// @Tags gamification
// @Produce  json
// @Param scope query string false "global (default) | division | district | upazila | bank"
// @Param value query string false "division / district / upazila name"
// @Param bank_id query int false "blood bank id (scope=bank)"
// @Param period query string false "all (default) | month"
// @Param size query int false "max 100" default(20)
// @Success 200 {array} model.LeaderboardEntry
// @Router /leaderboard [get]
func GetLeaderboard(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := model.LeaderboardFilter{Scope: q.Get("scope"), Value: q.Get("value"), Period: q.Get("period")}
	f.BankId, _ = strconv.Atoi(q.Get("bank_id"))
	f.Size, _ = strconv.Atoi(q.Get("size"))
	list, err := controller.GetLeaderboard(f)
	JSON(w, list, err)
}

// GetBankLeaderboard godoc
// @Summary top donors of a blood bank
// @Tags gamification
// @Produce  json
// @Param id path int true "blood bank id"
// @Param period query string false "all (default) | month"
// @Success 200 {array} model.LeaderboardEntry
// @Router /blood-bank/{id}/leaderboard [get]
func GetBankLeaderboard(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		JSON(w, nil, err)
		return
	}
	q := r.URL.Query()
	f := model.LeaderboardFilter{Scope: "bank", BankId: id, Period: q.Get("period")}
	f.Size, _ = strconv.Atoi(q.Get("size"))
	list, err := controller.GetLeaderboard(f)
	JSON(w, list, err)
}
