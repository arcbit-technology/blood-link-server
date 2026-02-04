package notifier

import (
	"os"
	"path/filepath"
	"testing"

	config "github.com/mdhasib01/go-rest-starter/config"

	"github.com/stretchr/testify/assert"
)

func TestSmS(t *testing.T) {

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal("Failed to get working dir: ", err)
	}

	rootPath := filepath.Join(cwd, "..", "..")

	err = config.InitConfig(rootPath)
	if err != nil {
		t.Fatal("Failed to load config: " + err.Error())
	}

	mockSms := MockSmSService{
		Destination: "+111111111",
		Body:        "Dear {seller_name}, please upload {doc_name} to proceed with Deal #1.",
	}

	response, err := mockSms.Send()

	assert.NoError(t, err)

	assert.Equal(t, "success", response.Status)

}
