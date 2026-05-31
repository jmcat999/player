package storage

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"

	"player-stats-backend-go/internal/config"
)

func EnsureDefaults(ctx context.Context, db *sql.DB, cfg config.Config) error {
	key, err := randomKey()
	if err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `
		insert into sync_config (
			id, smb_host, smb_port, smb_domain, smb_username, smb_password, smb_share,
			auto_run, sync_cron, skip_today, astrbot_api_key, share_ttl_minutes
		)
		values (1, ?, ?, ?, ?, ?, ?, true, '0 30 0 * * *', ?, ?, 60)
		on duplicate key update id = id
	`, cfg.SMBHost, cfg.SMBPort, cfg.SMBDomain, cfg.SMBUsername, cfg.SMBPassword, cfg.SMBShare, cfg.SkipToday, key); err != nil {
		return err
	}
	for _, source := range cfg.Sources() {
		if _, err := db.ExecContext(ctx, `
			insert into sync_source_config (source_id, source_name, smb_directory, smb_file_glob, smb_recursive, enabled)
			values (?, ?, ?, ?, ?, true)
			on duplicate key update source_id = source_id
		`, source.ID, source.Name, defaultString(cfg.SMBDirectory, ""), defaultString(cfg.SMBFileGlob, source.FileGlob), cfg.SMBRecursive); err != nil {
			return err
		}
	}
	return nil
}

func defaultString(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func randomKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "psk_" + base64.RawURLEncoding.EncodeToString(bytes), nil
}
