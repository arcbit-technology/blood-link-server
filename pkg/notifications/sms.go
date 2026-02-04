package notifier

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	config "github.com/mdhasib01/go-rest-starter/config"
	model "github.com/mdhasib01/go-rest-starter/model"
	"github.com/mdhasib01/go-rest-starter/pkg/logger"
)

type SmSService interface {
	Send() (*model.VoipmsResponse, error)
}

type RealSmSService struct {
	Destination string `json:"destination"`
	Body        string `json:"body"`
}

type MockSmSService struct {
	Destination string `json:"destination"`
	Body        string `json:"body"`
}

func NewSmSService() SmSService {
	return &RealSmSService{}
}

func NewMockSmSService() SmSService {
	return &MockSmSService{}
}

var smsGatewayBaseUrl = "https://voip.ms/api/v1/rest.php"

func (sms *RealSmSService) Send() (*model.VoipmsResponse, error) {

	apiURL := fmt.Sprintf("%s?api_username=%s&api_password=%s&method=sendSMS&did=%s&dst=%s&message=%s",
		smsGatewayBaseUrl, url.QueryEscape(config.Param.VOIP_DETAIL.USER_NAME), url.QueryEscape(config.Param.VOIP_DETAIL.PASSWORD), config.Param.VOIP_DETAIL.VOIP_DID, sms.Destination, url.QueryEscape(sms.Body))

	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("error_during_the_api_request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.GetLogger().LogErrors(err, map[string]interface{}{"err": err.Error(), "message": "unable_to_read_response_voip_response", "phone_number": sms.Destination, "body": sms.Body})
		return nil, fmt.Errorf("error_while_reading_the_response: %v", err)
	}

	var voipmsResp model.VoipmsResponse
	if err = json.Unmarshal(body, &voipmsResp); err != nil {
		logger.GetLogger().LogErrors(err, map[string]interface{}{"err": err.Error(), "message": "unable_to_marshal_voip_response", "phone_number": sms.Destination, "body": sms.Body})
		return nil, fmt.Errorf("error_while_decoding_the_json_response: %v", err)
	}
	return &voipmsResp, nil
}

func (sms *MockSmSService) Send() (*model.VoipmsResponse, error) {

	apiURL := fmt.Sprintf("%s?api_username=%s&api_password=%s&method=sendSMS&did=%s&dst=%s&message=%s",
		smsGatewayBaseUrl, url.QueryEscape(config.Param.VOIP_DETAIL.USER_NAME), url.QueryEscape(config.Param.VOIP_DETAIL.PASSWORD), config.Param.VOIP_DETAIL.VOIP_DID, sms.Destination, url.QueryEscape(sms.Body))

	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("error_during_the_api_request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error_while_reading_the_response: %v", err)
	}

	var voipmsResp model.VoipmsResponse
	if err = json.Unmarshal(body, &voipmsResp); err != nil {
		return nil, fmt.Errorf("error_while_decoding_the_json_response: %v", err)
	}

	return &voipmsResp, nil
}
