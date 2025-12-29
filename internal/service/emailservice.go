package service

import (
	"context"
	"fmt"
	"strings"

	"MtgLeaderwebserver/internal/domain"
	"MtgLeaderwebserver/internal/email"
)

type SMTPSettingsStore interface {
	GetSMTPSettings(ctx context.Context) (domain.SMTPSettings, bool, error)
	UpsertSMTPSettings(ctx context.Context, settings domain.SMTPSettings) error
}

type EmailService struct {
	Settings SMTPSettingsStore
}

func (s *EmailService) GetSMTPSettings(ctx context.Context) (domain.SMTPSettings, bool, error) {
	if s.Settings == nil {
		return domain.SMTPSettings{}, false, fmt.Errorf("smtp settings unavailable")
	}
	return s.Settings.GetSMTPSettings(ctx)
}

func (s *EmailService) SaveSMTPSettings(ctx context.Context, settings domain.SMTPSettings) error {
	if s.Settings == nil {
		return fmt.Errorf("smtp settings unavailable")
	}
	return s.Settings.UpsertSMTPSettings(ctx, settings)
}

func (s *EmailService) SendPasswordReset(ctx context.Context, fromEmail, toEmail, resetURL string) error {
	if s.Settings == nil {
		return fmt.Errorf("smtp settings unavailable")
	}
	settings, ok, err := s.Settings.GetSMTPSettings(ctx)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("smtp settings not configured")
	}

	subject := "Reset your MTG Friends password"
	body := strings.Join([]string{
		"You requested a password reset.",
		"",
		"Reset your password using this link:",
		resetURL,
		"",
		"If you did not request this, you can ignore this email.",
	}, "\n")

	return email.SendSMTP(email.SMTPSettings{
		Host:     settings.Host,
		Port:     settings.Port,
		Username: settings.Username,
		Password: settings.Password,
		TLSMode:  settings.TLSMode,
	}, email.Message{
		FromName:  settings.FromName,
		FromEmail: fromEmail,
		ToEmail:   toEmail,
		Subject:   subject,
		TextBody:  body,
	})
}

func (s *EmailService) SendTestEmail(ctx context.Context, settings domain.SMTPSettings, toEmail string) error {
	if s.Settings == nil {
		return fmt.Errorf("smtp settings unavailable")
	}
	toEmail = strings.TrimSpace(strings.ToLower(toEmail))
	if toEmail == "" {
		return fmt.Errorf("test email required")
	}
	subject := "MTG Leader SMTP test"
	body := strings.Join([]string{
		"This is a test email from MTG Leader.",
		"",
		"If you received this, your SMTP settings are working.",
	}, "\n")

	return email.SendSMTP(email.SMTPSettings{
		Host:     settings.Host,
		Port:     settings.Port,
		Username: settings.Username,
		Password: settings.Password,
		TLSMode:  settings.TLSMode,
	}, email.Message{
		FromName:  settings.FromName,
		FromEmail: settings.FromEmail,
		ToEmail:   toEmail,
		Subject:   subject,
		TextBody:  body,
	})
}
