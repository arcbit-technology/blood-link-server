package rest

const (
	ChangePasswordPermission = iota + 1
	ResetPasswordPermission
	CreateUserPermission
	UpdateUserPermission
	ListUserPermission
	ViewUserPermission
	DeleteUserPermission
	ToggleUserStatusPermission
	ProfileLoginPermission
	ManageBloodBanksPermission // 10, seeded by migration 000003
)
