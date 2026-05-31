package storage

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"player-stats-backend-go/internal/config"
)

func Open(ctx context.Context, cfg config.Config) (*sql.DB, error) {
	dsn, err := MySQLDSN(cfg)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func MySQLDSN(cfg config.Config) (string, error) {
	raw := strings.TrimSpace(cfg.DatasourceURL)
	if raw == "" {
		return "", fmt.Errorf("PLAYER_STATS_DATASOURCE_URL is empty")
	}
	if !strings.HasPrefix(raw, "jdbc:mysql://") {
		return raw, nil
	}

	parsed, err := url.Parse(strings.TrimPrefix(raw, "jdbc:"))
	if err != nil {
		return "", err
	}
	host := parsed.Host
	if host == "" {
		return "", fmt.Errorf("mysql host is empty")
	}
	dbName := strings.TrimPrefix(parsed.Path, "/")
	if dbName == "" {
		dbName = "player_stats"
	}

	values := url.Values{}
	values.Set("charset", "utf8mb4,utf8")
	values.Set("parseTime", "true")
	values.Set("loc", cfg.ZoneID)
	values.Set("timeout", "10s")
	values.Set("readTimeout", "5m")
	values.Set("writeTimeout", "5m")

	return fmt.Sprintf("%s:%s@tcp(%s)/%s?%s",
		cfg.DatasourceUsername,
		cfg.DatasourcePassword,
		host,
		dbName,
		values.Encode(),
	), nil
}
