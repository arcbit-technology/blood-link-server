package model

type BasicResponse struct {
	Target interface{} `json:"result"`
	Error  string      `json:"error"`
}
