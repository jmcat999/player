package storage

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"

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

	driverConfig := mysql.NewConfig()
	driverConfig.User = cfg.DatasourceUsername
	driverConfig.Passwd = cfg.DatasourcePassword
	driverConfig.Net = "tcp"
	driverConfig.Addr = host
	driverConfig.DBName = dbName
	driverConfig.ParseTime = true
	driverConfig.Loc = cfg.Location
	driverConfig.Timeout = 10 * time.Second
	driverConfig.ReadTimeout = 5 * time.Minute
	driverConfig.WriteTimeout = 5 * time.Minute
	driverConfig.Params = map[string]string{
		"charset": "utf8mb4",
	}

	return driverConfig.FormatDSN(), nil
}
