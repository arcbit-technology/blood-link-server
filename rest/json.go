package rest

import (
	"encoding/json"
	"net/http"

	"github.com/mdhasib01/go-rest-starter/model"
	"github.com/mdhasib01/go-rest-starter/pkg/logger"
)

const filePath = "rest/json.go"

func JSON(w http.ResponseWriter, data interface{}, err error) {
	w.Header().Set("Content-type", "application/json;charset=UTF-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")

	var resp model.BasicResponse
	var p model.Error
	var statusCode int
	var ok bool

	if err != nil && err.Error() != "" {
		resp.Error = err.Error()
		p, ok = err.(model.Error)
		if ok {
			statusCode = p.Code()
		} else {
			statusCode = 500
		}
	} else {
		resp.Target = data
		statusCode = 200
	}

	w.WriteHeader(statusCode)
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
	}
}

func ERROR(w http.ResponseWriter, statusCode int, err error) {
	w.Header().Set("Content-type", "application/json;charset=UTF-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")

	resp := model.BasicResponse{Error: err.Error()}
	w.WriteHeader(statusCode)

	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
	}
}
