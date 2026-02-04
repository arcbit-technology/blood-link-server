package geoip

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	config "github.com/mdhasib01/go-rest-starter/config"
	"github.com/mdhasib01/go-rest-starter/pkg/logger"
)

// Response mirrors the structure of the MaxMind GeoIP2 Response
// ensuring compatibility with existing code usage.
type Response struct {
	City    City    `json:"city"`
	Country Country `json:"country"`
	Traits  Traits  `json:"traits"`
}

type City struct {
	GeoNameId int               `json:"geoname_id"`
	Names     map[string]string `json:"names"`
}

type Country struct {
	GeoNameId int               `json:"geoname_id"`
	Names     map[string]string `json:"names"`
}

type Traits struct {
	IpAddress string `json:"ip_address"`
}

type GeoIP interface {
	GetLocation(ip string) (Response, error)
}

type GeoIPService struct {
	Client *http.Client
}

type MockGeoIP struct{}

var geoip GeoIP

func InitGeoIP() {
	service := &GeoIPService{
		Client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
	geoip = service
}

func InitMockGeoIP() {
	mockService := &MockGeoIP{}
	geoip = mockService
}

func GetLocation(ip string) (Response, error) {
	if geoip == nil {
		return Response{}, errors.New("geoip service not initialized")
	}
	return geoip.GetLocation(ip)
}

func (s *GeoIPService) GetLocation(ip string) (Response, error) {
	// Try Insights first as it was the original intent
	url := fmt.Sprintf("https://geoip.maxmind.com/geoip/v2.1/insights/%s", ip)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return Response{}, err
	}

	req.SetBasicAuth(config.Param.Geoip2UserID, config.Param.Geoip2LicenseKey)
	req.Header.Set("Accept", "application/json")

	resp, err := s.Client.Do(req)
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return Response{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// If Insights fails (e.g. unauthorized/not licensed), maybe try City?
		// For now, let's log the error. The user can switch to City endpoint if needed.
		// Note: 401/403 often returns JSON error, but 404/500 might return HTML.
		err = fmt.Errorf("geoip API returned status: %d", resp.StatusCode)
		logger.GetLogger().LogErrors(err, map[string]interface{}{
			"status": resp.Status,
			"url":    url,
		})
		return Response{}, err
	}

	var location Response
	if err := json.NewDecoder(resp.Body).Decode(&location); err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return location, err
	}

	return location, nil
}

func (m *MockGeoIP) GetLocation(ip string) (Response, error) {
	return Response{}, errors.New("failed to get location")
}
