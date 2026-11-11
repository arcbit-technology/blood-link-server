package controller

import (
	"strings"
	"time"

	dao "github.com/mdhasib01/go-rest-starter/dao"
	itn "github.com/mdhasib01/go-rest-starter/itn"
	model "github.com/mdhasib01/go-rest-starter/model"
)

// normalizeDonorFields cleans the donor-specific fields of a user before
// it is stored: gender as M / F / other, upper-case blood group, a sane
// age and a YYYY-MM-DD last donation date.
func normalizeDonorFields(user *model.User) error {
	switch strings.ToLower(strings.TrimSpace(user.Gender)) {
	case "m", "male":
		user.Gender = "M"
	case "f", "female":
		user.Gender = "F"
	case "":
		user.Gender = ""
	default:
		user.Gender = "other"
	}

	user.BloodGroup = strings.ToUpper(strings.TrimSpace(user.BloodGroup))
	if user.BloodGroup != "" && !model.IsValidBloodGroup(user.BloodGroup) {
		return model.NewError(itn.ErrorInvalidData, 400)
	}

	if user.Age != 0 && (user.Age < 16 || user.Age > 100) {
		return model.NewError(itn.ErrorInvalidData, 400)
	}

	user.DonatedAt = strings.TrimSpace(user.DonatedAt)
	if user.DonatedAt != "" {
		var parsed time.Time
		var err error
		for _, layout := range []string{"2006-01-02", time.RFC3339, "2006-01-02T15:04:05.000Z"} {
			if parsed, err = time.Parse(layout, user.DonatedAt); err == nil {
				break
			}
		}
		if err != nil || parsed.After(time.Now()) {
			return model.NewError(itn.ErrorInvalidData, 400)
		}
		user.DonatedAt = parsed.Format("2006-01-02")
	}

	user.Division = strings.TrimSpace(user.Division)
	user.District = strings.TrimSpace(user.District)
	return nil
}

const (
	defaultDonorPageSize = 12
	maxDonorPageSize     = 100
)

func GetDonors(f model.DonorFilter) (model.PaginatedResponse, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Size < 1 {
		f.Size = defaultDonorPageSize
	}
	if f.Size > maxDonorPageSize {
		f.Size = maxDonorPageSize
	}

	res, err := dao.GetDonors(f)
	if err != nil {
		return res, err
	}

	if list, ok := res.Data.([]model.Donor); ok {
		for i := range list {
			redactDonor(&list[i])
		}
	}
	return res, nil
}

func GetDonor(id int) (model.Donor, error) {
	d, err := dao.GetDonor(id)
	if err != nil {
		return d, err
	}
	redactDonor(&d)
	return d, nil
}

// redactDonor strips identifying details from donors who chose to hide
// their personal info. Blood group, gender and location stay visible so
// they can still be found.
func redactDonor(d *model.Donor) {
	if !d.HidePersonalInfo {
		return
	}
	d.Firstname = ""
	d.Lastname = ""
	d.Phone = ""
	d.Address = ""
	d.Age = 0
}
