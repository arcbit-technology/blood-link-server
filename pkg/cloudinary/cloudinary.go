// Package cloudinary signs direct browser uploads and manages images on
// Cloudinary without exposing the API secret.
//
// Flow: the API signs an upload for a fixed public id (e.g.
// bloodlink/users/42/avatar), the browser uploads the file straight to
// Cloudinary, then tells the API the resulting version / format and the
// API stores the delivery URL.
package cloudinary

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// AllowedFormats are the image formats accepted for uploads.
var AllowedFormats = []string{"jpg", "jpeg", "png", "webp"}

type Client struct {
	CloudName string
	ApiKey    string
	ApiSecret string
	HTTP      *http.Client
}

func New(cloudName, apiKey, apiSecret string) *Client {
	return &Client{CloudName: cloudName, ApiKey: apiKey, ApiSecret: apiSecret,
		HTTP: &http.Client{Timeout: 10 * time.Second}}
}

// SignedUpload is everything the browser needs to upload one image.
// Every field except UploadURL, CloudName, ApiKey and Signature must be
// sent to Cloudinary unchanged, or the signature check fails.
type SignedUpload struct {
	UploadURL      string `json:"upload_url"`
	CloudName      string `json:"cloud_name"`
	ApiKey         string `json:"api_key"`
	Timestamp      int64  `json:"timestamp"`
	Signature      string `json:"signature"`
	PublicId       string `json:"public_id"`
	Overwrite      string `json:"overwrite"`
	Invalidate     string `json:"invalidate"`
	AllowedFormats string `json:"allowed_formats"`
}

// Sign computes a Cloudinary API signature: the parameters sorted by name,
// joined as k=v with &, followed by the secret, hashed with SHA-1.
func Sign(params map[string]string, secret string) string {
	keys := make([]string, 0, len(params))
	for k, v := range params {
		if v != "" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = k + "=" + params[k]
	}
	sum := sha1.Sum([]byte(strings.Join(parts, "&") + secret))
	return hex.EncodeToString(sum[:])
}

// SignUpload signs an upload that overwrites the image at publicID.
func (c *Client) SignUpload(publicID string, now time.Time) SignedUpload {
	u := SignedUpload{
		UploadURL:      fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/image/upload", c.CloudName),
		CloudName:      c.CloudName,
		ApiKey:         c.ApiKey,
		Timestamp:      now.Unix(),
		PublicId:       publicID,
		Overwrite:      "true",
		Invalidate:     "true",
		AllowedFormats: strings.Join(AllowedFormats, ","),
	}
	u.Signature = Sign(map[string]string{
		"timestamp":       strconv.FormatInt(u.Timestamp, 10),
		"public_id":       u.PublicId,
		"overwrite":       u.Overwrite,
		"invalidate":      u.Invalidate,
		"allowed_formats": u.AllowedFormats,
	}, c.ApiSecret)
	return u
}

// DeliveryURL is the canonical HTTPS URL of an uploaded image version.
func (c *Client) DeliveryURL(publicID string, version int64, format string) string {
	return fmt.Sprintf("https://res.cloudinary.com/%s/image/upload/v%d/%s.%s", c.CloudName, version, publicID, format)
}

// Resource is the stored state of an uploaded image (Admin API).
type Resource struct {
	PublicId string `json:"public_id"`
	Version  int64  `json:"version"`
	Format   string `json:"format"`
	Bytes    int64  `json:"bytes"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}

// Resource looks up an uploaded image. found is false when there is none.
// Unlike the delivery URL (which serves any version number), this returns
// the real current version.
func (c *Client) Resource(publicID string) (res Resource, found bool, err error) {
	req, err := http.NewRequest(http.MethodGet,
		fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/resources/image/upload/%s", c.CloudName, publicID), nil)
	if err != nil {
		return res, false, err
	}
	req.SetBasicAuth(c.ApiKey, c.ApiSecret)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return res, false, err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		err = json.NewDecoder(resp.Body).Decode(&res)
		return res, err == nil, err
	case http.StatusNotFound:
		return res, false, nil
	default:
		return res, false, fmt.Errorf("cloudinary: resource %s returned %d", publicID, resp.StatusCode)
	}
}

// Destroy deletes the image and invalidates its cached copies.
func (c *Client) Destroy(publicID string) error {
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	form := url.Values{
		"public_id":  {publicID},
		"timestamp":  {ts},
		"invalidate": {"true"},
		"api_key":    {c.ApiKey},
		"signature": {Sign(map[string]string{
			"public_id": publicID, "timestamp": ts, "invalidate": "true"}, c.ApiSecret)},
	}
	resp, err := c.HTTP.PostForm(fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/image/destroy", c.CloudName), form)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("cloudinary: destroy %s returned %d", publicID, resp.StatusCode)
	}
	return nil
}
