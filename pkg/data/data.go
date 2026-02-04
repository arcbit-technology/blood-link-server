package data

import "encoding/json"

// HiddenJsonString is a string that is marshaled as an empty string in JSON
type HiddenJsonString string

func (h HiddenJsonString) MarshalJSON() ([]byte, error) {
	return json.Marshal("")
}
