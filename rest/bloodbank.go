package rest

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	controller "github.com/mdhasib01/go-rest-starter/controller"
	"github.com/mdhasib01/go-rest-starter/model"
)

// optionalSession returns the caller's session when a valid token is
// sent, nil otherwise (public endpoints that show more to members).
func optionalSession(r *http.Request) *model.Session {
	if strings.TrimSpace(r.Header.Get("Authorization")) == "" {
		return nil
	}
	s, err := isAuthorized(r, anyAuthenticatedUser)
	if err != nil {
		return nil
	}
	return &s
}

func pathInt(r *http.Request, name string) (int, error) {
	id, err := strconv.Atoi(mux.Vars(r)[name])
	if err != nil || id < 1 {
		return 0, invalidDataError
	}
	return id, nil
}

// CreateBloodBank godoc
// @Summary request a new blood bank
// @Description any donor can request a blood bank for an upazila; a platform admin approves it and the creator becomes its admin
// @Tags blood banks
// @Accept  json
// @Produce  json
// @Security Bearer
// @Param Authorization header string true "Bearer Token" default("Bearer YOUR_ACCESS_TOKEN")
// @Param bank body model.BloodBank true "name, description, division, district, upazila, address, contact_phone"
// @Success 200 {object} model.BloodBank
// @Failure 400 {string} string
// @Failure 409 {string} string
// @Router /blood-bank [post]
func CreateBloodBank(w http.ResponseWriter, r *http.Request) {
	session, err := isAuthorized(r, anyAuthenticatedUser)
	if err != nil {
		JSON(w, nil, err)
		return
	}
	var b model.BloodBank
	if err := readRequestBody(r, &b); err != nil {
		JSON(w, nil, err)
		return
	}
	created, err := controller.CreateBloodBank(b, session)
	JSON(w, created, err)
}

// GetBloodBanks godoc
// @Summary list blood banks
// @Description approved blood banks; platform admins may filter by status (pending | approved | rejected | suspended | all)
// @Tags blood banks
// @Produce  json
// @Param division query string false "Division"
// @Param district query string false "District"
// @Param upazila query string false "Upazila / thana"
// @Param search query string false "Search name, description, location"
// @Param status query string false "admin only"
// @Param page query int false "Page number" default(1)
// @Param size query int false "Page size (max 100)" default(30)
// @Success 200 {object} model.PaginatedResponse
// @Router /blood-banks [get]
func GetBloodBanks(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := model.BloodBankFilter{
		Division: strings.TrimSpace(q.Get("division")),
		District: strings.TrimSpace(q.Get("district")),
		Upazila:  strings.TrimSpace(q.Get("upazila")),
		Search:   q.Get("search"),
		Status:   strings.ToLower(strings.TrimSpace(q.Get("status"))),
	}
	f.Page, _ = strconv.Atoi(q.Get("page"))
	f.Size, _ = strconv.Atoi(q.Get("size"))
	if f.Status != "" && f.Status != "all" && !contains(model.BankStatuses, f.Status) {
		JSON(w, nil, invalidDataError)
		return
	}
	set, err := controller.GetBloodBanks(f, optionalSession(r))
	JSON(w, set, err)
}

// GetBloodBank godoc
// @Summary get a blood bank
// @Tags blood banks
// @Produce  json
// @Param id path int true "blood bank id"
// @Success 200 {object} model.BloodBank
// @Failure 404 {string} string
// @Router /blood-bank/{id} [get]
func GetBloodBank(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		JSON(w, nil, err)
		return
	}
	b, err := controller.GetBloodBank(id, optionalSession(r))
	JSON(w, b, err)
}

// GetMyBloodBank godoc
// @Summary the caller's blood bank membership and pending creation request
// @Tags blood banks
// @Produce  json
// @Security Bearer
// @Param Authorization header string true "Bearer Token" default("Bearer YOUR_ACCESS_TOKEN")
// @Success 200 {object} model.MyBloodBank
// @Router /blood-bank/mine [get]
func GetMyBloodBank(w http.ResponseWriter, r *http.Request) {
	session, err := isAuthorized(r, anyAuthenticatedUser)
	if err != nil {
		JSON(w, nil, err)
		return
	}
	res, err := controller.GetMyBloodBank(session)
	JSON(w, res, err)
}

// UpdateBloodBank godoc
// @Summary edit a blood bank (bank admin)
// @Tags blood banks
// @Accept  json
// @Produce  json
// @Security Bearer
// @Param Authorization header string true "Bearer Token" default("Bearer YOUR_ACCESS_TOKEN")
// @Param id path int true "blood bank id"
// @Param bank body model.BloodBank true "name, description, address, contact_phone, logo"
// @Success 200 {object} model.BloodBank
// @Router /blood-bank/{id} [put]
func UpdateBloodBank(w http.ResponseWriter, r *http.Request) {
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
	var b model.BloodBank
	if err := readRequestBody(r, &b); err != nil {
		JSON(w, nil, err)
		return
	}
	updated, err := controller.UpdateBloodBank(id, b, session)
	JSON(w, updated, err)
}

// ReviewBloodBank godoc
// @Summary approve, reject or suspend a blood bank (platform admin)
// @Description approve: pending or suspended -> approved; reject: pending -> rejected; suspend: approved -> suspended. reject and suspend need a reason.
// @Tags blood banks
// @Accept  json
// @Produce  json
// @Security Bearer
// @Param Authorization header string true "Bearer Token" default("Bearer YOUR_ACCESS_TOKEN")
// @Param id path int true "blood bank id"
// @Param action path string true "approve | reject | suspend"
// @Param review body model.BankReview false "reason"
// @Success 200 {object} model.BloodBank
// @Failure 403 {string} string
// @Failure 409 {string} string
// @Router /blood-bank/{id}/{action} [put]
func ReviewBloodBank(w http.ResponseWriter, r *http.Request) {
	session, err := isAuthorized(r, ManageBloodBanksPermission)
	if err != nil {
		JSON(w, nil, err)
		return
	}
	id, err := pathInt(r, "id")
	if err != nil {
		JSON(w, nil, err)
		return
	}
	var review model.BankReview
	if r.ContentLength != 0 {
		if err := readRequestBody(r, &review); err != nil {
			JSON(w, nil, err)
			return
		}
	}

	var b model.BloodBank
	switch mux.Vars(r)["action"] {
	case "approve":
		b, err = controller.ApproveBloodBank(id, session)
	case "reject":
		b, err = controller.RejectBloodBank(id, review.Reason, session)
	case "suspend":
		b, err = controller.SuspendBloodBank(id, review.Reason, session)
	default:
		err = invalidDataError
	}
	JSON(w, b, err)
}

// JoinBloodBank godoc
// @Summary ask to join a blood bank
// @Tags blood banks
// @Produce  json
// @Security Bearer
// @Param Authorization header string true "Bearer Token" default("Bearer YOUR_ACCESS_TOKEN")
// @Param id path int true "blood bank id"
// @Success 200 {object} model.MyBloodBank
// @Failure 409 {string} string
// @Router /blood-bank/{id}/join [post]
func JoinBloodBank(w http.ResponseWriter, r *http.Request) {
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
	res, err := controller.JoinBloodBank(id, session)
	JSON(w, res, err)
}

// LeaveBloodBank godoc
// @Summary leave a blood bank or cancel a join request
// @Description the bank admin must transfer admin first
// @Tags blood banks
// @Produce  json
// @Security Bearer
// @Param Authorization header string true "Bearer Token" default("Bearer YOUR_ACCESS_TOKEN")
// @Param id path int true "blood bank id"
// @Success 200 {string} string
// @Router /blood-bank/{id}/membership [delete]
func LeaveBloodBank(w http.ResponseWriter, r *http.Request) {
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
	err = controller.LeaveBloodBank(id, session)
	JSON(w, "ok", err)
}

// GetBankMembers godoc
// @Summary list blood bank members
// @Description active members are public; pending requests are visible to the bank admin and officers
// @Tags blood banks
// @Produce  json
// @Param id path int true "blood bank id"
// @Param status query string false "active (default) | pending"
// @Success 200 {array} model.BankMember
// @Router /blood-bank/{id}/members [get]
func GetBankMembers(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		JSON(w, nil, err)
		return
	}
	status := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("status")))
	members, err := controller.GetBankMembers(id, status, optionalSession(r))
	JSON(w, members, err)
}

// ReviewBankMember godoc
// @Summary approve or reject a join request (bank admin / officer)
// @Tags blood banks
// @Produce  json
// @Security Bearer
// @Param Authorization header string true "Bearer Token" default("Bearer YOUR_ACCESS_TOKEN")
// @Param id path int true "blood bank id"
// @Param userId path int true "user id"
// @Param action path string true "approve | reject"
// @Success 200 {string} string
// @Router /blood-bank/{id}/member/{userId}/{action} [put]
func ReviewBankMember(w http.ResponseWriter, r *http.Request) {
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
	userID, err := pathInt(r, "userId")
	if err != nil {
		JSON(w, nil, err)
		return
	}
	switch mux.Vars(r)["action"] {
	case "approve":
		err = controller.ReviewMember(id, userID, true, session)
	case "reject":
		err = controller.ReviewMember(id, userID, false, session)
	default:
		err = invalidDataError
	}
	JSON(w, "ok", err)
}

// RemoveBankMember godoc
// @Summary remove a member (bank admin / officer)
// @Tags blood banks
// @Produce  json
// @Security Bearer
// @Param Authorization header string true "Bearer Token" default("Bearer YOUR_ACCESS_TOKEN")
// @Param id path int true "blood bank id"
// @Param userId path int true "user id"
// @Success 200 {string} string
// @Router /blood-bank/{id}/member/{userId} [delete]
func RemoveBankMember(w http.ResponseWriter, r *http.Request) {
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
	userID, err := pathInt(r, "userId")
	if err != nil {
		JSON(w, nil, err)
		return
	}
	err = controller.RemoveMember(id, userID, session)
	JSON(w, "ok", err)
}

// SetBankMemberRole godoc
// @Summary set a member's role and position (bank admin)
// @Tags blood banks
// @Accept  json
// @Produce  json
// @Security Bearer
// @Param Authorization header string true "Bearer Token" default("Bearer YOUR_ACCESS_TOKEN")
// @Param id path int true "blood bank id"
// @Param userId path int true "user id"
// @Param role body model.BankMemberRoleUpdate true "role (officer | member), position"
// @Success 200 {array} model.BankMember
// @Router /blood-bank/{id}/member/{userId}/role [put]
func SetBankMemberRole(w http.ResponseWriter, r *http.Request) {
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
	userID, err := pathInt(r, "userId")
	if err != nil {
		JSON(w, nil, err)
		return
	}
	var req model.BankMemberRoleUpdate
	if err := readRequestBody(r, &req); err != nil {
		JSON(w, nil, err)
		return
	}
	members, err := controller.SetMemberRole(id, userID, req, session)
	JSON(w, members, err)
}

// TransferBankAdmin godoc
// @Summary hand the bank admin role to another active member
// @Tags blood banks
// @Accept  json
// @Produce  json
// @Security Bearer
// @Param Authorization header string true "Bearer Token" default("Bearer YOUR_ACCESS_TOKEN")
// @Param id path int true "blood bank id"
// @Param body body model.TransferAdmin true "user_id"
// @Success 200 {object} model.BloodBank
// @Router /blood-bank/{id}/transfer-admin [put]
func TransferBankAdmin(w http.ResponseWriter, r *http.Request) {
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
	var req model.TransferAdmin
	if err := readRequestBody(r, &req); err != nil {
		JSON(w, nil, err)
		return
	}
	b, err := controller.TransferBankAdmin(id, req.UserId, session)
	JSON(w, b, err)
}

// GetBankConversation godoc
// @Summary the blood bank's group chat (members only)
// @Tags blood banks
// @Produce  json
// @Security Bearer
// @Param Authorization header string true "Bearer Token" default("Bearer YOUR_ACCESS_TOKEN")
// @Param id path int true "blood bank id"
// @Success 200 {object} model.Conversation
// @Router /blood-bank/{id}/conversation [get]
func GetBankConversation(w http.ResponseWriter, r *http.Request) {
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
	convID, err := controller.GetBankConversationID(id, session)
	if err != nil {
		JSON(w, nil, err)
		return
	}
	c, err := controller.GetConversation(session, convID)
	JSON(w, c, err)
}
