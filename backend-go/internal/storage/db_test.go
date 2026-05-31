package storage

import (
	"strings"
	"testing"
	"time"

	"player-stats-backend-go/internal/config"
)

func TestMySQLDSNFromJDBCUsesValidCharset(t *testing.T) {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	dsn, err := MySQLDSN(config.Config{
		DatasourceURL:      "jdbc:mysql://mysql:3306/player_stats?useUnicode=true&characterEncoding=utf8&serverTimezone=Asia/Shanghai",
		DatasourceUsername: "player_stats",
		DatasourcePassword: "mysql123ll",
		Location:           location,
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(dsn, "%2C") {
		t.Fatalf("dsn must not encode comma into charset: %s", dsn)
	}
	if !strings.Contains(dsn, "charset=utf8mb4") {
		t.Fatalf("dsn should force utf8mb4 charset: %s", dsn)
	}
}
