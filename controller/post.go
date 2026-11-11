package controller

import (
	"strings"
	"time"
	"unicode/utf8"

	dao "github.com/mdhasib01/go-rest-starter/dao"
	itn "github.com/mdhasib01/go-rest-starter/itn"
	model "github.com/mdhasib01/go-rest-starter/model"
)

const (
	defaultPostPageSize = 30
	maxPostPageSize     = 100
	maxUnitsPerPost     = 20
)

func GetPosts(f model.PostFilter) (model.PaginatedResponse, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Size < 1 {
		f.Size = defaultPostPageSize
	}
	if f.Size > maxPostPageSize {
		f.Size = maxPostPageSize
	}
	return dao.GetPosts(f)
}

func GetPost(id int) (model.DonationPost, error) {
	return dao.GetPost(id)
}

func CreatePost(p model.DonationPost, session model.Session) (model.DonationPost, error) {
	p.AuthorId = session.UserId
	if err := validatePost(&p); err != nil {
		return model.DonationPost{}, err
	}

	id, err := dao.CreatePost(p)
	if err != nil {
		return model.DonationPost{}, err
	}
	return dao.GetPost(id)
}

// ToggleInterest flips the session user's interest in a post and
// returns the updated post.
func ToggleInterest(postID int, session model.Session) (model.DonationPost, error) {
	if _, err := dao.ToggleInterest(postID, session.UserId); err != nil {
		return model.DonationPost{}, err
	}
	return dao.GetPost(postID)
}

func validatePost(p *model.DonationPost) error {
	p.Title = strings.TrimSpace(p.Title)
	p.Description = strings.TrimSpace(p.Description)
	p.HospitalName = strings.TrimSpace(p.HospitalName)
	p.Address = strings.TrimSpace(p.Address)
	p.District = strings.TrimSpace(p.District)
	p.Division = strings.TrimSpace(p.Division)
	p.ContactPhone = strings.TrimSpace(p.ContactPhone)
	p.BloodGroup = strings.ToUpper(strings.TrimSpace(p.BloodGroup))

	if p.Title == "" || utf8.RuneCountInString(p.Title) > 255 {
		return model.NewError(itn.ErrorPostTitleRequired, 400)
	}
	if !model.IsValidBloodGroup(p.BloodGroup) {
		return model.NewError(itn.ErrorPostBloodGroupInvalid, 400)
	}
	if p.HospitalName == "" || utf8.RuneCountInString(p.HospitalName) > 255 {
		return model.NewError(itn.ErrorPostHospitalRequired, 400)
	}
	if p.Division == "" || p.District == "" {
		return model.NewError(itn.ErrorPostLocationRequired, 400)
	}

	required, err := time.Parse("2006-01-02", strings.TrimSpace(p.RequiredDate))
	today := time.Now().Truncate(24 * time.Hour)
	if err != nil || required.Before(today.AddDate(0, 0, -1)) {
		return model.NewError(itn.ErrorPostRequiredDateInvalid, 400)
	}
	p.RequiredDate = required.Format("2006-01-02")

	if p.UnitsNeeded < 1 {
		p.UnitsNeeded = 1
	}
	if p.UnitsNeeded > maxUnitsPerPost {
		return model.NewError(itn.ErrorInvalidData, 400)
	}
	return nil
}
