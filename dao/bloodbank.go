package dao

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	itn "github.com/mdhasib01/go-rest-starter/itn"
	model "github.com/mdhasib01/go-rest-starter/model"
	"github.com/mdhasib01/go-rest-starter/pkg/logger"
)

const bankColumns = `
	b.id, b.name, COALESCE(b.description, ''), b.division, b.district, b.upazila,
	COALESCE(b.address, ''), COALESCE(b.contact_phone, ''), COALESCE(b.logo, ''), b.status,
	b.admin_id, TRIM(COALESCE(a.first_name, '') || ' ' || COALESCE(a.last_name, '')),
	b.created_by, TRIM(COALESCE(c.first_name, '') || ' ' || COALESCE(c.last_name, '')),
	b.reviewed_at, COALESCE(b.rejection_reason, ''),
	(SELECT COUNT(*) FROM blood_bank_members m WHERE m.blood_bank_id = b.id AND m.status = 'active'),
	(SELECT COUNT(*) FROM donation_history h WHERE h.blood_bank_id = b.id),
	(SELECT COALESCE(SUM(mu.points), 0) FROM blood_bank_members m
	   JOIN users mu ON mu.id = m.user_id WHERE m.blood_bank_id = b.id AND m.status = 'active'),
	b.created_at, COALESCE(b.cover, '')`

const bankFrom = ` FROM blood_banks b
	JOIN users a ON a.id = b.admin_id
	JOIN users c ON c.id = b.created_by`

func scanBank(row rowScanner) (model.BloodBank, error) {
	var b model.BloodBank
	var reviewed sql.NullTime
	err := row.Scan(
		&b.Id, &b.Name, &b.Description, &b.Division, &b.District, &b.Upazila,
		&b.Address, &b.ContactPhone, &b.Logo, &b.Status,
		&b.AdminId, &b.AdminName, &b.CreatedBy, &b.CreatedByName,
		&reviewed, &b.RejectionReason, &b.MemberCount, &b.DonationCount, &b.TotalPoints,
		&b.CreatedAt, &b.Cover,
	)
	if reviewed.Valid {
		t := reviewed.Time
		b.ReviewedAt = &t
	}
	return b, err
}

// CreateBloodBank stores a creation request (status pending). The
// creator is recorded as the future admin.
func CreateBloodBank(b model.BloodBank) (int, error) {
	query := `INSERT INTO blood_banks (name, description, division, district, upazila,
			address, contact_phone, status, admin_id, created_by)
		VALUES ($1, NULLIF($2, ''), $3, $4, $5, NULLIF($6, ''),
			COALESCE(NULLIF($7, ''), (SELECT NULLIF(phone, '') FROM users WHERE id = $8)),
			'pending', $8, $8)
		RETURNING id`

	var id int
	err := DB.QueryRow(query, b.Name, b.Description, b.Division, b.District, b.Upazila,
		b.Address, b.ContactPhone, b.CreatedBy).Scan(&id)
	if isUniqueViolation(err, "ux_blood_banks_one_pending_per_creator") {
		return 0, model.NewError(itn.ErrorBankPendingRequestExists, 409)
	}
	if err != nil {
		logger.GetLogger().LogErrors(err, map[string]interface{}{"query": query})
		return 0, model.UnknownError
	}
	return id, nil
}

func GetBloodBanks(f model.BloodBankFilter) (model.PaginatedResponse, error) {
	where := []string{}
	args := []interface{}{}
	arg := func(v interface{}) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}

	if f.Status != "" {
		where = append(where, "b.status = "+arg(f.Status))
	}
	if f.Division != "" {
		where = append(where, "LOWER(b.division) = LOWER("+arg(f.Division)+")")
	}
	if f.District != "" {
		where = append(where, "LOWER(b.district) = LOWER("+arg(f.District)+")")
	}
	if f.Upazila != "" {
		where = append(where, "LOWER(b.upazila) = LOWER("+arg(f.Upazila)+")")
	}
	if s := strings.TrimSpace(f.Search); s != "" {
		k := arg("%" + escapeLike(s) + "%")
		where = append(where, fmt.Sprintf(
			"(b.name ILIKE %[1]s OR b.description ILIKE %[1]s OR b.upazila ILIKE %[1]s OR b.district ILIKE %[1]s)", k))
	}

	whereSQL := ""
	if len(where) > 0 {
		whereSQL = " WHERE " + strings.Join(where, " AND ")
	}

	var count int
	if err := DB.QueryRow("SELECT COUNT(*)"+bankFrom+whereSQL, args...).Scan(&count); err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return model.PaginatedResponse{}, model.UnknownError
	}

	// pending requests: oldest first (review queue); otherwise biggest first
	order := " ORDER BY 17 DESC, b.created_at DESC, b.id DESC"
	if f.Status == model.BankStatusPending {
		order = " ORDER BY b.created_at ASC, b.id ASC"
	}
	query := "SELECT " + bankColumns + bankFrom + whereSQL + order +
		" LIMIT " + arg(f.Size) + " OFFSET " + arg((f.Page-1)*f.Size)

	rows, err := DB.Query(query, args...)
	if err != nil {
		logger.GetLogger().LogErrors(err, map[string]interface{}{"query": query})
		return model.PaginatedResponse{}, model.UnknownError
	}
	defer rows.Close()

	list := []model.BloodBank{}
	for rows.Next() {
		b, err := scanBank(rows)
		if err != nil {
			logger.GetLogger().LogErrors(err, nil)
			return model.PaginatedResponse{}, model.UnknownError
		}
		list = append(list, b)
	}
	if err := rows.Err(); err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return model.PaginatedResponse{}, model.UnknownError
	}
	return model.PaginatedResponse{Data: list, Count: count, Page: f.Page, Size: f.Size}, nil
}

func GetBloodBank(id int) (model.BloodBank, error) {
	b, err := scanBank(DB.QueryRow("SELECT "+bankColumns+bankFrom+" WHERE b.id = $1", id))
	if errors.Is(err, sql.ErrNoRows) {
		return model.BloodBank{}, model.NewError(itn.ErrorBankNotFound, 404)
	}
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return model.BloodBank{}, model.UnknownError
	}
	return b, nil
}

// GetPendingBankRequest returns the creation request the user is waiting
// on, or nil.
func GetPendingBankRequest(userID int) (*model.BloodBank, error) {
	b, err := scanBank(DB.QueryRow("SELECT "+bankColumns+bankFrom+
		" WHERE b.created_by = $1 AND b.status = 'pending'", userID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return nil, model.UnknownError
	}
	return &b, nil
}

// UpdateBloodBank edits the descriptive fields of a bank (logo and cover
// are set through the image endpoints).
func UpdateBloodBank(id int, b model.BloodBank) error {
	_, err := DB.Exec(`UPDATE blood_banks SET name = $2, description = NULLIF($3, ''),
			address = NULLIF($4, ''), contact_phone = NULLIF($5, ''), updated_at = now()
		WHERE id = $1`, id, b.Name, b.Description, b.Address, b.ContactPhone)
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return model.UnknownError
	}
	return nil
}

// ApproveBloodBank approves a pending (or re-activates a suspended) bank.
// For a pending bank it also makes the creator the active admin member,
// creates the bank chat and awards the founder badge. It returns whether
// the founder badge was newly awarded.
func ApproveBloodBank(id, reviewerID int) (model.BloodBank, *model.Badge, error) {
	tx, err := DB.Begin()
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return model.BloodBank{}, nil, model.UnknownError
	}
	defer tx.Rollback()

	var status string
	var adminID int
	err = tx.QueryRow(`SELECT status, admin_id FROM blood_banks WHERE id = $1 FOR UPDATE`, id).Scan(&status, &adminID)
	if errors.Is(err, sql.ErrNoRows) {
		return model.BloodBank{}, nil, model.NewError(itn.ErrorBankNotFound, 404)
	}
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return model.BloodBank{}, nil, model.UnknownError
	}
	if status != model.BankStatusPending && status != model.BankStatusSuspended {
		return model.BloodBank{}, nil, model.NewError(itn.ErrorBankNotPending, 409)
	}

	if _, err = tx.Exec(`UPDATE blood_banks SET status = 'approved', reviewed_by = $2, reviewed_at = now(),
			rejection_reason = NULL, updated_at = now() WHERE id = $1`, id, reviewerID); err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return model.BloodBank{}, nil, model.UnknownError
	}

	var founder *model.Badge
	if status == model.BankStatusPending {
		_, err = tx.Exec(`INSERT INTO blood_bank_members (blood_bank_id, user_id, role, position, status, approved_by, joined_at)
			VALUES ($1, $2, 'admin', 'Founder', 'active', $3, now())`, id, adminID, reviewerID)
		if isUniqueViolation(err, "ux_blood_bank_members_one_bank") {
			// the creator joined another bank while the request waited
			return model.BloodBank{}, nil, model.NewError(itn.ErrorBankAlreadyMember, 409)
		}
		if err != nil {
			logger.GetLogger().LogErrors(err, nil)
			return model.BloodBank{}, nil, model.UnknownError
		}

		convID, err := createBankConversation(tx, id)
		if err != nil {
			return model.BloodBank{}, nil, err
		}
		if err = addParticipant(tx, convID, adminID); err != nil {
			return model.BloodBank{}, nil, err
		}

		if err = awardPoints(tx, adminID, model.PointsBankFounder, model.PointReasonBankFounder, "blood_bank", id); err != nil {
			return model.BloodBank{}, nil, err
		}
		if founder, err = awardBadge(tx, adminID, model.BadgeBankFounder, id, ""); err != nil {
			return model.BloodBank{}, nil, err
		}
	}

	if err = tx.Commit(); err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return model.BloodBank{}, nil, model.UnknownError
	}

	b, err := GetBloodBank(id)
	return b, founder, err
}

// SetBloodBankStatus moves a bank from one of the `from` statuses to
// `to` (reject / suspend), recording the reviewer and reason.
func SetBloodBankStatus(id, reviewerID int, from []string, to, reason string) (model.BloodBank, error) {
	res, err := DB.Exec(`UPDATE blood_banks SET status = $2, reviewed_by = $3, reviewed_at = now(),
			rejection_reason = NULLIF($4, ''), updated_at = now()
		WHERE id = $1 AND status = ANY($5)`, id, to, reviewerID, reason, stringArray(from))
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return model.BloodBank{}, model.UnknownError
	}
	if n, _ := res.RowsAffected(); n == 0 {
		if _, err := GetBloodBank(id); err != nil {
			return model.BloodBank{}, err
		}
		return model.BloodBank{}, model.NewError(itn.ErrorBankNotPending, 409)
	}
	return GetBloodBank(id)
}

// ---------------------------------------------------------------------
// Members
// ---------------------------------------------------------------------

const memberColumns = `
	m.id, m.blood_bank_id, m.user_id, COALESCE(u.first_name, ''), COALESCE(u.last_name, ''),
	COALESCE(u.blood_group, ''), COALESCE(u.phone, ''), COALESCE(u.upazila, ''),
	m.role, COALESCE(m.position, ''), m.status, u.points, u.donation_count, u.donated_at,
	m.joined_at, m.created_at, COALESCE(u.hide_personal_info, false), COALESCE(u.photo, '')`

func scanMember(row rowScanner) (model.BankMember, error) {
	var m model.BankMember
	var donated, joined sql.NullTime
	err := row.Scan(&m.Id, &m.BloodBankId, &m.UserId, &m.FirstName, &m.LastName,
		&m.BloodGroup, &m.Phone, &m.Upazila, &m.Role, &m.Position, &m.Status,
		&m.Points, &m.DonationCount, &donated, &joined, &m.CreatedAt, &m.HidePersonalInfo, &m.Photo)
	if donated.Valid {
		m.DonatedAt = donated.Time.Format("2006-01-02")
	}
	if joined.Valid {
		t := joined.Time
		m.JoinedAt = &t
	}
	return m, err
}

// GetMembership returns the user's pending or active membership, or nil.
func GetMembership(userID int) (*model.BankMember, error) {
	m, err := scanMember(DB.QueryRow(`SELECT `+memberColumns+`
		FROM blood_bank_members m JOIN users u ON u.id = m.user_id
		WHERE m.user_id = $1 AND m.status IN ('pending', 'active')`, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return nil, model.UnknownError
	}
	return &m, nil
}

// BankRole returns the user's role in an active membership of the bank,
// or "" when they are not an active member.
func BankRole(bankID, userID int) (string, error) {
	var role string
	err := DB.QueryRow(`SELECT role FROM blood_bank_members
		WHERE blood_bank_id = $1 AND user_id = $2 AND status = 'active'`, bankID, userID).Scan(&role)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return "", model.UnknownError
	}
	return role, nil
}

// GetBankMembers lists members with the given status, admins and
// officers first, then by points.
func GetBankMembers(bankID int, status string) ([]model.BankMember, error) {
	rows, err := DB.Query(`SELECT `+memberColumns+`
		FROM blood_bank_members m JOIN users u ON u.id = m.user_id
		WHERE m.blood_bank_id = $1 AND m.status = $2 AND u.is_available = true
		ORDER BY CASE m.role WHEN 'admin' THEN 0 WHEN 'officer' THEN 1 ELSE 2 END,
			u.points DESC, m.created_at ASC`, bankID, status)
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return nil, model.UnknownError
	}
	defer rows.Close()

	list := []model.BankMember{}
	for rows.Next() {
		m, err := scanMember(rows)
		if err != nil {
			logger.GetLogger().LogErrors(err, nil)
			return nil, model.UnknownError
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

// BankOfficerIDs returns the user ids of the bank's admin and officers.
func BankOfficerIDs(bankID int) ([]int, error) {
	return queryInts(`SELECT user_id FROM blood_bank_members
		WHERE blood_bank_id = $1 AND status = 'active' AND role IN ('admin', 'officer')`, bankID)
}

// JoinBloodBank creates a pending membership request.
func JoinBloodBank(bankID, userID int) error {
	_, err := DB.Exec(`INSERT INTO blood_bank_members (blood_bank_id, user_id, role, status)
		VALUES ($1, $2, 'member', 'pending')`, bankID, userID)
	if isUniqueViolation(err, "ux_blood_bank_members_one_bank") {
		return model.NewError(itn.ErrorBankAlreadyMember, 409)
	}
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return model.UnknownError
	}
	return nil
}

// ReviewMember approves (active) or rejects a pending membership. An
// approved member is added to the bank chat.
func ReviewMember(bankID, userID, reviewerID int, approve bool) error {
	tx, err := DB.Begin()
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return model.UnknownError
	}
	defer tx.Rollback()

	status := model.MemberStatusRejected
	joined := "NULL"
	if approve {
		status = model.MemberStatusActive
		joined = "now()"
	}
	res, err := tx.Exec(`UPDATE blood_bank_members SET status = $3, approved_by = $4, joined_at = `+joined+`,
			updated_at = now()
		WHERE blood_bank_id = $1 AND user_id = $2 AND status = 'pending'`, bankID, userID, status, reviewerID)
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return model.UnknownError
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return model.NewError(itn.ErrorBankMemberNotFound, 404)
	}

	if approve {
		convID, err := createBankConversation(tx, bankID)
		if err != nil {
			return err
		}
		if err = addParticipant(tx, convID, userID); err != nil {
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return model.UnknownError
	}
	return nil
}

// EndMembership marks a pending or active membership as left / removed
// and takes the user out of the bank chat. The bank admin's membership
// cannot be ended (the admin must transfer first).
func EndMembership(bankID, userID int, status string) error {
	tx, err := DB.Begin()
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return model.UnknownError
	}
	defer tx.Rollback()

	var role string
	err = tx.QueryRow(`SELECT role FROM blood_bank_members
		WHERE blood_bank_id = $1 AND user_id = $2 AND status IN ('pending', 'active') FOR UPDATE`,
		bankID, userID).Scan(&role)
	if errors.Is(err, sql.ErrNoRows) {
		return model.NewError(itn.ErrorBankMemberNotFound, 404)
	}
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return model.UnknownError
	}
	if role == model.BankRoleAdmin {
		return model.NewError(itn.ErrorBankAdminCannotLeave, 409)
	}

	if _, err = tx.Exec(`UPDATE blood_bank_members SET status = $3, updated_at = now()
		WHERE blood_bank_id = $1 AND user_id = $2 AND status IN ('pending', 'active')`,
		bankID, userID, status); err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return model.UnknownError
	}
	if _, err = tx.Exec(`DELETE FROM conversation_participants cp USING conversations c
		WHERE cp.conversation_id = c.id AND c.blood_bank_id = $1 AND cp.user_id = $2`, bankID, userID); err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return model.UnknownError
	}

	if err = tx.Commit(); err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return model.UnknownError
	}
	return nil
}

// SetMemberRole changes an active non-admin member's role and position.
func SetMemberRole(bankID, userID int, role, position string) error {
	res, err := DB.Exec(`UPDATE blood_bank_members SET role = $3, position = NULLIF($4, ''), updated_at = now()
		WHERE blood_bank_id = $1 AND user_id = $2 AND status = 'active' AND role <> 'admin'`,
		bankID, userID, role, position)
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return model.UnknownError
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return model.NewError(itn.ErrorBankMemberNotFound, 404)
	}
	return nil
}

// SetAdminPosition changes only the admin's position title.
func SetAdminPosition(bankID, userID int, position string) error {
	_, err := DB.Exec(`UPDATE blood_bank_members SET position = NULLIF($3, ''), updated_at = now()
		WHERE blood_bank_id = $1 AND user_id = $2 AND status = 'active' AND role = 'admin'`,
		bankID, userID, position)
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return model.UnknownError
	}
	return nil
}

// TransferBankAdmin makes newAdminID (an active member) the bank admin;
// the previous admin becomes an officer. It returns the previous admin.
func TransferBankAdmin(bankID, newAdminID int) (int, error) {
	tx, err := DB.Begin()
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return 0, model.UnknownError
	}
	defer tx.Rollback()

	var oldAdminID int
	var status string
	err = tx.QueryRow(`SELECT admin_id, status FROM blood_banks WHERE id = $1 FOR UPDATE`, bankID).Scan(&oldAdminID, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, model.NewError(itn.ErrorBankNotFound, 404)
	}
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return 0, model.UnknownError
	}
	if status != model.BankStatusApproved {
		return 0, model.NewError(itn.ErrorBankNotApproved, 409)
	}
	if oldAdminID == newAdminID {
		return 0, model.NewError(itn.ErrorBankTransferTargetInvalid, 400)
	}

	res, err := tx.Exec(`UPDATE blood_bank_members SET role = 'admin', updated_at = now()
		WHERE blood_bank_id = $1 AND user_id = $2 AND status = 'active'`, bankID, newAdminID)
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return 0, model.UnknownError
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return 0, model.NewError(itn.ErrorBankTransferTargetInvalid, 400)
	}
	if _, err = tx.Exec(`UPDATE blood_bank_members SET role = 'officer', updated_at = now()
		WHERE blood_bank_id = $1 AND user_id = $2 AND status = 'active' AND role = 'admin'`, bankID, oldAdminID); err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return 0, model.UnknownError
	}
	if _, err = tx.Exec(`UPDATE blood_banks SET admin_id = $2, updated_at = now() WHERE id = $1`, bankID, newAdminID); err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return 0, model.UnknownError
	}

	if err = tx.Commit(); err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return 0, model.UnknownError
	}
	return oldAdminID, nil
}

// ---------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------

func queryInts(query string, args ...interface{}) ([]int, error) {
	rows, err := DB.Query(query, args...)
	if err != nil {
		logger.GetLogger().LogErrors(err, map[string]interface{}{"query": query})
		return nil, model.UnknownError
	}
	defer rows.Close()

	ids := []int{}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			logger.GetLogger().LogErrors(err, nil)
			return nil, model.UnknownError
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func nullTimePtr(t sql.NullTime) *time.Time {
	if !t.Valid {
		return nil
	}
	v := t.Time
	return &v
}
