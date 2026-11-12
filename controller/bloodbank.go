package controller

import (
	"fmt"
	"strings"
	"unicode/utf8"

	dao "github.com/mdhasib01/go-rest-starter/dao"
	itn "github.com/mdhasib01/go-rest-starter/itn"
	model "github.com/mdhasib01/go-rest-starter/model"
)

const (
	defaultBankPageSize = 30
	maxBankPageSize     = 100
	maxBankDescription  = 2000
	maxPositionLength   = 60
)

// ---------------------------------------------------------------------
// Authorization helpers
// ---------------------------------------------------------------------

// isPlatformAdmin reports whether the session may manage all blood banks.
func isPlatformAdmin(session model.Session) bool {
	if session.RoleId == model.SUPERADMIN {
		return true
	}
	ok, _ := hasPermission(session.RoleId, ManageBloodBanksPermission)
	return ok
}

// bankRole returns the session user's role in an active membership of the
// bank ("" when not a member).
func bankRole(bankID, userID int) (string, error) {
	return dao.BankRole(bankID, userID)
}

// requireBankRole succeeds when the user's bank role is one of roles, or
// when the user is a platform admin.
func requireBankRole(session model.Session, bankID int, roles ...string) (string, error) {
	role, err := bankRole(bankID, session.UserId)
	if err != nil {
		return "", err
	}
	for _, r := range roles {
		if role == r {
			return role, nil
		}
	}
	if isPlatformAdmin(session) {
		return role, nil
	}
	return role, model.NewError(itn.ErrorBankForbidden, 403)
}

// ManageBloodBanksPermission mirrors rest.ManageBloodBanksPermission (the
// permission table id seeded by migration 000003).
const ManageBloodBanksPermission = 10

// ---------------------------------------------------------------------
// Banks
// ---------------------------------------------------------------------

func validateBloodBank(b *model.BloodBank, withLocation bool) error {
	b.Name = strings.TrimSpace(b.Name)
	b.Description = strings.TrimSpace(b.Description)
	b.Address = strings.TrimSpace(b.Address)
	b.ContactPhone = strings.TrimSpace(b.ContactPhone)
	b.Logo = strings.TrimSpace(b.Logo)

	if n := utf8.RuneCountInString(b.Name); n < 3 || n > 120 {
		return model.NewError(itn.ErrorBankNameRequired, 400)
	}
	if utf8.RuneCountInString(b.Description) > maxBankDescription ||
		utf8.RuneCountInString(b.ContactPhone) > 50 || utf8.RuneCountInString(b.Logo) > 255 {
		return model.NewError(itn.ErrorInvalidData, 400)
	}
	if !withLocation {
		return nil
	}

	b.Division = strings.TrimSpace(b.Division)
	b.District = strings.TrimSpace(b.District)
	b.Upazila = strings.TrimSpace(b.Upazila)
	if b.Division == "" || b.District == "" || b.Upazila == "" {
		return model.NewError(itn.ErrorBankLocationRequired, 400)
	}
	if !model.IsDistrictInDivision(b.Division, b.District) {
		return model.NewError(itn.ErrorBankLocationRequired, 400)
	}
	if !model.IsValidUpazila(b.District, b.Upazila) {
		return model.NewError(itn.ErrorUpazilaInvalid, 400)
	}
	return nil
}

// CreateBloodBank submits a creation request. A donor who already belongs
// to (or asked to join) a bank, or who has a pending request, cannot.
func CreateBloodBank(b model.BloodBank, session model.Session) (model.BloodBank, error) {
	if err := validateBloodBank(&b, true); err != nil {
		return model.BloodBank{}, err
	}
	membership, err := dao.GetMembership(session.UserId)
	if err != nil {
		return model.BloodBank{}, err
	}
	if membership != nil {
		return model.BloodBank{}, model.NewError(itn.ErrorBankAlreadyMember, 409)
	}

	b.CreatedBy = session.UserId
	id, err := dao.CreateBloodBank(b)
	if err != nil {
		return model.BloodBank{}, err
	}
	return dao.GetBloodBank(id)
}

// GetBloodBanks lists approved banks. Platform admins can list any status.
func GetBloodBanks(f model.BloodBankFilter, session *model.Session) (model.PaginatedResponse, error) {
	if f.Status == "" || session == nil || !isPlatformAdmin(*session) {
		f.Status = model.BankStatusApproved
	}
	if f.Status == "all" {
		f.Status = ""
	}
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Size < 1 {
		f.Size = defaultBankPageSize
	}
	if f.Size > maxBankPageSize {
		f.Size = maxBankPageSize
	}
	return dao.GetBloodBanks(f)
}

// GetBloodBank returns an approved bank to anyone; other statuses only to
// the creator and platform admins.
func GetBloodBank(id int, session *model.Session) (model.BloodBank, error) {
	b, err := dao.GetBloodBank(id)
	if err != nil {
		return b, err
	}
	if b.Status == model.BankStatusApproved {
		return b, nil
	}
	if session != nil && (session.UserId == b.CreatedBy || session.UserId == b.AdminId || isPlatformAdmin(*session)) {
		return b, nil
	}
	return model.BloodBank{}, model.NewError(itn.ErrorBankNotFound, 404)
}

// GetMyBloodBank returns the caller's membership and/or pending request.
func GetMyBloodBank(session model.Session) (model.MyBloodBank, error) {
	var res model.MyBloodBank
	m, err := dao.GetMembership(session.UserId)
	if err != nil {
		return res, err
	}
	if m != nil {
		res.Membership = m
		b, err := dao.GetBloodBank(m.BloodBankId)
		if err != nil {
			return res, err
		}
		res.Bank = &b
	}
	res.PendingRequest, err = dao.GetPendingBankRequest(session.UserId)
	return res, err
}

// UpdateBloodBank lets the bank admin edit its description and contact.
func UpdateBloodBank(id int, b model.BloodBank, session model.Session) (model.BloodBank, error) {
	if _, err := requireBankRole(session, id, model.BankRoleAdmin); err != nil {
		return model.BloodBank{}, err
	}
	if err := validateBloodBank(&b, false); err != nil {
		return model.BloodBank{}, err
	}
	if err := dao.UpdateBloodBank(id, b); err != nil {
		return model.BloodBank{}, err
	}
	return dao.GetBloodBank(id)
}

// ApproveBloodBank (platform admin) approves a pending or re-activates a
// suspended bank.
func ApproveBloodBank(id int, session model.Session) (model.BloodBank, error) {
	b, founder, err := dao.ApproveBloodBank(id, session.UserId)
	if err != nil {
		return b, err
	}
	notify([]int{b.AdminId}, model.Notification{
		Kind:  model.NotifyBankApproved,
		Title: "Blood bank approved",
		Body:  fmt.Sprintf("%s is now live. You are its admin.", b.Name),
		Data:  map[string]any{"blood_bank_id": b.Id},
	})
	if founder != nil {
		notifyBadge(b.AdminId, *founder)
	}
	return b, nil
}

// RejectBloodBank (platform admin) rejects a pending request.
func RejectBloodBank(id int, reason string, session model.Session) (model.BloodBank, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" || utf8.RuneCountInString(reason) > 500 {
		return model.BloodBank{}, model.NewError(itn.ErrorBankReasonRequired, 400)
	}
	b, err := dao.SetBloodBankStatus(id, session.UserId, []string{model.BankStatusPending}, model.BankStatusRejected, reason)
	if err != nil {
		return b, err
	}
	notify([]int{b.CreatedBy}, model.Notification{
		Kind:  model.NotifyBankRejected,
		Title: "Blood bank request rejected",
		Body:  fmt.Sprintf("%s was not approved: %s", b.Name, reason),
		Data:  map[string]any{"blood_bank_id": b.Id},
	})
	return b, nil
}

// SuspendBloodBank (platform admin) suspends an approved bank.
func SuspendBloodBank(id int, reason string, session model.Session) (model.BloodBank, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" || utf8.RuneCountInString(reason) > 500 {
		return model.BloodBank{}, model.NewError(itn.ErrorBankReasonRequired, 400)
	}
	b, err := dao.SetBloodBankStatus(id, session.UserId, []string{model.BankStatusApproved}, model.BankStatusSuspended, reason)
	if err != nil {
		return b, err
	}
	officers, _ := dao.BankOfficerIDs(id)
	notify(officers, model.Notification{
		Kind:  model.NotifyBankSuspended,
		Title: "Blood bank suspended",
		Body:  fmt.Sprintf("%s was suspended: %s", b.Name, reason),
		Data:  map[string]any{"blood_bank_id": b.Id},
	})
	return b, nil
}

// ---------------------------------------------------------------------
// Membership
// ---------------------------------------------------------------------

// JoinBloodBank asks to join an approved bank.
func JoinBloodBank(id int, session model.Session) (model.MyBloodBank, error) {
	b, err := dao.GetBloodBank(id)
	if err != nil {
		return model.MyBloodBank{}, err
	}
	if b.Status != model.BankStatusApproved {
		return model.MyBloodBank{}, model.NewError(itn.ErrorBankNotApproved, 409)
	}
	pending, err := dao.GetPendingBankRequest(session.UserId)
	if err != nil {
		return model.MyBloodBank{}, err
	}
	if pending != nil {
		return model.MyBloodBank{}, model.NewError(itn.ErrorBankPendingRequestExists, 409)
	}
	if err := dao.JoinBloodBank(id, session.UserId); err != nil {
		return model.MyBloodBank{}, err
	}

	officers, _ := dao.BankOfficerIDs(id)
	notify(officers, model.Notification{
		Kind:  model.NotifyJoinRequested,
		Title: "New join request",
		Body:  fmt.Sprintf("A donor asked to join %s.", b.Name),
		Data:  map[string]any{"blood_bank_id": b.Id, "user_id": session.UserId},
	})
	return GetMyBloodBank(session)
}

// LeaveBloodBank leaves the bank (or cancels a pending join request).
func LeaveBloodBank(id int, session model.Session) error {
	return dao.EndMembership(id, session.UserId, model.MemberStatusLeft)
}

// GetBankMembers lists members by status. Anyone can see active members of
// an approved bank (phones only for fellow members); pending requests are
// visible to the bank's admin and officers.
func GetBankMembers(id int, status string, session *model.Session) ([]model.BankMember, error) {
	if status == "" {
		status = model.MemberStatusActive
	}
	if !contains(model.MemberStatuses, status) {
		return nil, model.NewError(itn.ErrorInvalidData, 400)
	}
	if _, err := GetBloodBank(id, session); err != nil {
		return nil, err
	}

	viewerRole := ""
	if session != nil {
		var err error
		if viewerRole, err = bankRole(id, session.UserId); err != nil {
			return nil, err
		}
	}
	isManager := viewerRole == model.BankRoleAdmin || viewerRole == model.BankRoleOfficer ||
		(session != nil && isPlatformAdmin(*session))
	if status != model.MemberStatusActive && !isManager {
		return nil, model.NewError(itn.ErrorBankForbidden, 403)
	}

	members, err := dao.GetBankMembers(id, status)
	if err != nil {
		return nil, err
	}
	isMember := viewerRole != ""
	for i := range members {
		m := &members[i]
		if !isMember && !isManager {
			m.Phone = ""
			if m.HidePersonalInfo {
				m.FirstName, m.LastName, m.Photo = "", "", ""
			}
		}
	}
	return members, nil
}

// ReviewMember approves or rejects a pending join request (admin/officer).
func ReviewMember(bankID, userID int, approve bool, session model.Session) error {
	if _, err := requireBankRole(session, bankID, model.BankRoleAdmin, model.BankRoleOfficer); err != nil {
		return err
	}
	if err := dao.ReviewMember(bankID, userID, session.UserId, approve); err != nil {
		return err
	}

	b, _ := dao.GetBloodBank(bankID)
	n := model.Notification{
		Kind:  model.NotifyJoinApproved,
		Title: "Welcome aboard!",
		Body:  fmt.Sprintf("Your request to join %s was approved.", b.Name),
		Data:  map[string]any{"blood_bank_id": bankID},
	}
	if !approve {
		n.Kind = model.NotifyJoinRejected
		n.Title = "Join request declined"
		n.Body = fmt.Sprintf("Your request to join %s was declined.", b.Name)
	}
	notify([]int{userID}, n)
	return nil
}

// RemoveMember removes a member (admin/officer). Officers cannot remove
// other officers; nobody can remove the admin.
func RemoveMember(bankID, userID int, session model.Session) error {
	role, err := requireBankRole(session, bankID, model.BankRoleAdmin, model.BankRoleOfficer)
	if err != nil {
		return err
	}
	if userID == session.UserId {
		return dao.EndMembership(bankID, userID, model.MemberStatusLeft)
	}
	targetRole, err := bankRole(bankID, userID)
	if err != nil {
		return err
	}
	if role == model.BankRoleOfficer && targetRole == model.BankRoleOfficer && !isPlatformAdmin(session) {
		return model.NewError(itn.ErrorBankForbidden, 403)
	}
	if err := dao.EndMembership(bankID, userID, model.MemberStatusRemoved); err != nil {
		return err
	}
	b, _ := dao.GetBloodBank(bankID)
	notify([]int{userID}, model.Notification{
		Kind:  model.NotifyMemberRemoved,
		Title: "Removed from blood bank",
		Body:  fmt.Sprintf("You are no longer a member of %s.", b.Name),
		Data:  map[string]any{"blood_bank_id": bankID},
	})
	return nil
}

// SetMemberRole (bank admin) sets a member's role (officer/member) and
// position title. For the admin themself only the position changes.
func SetMemberRole(bankID, userID int, req model.BankMemberRoleUpdate, session model.Session) ([]model.BankMember, error) {
	if _, err := requireBankRole(session, bankID, model.BankRoleAdmin); err != nil {
		return nil, err
	}
	req.Role = strings.ToLower(strings.TrimSpace(req.Role))
	req.Position = strings.TrimSpace(req.Position)
	if utf8.RuneCountInString(req.Position) > maxPositionLength {
		return nil, model.NewError(itn.ErrorBankPositionTooLong, 400)
	}

	targetRole, err := bankRole(bankID, userID)
	if err != nil {
		return nil, err
	}
	if targetRole == "" {
		return nil, model.NewError(itn.ErrorBankMemberNotFound, 404)
	}

	if targetRole == model.BankRoleAdmin {
		if req.Role != "" && req.Role != model.BankRoleAdmin {
			// demoting the admin is done by transferring admin
			return nil, model.NewError(itn.ErrorBankInvalidRole, 400)
		}
		if err := dao.SetAdminPosition(bankID, userID, req.Position); err != nil {
			return nil, err
		}
	} else {
		if req.Role != model.BankRoleOfficer && req.Role != model.BankRoleMember {
			return nil, model.NewError(itn.ErrorBankInvalidRole, 400)
		}
		if err := dao.SetMemberRole(bankID, userID, req.Role, req.Position); err != nil {
			return nil, err
		}
	}

	if userID != session.UserId {
		b, _ := dao.GetBloodBank(bankID)
		title := req.Position
		if title == "" {
			title = req.Role
		}
		notify([]int{userID}, model.Notification{
			Kind:  model.NotifyRoleChanged,
			Title: "New position",
			Body:  fmt.Sprintf("You are now %s in %s.", title, b.Name),
			Data:  map[string]any{"blood_bank_id": bankID},
		})
	}
	return dao.GetBankMembers(bankID, model.MemberStatusActive)
}

// TransferBankAdmin hands the admin role to another active member. Only
// the current admin (or a platform admin) can do this.
func TransferBankAdmin(bankID, newAdminID int, session model.Session) (model.BloodBank, error) {
	if _, err := requireBankRole(session, bankID, model.BankRoleAdmin); err != nil {
		return model.BloodBank{}, err
	}
	oldAdmin, err := dao.TransferBankAdmin(bankID, newAdminID)
	if err != nil {
		return model.BloodBank{}, err
	}
	b, err := dao.GetBloodBank(bankID)
	if err != nil {
		return b, err
	}
	notify([]int{newAdminID, oldAdmin}, model.Notification{
		Kind:  model.NotifyAdminTransferred,
		Title: "Blood bank admin changed",
		Body:  fmt.Sprintf("%s is now the admin of %s.", b.AdminName, b.Name),
		Data:  map[string]any{"blood_bank_id": bankID, "admin_id": newAdminID},
	})
	return b, nil
}

// GetBankConversationID returns the bank chat id for an active member.
func GetBankConversationID(bankID int, session model.Session) (int, error) {
	if _, err := requireBankRole(session, bankID, model.BankRoleAdmin, model.BankRoleOfficer, model.BankRoleMember); err != nil {
		return 0, err
	}
	return dao.GetBankConversationID(bankID)
}
