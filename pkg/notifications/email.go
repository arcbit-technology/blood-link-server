package notifier

import (
	"fmt"
	"strconv"

	config "github.com/mdhasib01/go-rest-starter/config"
	model "github.com/mdhasib01/go-rest-starter/model"
	"github.com/mdhasib01/go-rest-starter/pkg/logger"

	"gopkg.in/gomail.v2"
)

var ES EmailService

type EmailService interface {
	SendEmail(mail model.Mail) error
}

type RealEmailService struct {
}

type MockEmailService struct {
}

func NewEmailService() EmailService {
	return &RealEmailService{}
}

func NewMockEmailService() EmailService {
	return &MockEmailService{}
}

func (es *RealEmailService) SendEmail(mail model.Mail) error {

	// TODO: ADJUST LOGS FOR INFO

	smtpPort, _ := strconv.Atoi(config.Param.SMTP_DETAILS.SMTP_SERVER_PORT)
	smtpserverObj := gomail.NewDialer(config.Param.SMTP_DETAILS.SMTP_SERVER, smtpPort, config.Param.SMTP_DETAILS.SMTP_USER, config.Param.SMTP_DETAILS.SMTP_PASS)

	mailObj := gomail.NewMessage()
	mailObj.SetAddressHeader("From", mail.SENDER, mail.SENDER_NAME)
	mailObj.SetAddressHeader("To", mail.ReceiverEmail, mail.ReceiverName)

	mailObj.SetHeader("Subject", mail.SUBJECT)
	mailObj.SetBody("text/html", mail.BODY)

	logger.GetLogger().LogInfo(fmt.Sprintf("Sending email to %s", mail.ReceiverEmail), nil)

	if err := smtpserverObj.DialAndSend(mailObj); err != nil {
		return err
	}

	return nil
}
func (mes *MockEmailService) SendEmail(mail model.Mail) error {
	return nil
}
