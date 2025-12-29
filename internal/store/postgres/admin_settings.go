package postgres

import (
	"context"
	"errors"
	"fmt"

	"MtgLeaderwebserver/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AdminSettingsStore struct {
	pool *pgxpool.Pool
}

func NewAdminSettingsStore(pool *pgxpool.Pool) *AdminSettingsStore {
	return &AdminSettingsStore{pool: pool}
}

func (s *AdminSettingsStore) GetSMTPSettings(ctx context.Context) (domain.SMTPSettings, bool, error) {
	const q = `
		SELECT host, port, username, password, tls_mode, from_name, from_email, alias_emails, created_at, updated_at
		FROM smtp_settings
		WHERE id = 1
	`

	var (
		settings domain.SMTPSettings
		aliases  pgtype.FlatArray[string]
	)
	err := s.pool.QueryRow(ctx, q).Scan(
		&settings.Host,
		&settings.Port,
		&settings.Username,
		&settings.Password,
		&settings.TLSMode,
		&settings.FromName,
		&settings.FromEmail,
		&aliases,
		&settings.CreatedAt,
		&settings.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.SMTPSettings{}, false, nil
		}
		return domain.SMTPSettings{}, false, fmt.Errorf("get smtp settings: %w", err)
	}
	settings.AliasEmails = textArrayOrEmpty(aliases)
	return settings, true, nil
}

func (s *AdminSettingsStore) UpsertSMTPSettings(ctx context.Context, settings domain.SMTPSettings) error {
	const q = `
		INSERT INTO smtp_settings (
			id, host, port, username, password, tls_mode, from_name, from_email, alias_emails, created_at, updated_at
		)
		VALUES (1, $1, $2, $3, $4, $5, $6, $7, $8, now(), now())
		ON CONFLICT (id)
		DO UPDATE SET
			host = EXCLUDED.host,
			port = EXCLUDED.port,
			username = EXCLUDED.username,
			password = EXCLUDED.password,
			tls_mode = EXCLUDED.tls_mode,
			from_name = EXCLUDED.from_name,
			from_email = EXCLUDED.from_email,
			alias_emails = EXCLUDED.alias_emails,
			updated_at = now()
	`

	aliases := settings.AliasEmails
	if aliases == nil {
		aliases = []string{}
	}
	_, err := s.pool.Exec(ctx, q,
		settings.Host,
		settings.Port,
		settings.Username,
		settings.Password,
		settings.TLSMode,
		settings.FromName,
		settings.FromEmail,
		pgtype.FlatArray[string](aliases),
	)
	if err != nil {
		return fmt.Errorf("upsert smtp settings: %w", err)
	}
	return nil
}
