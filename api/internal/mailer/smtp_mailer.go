package mailer

import (
	"fmt"
	"net/smtp"
	"net/url"
	"thing-index/internal/config"
	"thing-index/internal/domain"
)

type smtpMailer struct {
	config *config.Config
}

func NewSMTPMailer(cfg *config.Config) domain.Mailer {
	return &smtpMailer{config: cfg}
}

func (m *smtpMailer) SendResetPasswordEmail(to string, token string) error {
	resetURL, err := url.Parse(m.config.ResetPasswordURL)
	if err != nil {
		return fmt.Errorf("failed to parse reset password url: %w", err)
	}

	query := resetURL.Query()
	query.Set("token", token)
	resetURL.RawQuery = query.Encode()

	subject := "Reset Your Password"
	body := fmt.Sprintf("Click the link to reset your password: %s", resetURL.String())

	msg := []byte("To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"\r\n" +
		body + "\r\n")

	var auth smtp.Auth
	if m.config.SMTPUser != "" || m.config.SMTPPass != "" {
		auth = smtp.PlainAuth("", m.config.SMTPUser, m.config.SMTPPass, m.config.SMTPHost)
	}
	addr := fmt.Sprintf("%s:%s", m.config.SMTPHost, m.config.SMTPPort)

	// Note: In production, consider using a more robust email handling approach (SSL/TLS)
	// or libraries like gomail or mailguy.
	err = smtp.SendMail(addr, auth, m.config.SMTPFrom, []string{to}, msg)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}
