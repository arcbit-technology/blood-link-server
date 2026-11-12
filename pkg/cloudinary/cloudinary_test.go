package cloudinary

import (
	"testing"
	"time"
)

// Example from the Cloudinary "Authentication signatures" documentation.
func TestSignMatchesCloudinaryDocs(t *testing.T) {
	got := Sign(map[string]string{
		"eager":     "w_400,h_300,c_pad|w_260,h_200,c_crop",
		"public_id": "sample_image",
		"timestamp": "1315060510",
	}, "abcd")
	if want := "bfd09f95f331f558cbd1320e67aa8d488770583e"; got != want {
		t.Fatalf("signature = %s, want %s", got, want)
	}
}

func TestSignSkipsEmptyParams(t *testing.T) {
	a := Sign(map[string]string{"public_id": "x", "timestamp": "1"}, "s")
	b := Sign(map[string]string{"public_id": "x", "timestamp": "1", "folder": ""}, "s")
	if a != b {
		t.Fatal("empty parameters must not change the signature")
	}
}

func TestSignUploadAndURL(t *testing.T) {
	c := New("demo", "key", "secret")
	u := c.SignUpload("bloodlink/users/7/avatar", time.Unix(1700000000, 0))
	if u.UploadURL != "https://api.cloudinary.com/v1_1/demo/image/upload" || u.Timestamp != 1700000000 {
		t.Fatalf("unexpected upload %+v", u)
	}
	want := Sign(map[string]string{
		"timestamp": "1700000000", "public_id": "bloodlink/users/7/avatar",
		"overwrite": "true", "invalidate": "true", "allowed_formats": "jpg,jpeg,png,webp",
	}, "secret")
	if u.Signature != want {
		t.Fatal("upload signature does not cover the upload parameters")
	}
	if got := c.DeliveryURL("bloodlink/users/7/avatar", 12, "webp"); got !=
		"https://res.cloudinary.com/demo/image/upload/v12/bloodlink/users/7/avatar.webp" {
		t.Fatalf("url = %s", got)
	}
}
