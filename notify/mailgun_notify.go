package notify

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mailgun/mailgun-go"
)

var mailGunClient mailgun.Mailgun
var ctx context.Context

type MailgunNotify struct {
	Email        string `json:"email"`
	ApiKey       string `json:"apiKey"`
	Domain       string `json:"domain"`
	PublicApiKey string `json:"publicApiKey"`
}

func (mailgunNotify MailgunNotify) GetClientName() string {
	return "Mailgun"
}

func (mailgunNotify MailgunNotify) Initialize() error {
	if !validateEmail(mailgunNotify.Email) {
		return errors.New("mailgun: invalid email address")
	}

	if len(strings.TrimSpace(mailgunNotify.ApiKey)) == 0 {
		return errors.New("mailgun: invalid api key")
	}

	if len(strings.TrimSpace(mailgunNotify.Domain)) == 0 {
		return errors.New("mailgun: invalid domain name")
	}

	// if len(strings.TrimSpace(mailgunNotify.PublicApiKey)) == 0 {
	// 	return errors.New("Mailgun: Invalid PublicApiKey")
	// }

	mailGunClient = mailgun.NewMailgun(mailgunNotify.Domain, mailgunNotify.ApiKey)

	return nil
}

func (mailgunNotify MailgunNotify) SendResponseTimeNotification(responseTimeNotification ResponseTimeNotification) error {

	subject := "Response Time Notification from StatusOK"
	message := getMessageFromResponseTimeNotification(responseTimeNotification)

	mail := mailGunClient.NewMessage("StatusOkNotifier <notify@StatusOk.com>", subject, message, fmt.Sprintf("<%s>", mailgunNotify.Email))

	ctx, _ := context.WithTimeout(context.Background(), time.Second*30)
	_, _, mailgunErr := mailGunClient.Send(ctx, mail)

	if mailgunErr != nil {
		return mailgunErr
	}

	return nil
}

func (mailgunNotify MailgunNotify) SendErrorNotification(errorNotification ErrorNotification) error {
	subject := "Error Time Notification from StatusOK"

	message := getMessageFromErrorNotification(errorNotification)

	mail := mailGunClient.NewMessage("StatusOkNotifier <notify@StatusOk.com>", subject, message, fmt.Sprintf("<%s>", mailgunNotify.Email))

	ctx, _ := context.WithTimeout(context.Background(), time.Second*30)
	_, _, mailgunErr := mailGunClient.Send(ctx, mail)

	if mailgunErr != nil {
		return mailgunErr
	}

	return nil
}
