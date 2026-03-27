package mailer

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strings"
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
	resetURL, err := m.config.ResetPasswordURLParsed()
	if err != nil {
		return fmt.Errorf("failed to parse reset password url: %w", err)
	}

	query := resetURL.Query()
	query.Set("token", token)
	resetURL.RawQuery = query.Encode()

	subject := "Reset Your Password"
	body := fmt.Sprintf("Click the link to reset your password: %s", resetURL.String())

	msg := []byte("From: " + m.config.SMTPFrom + "\r\n" +
		"To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"\r\n" +
		body + "\r\n")

	client, err := m.newClient()
	if err != nil {
		return err
	}
	defer client.Close()

	if err := sendSMTPMessage(client, m.config.SMTPFrom, to, msg); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	if err := client.Quit(); err != nil {
		return fmt.Errorf("failed to close smtp session: %w", err)
	}

	return nil
}

func (m *smtpMailer) newClient() (*smtp.Client, error) {
	addr := net.JoinHostPort(m.config.SMTPHost, m.config.SMTPPort)
	mode := m.config.NormalizedSMTPSecurityMode()

	var (
		client *smtp.Client
		err    error
	)

	switch mode {
	case "tls":
		client, err = m.newTLSClient(addr)
	case "starttls":
		client, err = m.newPlainClient(addr)
		if err == nil {
			err = m.upgradeClientToTLS(client)
		}
	case "none":
		client, err = m.newPlainClient(addr)
	default:
		return nil, fmt.Errorf("unsupported smtp security mode: %s", mode)
	}
	if err != nil {
		return nil, err
	}

	if err := m.authenticateClient(client); err != nil {
		client.Close()
		return nil, err
	}

	return client, nil
}

func (m *smtpMailer) newPlainClient(addr string) (*smtp.Client, error) {
	client, err := smtp.Dial(addr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to smtp server: %w", err)
	}

	return client, nil
}

func (m *smtpMailer) newTLSClient(addr string) (*smtp.Client, error) {
	conn, err := tls.Dial("tcp", addr, m.tlsConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to establish tls smtp connection: %w", err)
	}

	client, err := smtp.NewClient(conn, m.config.SMTPHost)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to initialize tls smtp client: %w", err)
	}

	return client, nil
}

func (m *smtpMailer) upgradeClientToTLS(client *smtp.Client) error {
	ok, _ := client.Extension("STARTTLS")
	if !ok {
		return errors.New("smtp server does not support STARTTLS")
	}

	if err := client.StartTLS(m.tlsConfig()); err != nil {
		return fmt.Errorf("failed to upgrade smtp connection with STARTTLS: %w", err)
	}

	return nil
}

func (m *smtpMailer) authenticateClient(client *smtp.Client) error {
	if strings.TrimSpace(m.config.SMTPUser) == "" && strings.TrimSpace(m.config.SMTPPass) == "" {
		return nil
	}

	ok, _ := client.Extension("AUTH")
	if !ok {
		return errors.New("smtp server does not support authentication")
	}

	auth := smtp.PlainAuth("", m.config.SMTPUser, m.config.SMTPPass, m.config.SMTPHost)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("failed to authenticate with smtp server: %w", err)
	}

	return nil
}

func (m *smtpMailer) tlsConfig() *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		ServerName: m.config.SMTPHost,
	}
}

func sendSMTPMessage(client *smtp.Client, from, to string, message []byte) error {
	if err := client.Mail(from); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}

	writer, err := client.Data()
	if err != nil {
		return err
	}

	if _, err := writer.Write(message); err != nil {
		writer.Close()
		return err
	}

	return writer.Close()
}
