package rest

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	controller "github.com/mdhasib01/go-rest-starter/controller"
	"github.com/mdhasib01/go-rest-starter/itn"
	"github.com/mdhasib01/go-rest-starter/model"
	"github.com/mdhasib01/go-rest-starter/pkg/logger"
)

var invalidDataError = model.NewError(itn.ErrorInvalidData, 400)

func isAuthorized(r *http.Request, idPermission int) (model.Session, error) {
	token := r.Header.Get("Authorization")
	token = strings.Replace(token, "Bearer ", "", 1)
	return controller.IsAuthorized(token, idPermission)
}

func readRequestBody(r *http.Request, dst any) error {
	var typeError *json.UnmarshalTypeError

	data, err := io.ReadAll(r.Body)
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return model.NewError(itn.ErrorInvalidData, http.StatusBadRequest)
	}

	err = json.Unmarshal(data, &dst)
	if err != nil {
		if errors.As(err, &typeError) {
			logger.GetLogger().LogErrors(err, nil)
			return model.NewError(itn.ErrorInvalidTypeInJSONData, http.StatusBadRequest)
		} else {
			return model.NewError(itn.ErrorInvalidData, http.StatusBadRequest)
		}
	}
	return nil
}
