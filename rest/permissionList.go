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
)
