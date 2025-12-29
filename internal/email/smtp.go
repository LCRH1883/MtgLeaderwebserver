package email

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
)

type Message struct {
	FromName  string
	FromEmail string
	ToEmail   string
	Subject   string
	TextBody  string
}

func SendSMTP(settings SMTPSettings, msg Message) error {
	addr := fmt.Sprintf("%s:%d", settings.Host, settings.Port)
	client, err := smtpConnect(settings, addr)
	if err != nil {
		return err
	}
	defer client.Close()

	if settings.Username != "" {
		auth := smtp.PlainAuth("", settings.Username, settings.Password, settings.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	if err := client.Mail(msg.FromEmail); err != nil {
		return fmt.Errorf("smtp from: %w", err)
	}
	if err := client.Rcpt(msg.ToEmail); err != nil {
		return fmt.Errorf("smtp rcpt: %w", err)
	}

	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}

	from := msg.FromEmail
	if msg.FromName != "" {
		from = fmt.Sprintf("%s <%s>", msg.FromName, msg.FromEmail)
	}
	body := buildMessage(from, msg.ToEmail, msg.Subject, msg.TextBody)
	if _, err := writer.Write([]byte(body)); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("smtp close: %w", err)
	}
	if err := client.Quit(); err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
		return fmt.Errorf("smtp quit: %w", err)
	}
	return nil
}

type SMTPSettings struct {
	Host     string
	Port     int
	Username string
	Password string
	TLSMode  string
}

func smtpConnect(settings SMTPSettings, addr string) (*smtp.Client, error) {
	tlsMode := settings.TLSMode
	if tlsMode == "" {
		tlsMode = "starttls"
	}
	switch tlsMode {
	case "tls":
		conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: settings.Host, MinVersion: tls.VersionTLS12})
		if err != nil {
			return nil, fmt.Errorf("smtp tls dial: %w", err)
		}
		client, err := smtp.NewClient(conn, settings.Host)
		if err != nil {
			return nil, fmt.Errorf("smtp client: %w", err)
		}
		return client, nil
	default:
		client, err := smtp.Dial(addr)
		if err != nil {
			return nil, fmt.Errorf("smtp dial: %w", err)
		}
		if tlsMode == "starttls" {
			if err := client.StartTLS(&tls.Config{ServerName: settings.Host, MinVersion: tls.VersionTLS12}); err != nil {
				_ = client.Close()
				return nil, fmt.Errorf("smtp starttls: %w", err)
			}
		}
		return client, nil
	}
}

func buildMessage(from, to, subject, body string) string {
	lines := []string{
		"From: " + from,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=utf-8",
		"",
		body,
	}
	return strings.Join(lines, "\r\n")
}
