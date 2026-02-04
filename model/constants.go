package model

const (
	SUPERADMIN = 1
	ADMIN      = 2
	MANAGER    = 3
	USERROLE   = 4

	OUTBOUND_USER = 0

	MAX_PROFILE_PICTURE_FILE_SIZE = 10 * 1024 * 1024 // 10MB

	ConfirmationEmailBody = `Hello %s, <br> <br>
	Please verify your account by clicking the link below: <br>
	<a href="%s">Verify Account</a> <br> <br>
	Thank you, <br>
	BloodLinkDB`

	HtmlErrorVerificationPage = `<html><body><h1>Error</h1><p>%s</p></body></html>`
	HtmlVerifiedEmailPage     = `<html><body><h1>Verified</h1><p>Hello %s, your email is verified. <a href="%s">Login here</a></p></body></html>`
)

func ProfessionExists(p Profession) bool {
	// TODO: Implement actual check
	return true
}
