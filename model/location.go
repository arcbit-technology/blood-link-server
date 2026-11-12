package model

// IsValidDivision reports whether d is one of the 8 divisions.
func IsValidDivision(d string) bool {
	_, ok := DISTRICTS[Division(d)]
	return ok
}

// IsDistrictInDivision reports whether district belongs to division.
func IsDistrictInDivision(division, district string) bool {
	for _, d := range DISTRICTS[Division(division)] {
		if string(d) == district {
			return true
		}
	}
	return false
}

// IsValidUpazila reports whether upazila belongs to district.
func IsValidUpazila(district, upazila string) bool {
	for _, u := range UPAZILAS[District(district)] {
		if u == upazila {
			return true
		}
	}
	return false
}
