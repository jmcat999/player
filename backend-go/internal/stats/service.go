package stats

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"player-stats-backend-go/internal/apitype"
	"player-stats-backend-go/internal/config"
)

type Service struct {
	db  *sql.DB
	cfg config.Config
}

func NewService(db *sql.DB, cfg config.Config) *Service {
	return &Service{db: db, cfg: cfg}
}

type OverviewResponse struct {
	PlayerCount       int64      `json:"playerCount"`
	BrokenCount       int64      `json:"brokenCount"`
	PlacedCount       int64      `json:"placedCount"`
	TotalCount        int64      `json:"totalCount"`
	ImportedFileCount int64      `json:"importedFileCount"`
	LastImportedAt    *time.Time `json:"lastImportedAt"`
}

type PlayerSummary struct {
	PlayerName  string                 `json:"playerName"`
	BrokenCount int64                  `json:"brokenCount"`
	PlacedCount int64                  `json:"placedCount"`
	TotalCount  int64                  `json:"totalCount"`
	FirstSeenAt *apitype.LocalDateTime `json:"firstSeenAt"`
}

type ServerSummary struct {
	ServerID    string `json:"serverId"`
	ServerName  string `json:"serverName"`
	PlayerCount int64  `json:"playerCount"`
	BrokenCount int64  `json:"brokenCount"`
	PlacedCount int64  `json:"placedCount"`
	TotalCount  int64  `json:"totalCount"`
}

type DailySummary struct {
	StatDate    apitype.Date `json:"statDate"`
	BrokenCount int64        `json:"brokenCount"`
	PlacedCount int64        `json:"placedCount"`
	TotalCount  int64        `json:"totalCount"`
}

type ServerOption struct {
	ServerID   string `json:"serverId"`
	ServerName string `json:"serverName"`
}

type ImportedServerLogFileView struct {
	ID           int64         `json:"id"`
	ServerID     string        `json:"serverId"`
	ServerName   string        `json:"serverName"`
	RemotePath   string        `json:"remotePath"`
	FileName     string        `json:"fileName"`
	LogDate      *apitype.Date `json:"logDate"`
	FileSize     int64         `json:"fileSize"`
	LastModified time.Time     `json:"lastModified"`
	ContentHash  string        `json:"contentHash"`
	ImportedAt   time.Time     `json:"importedAt"`
	RowCount     int           `json:"rowCount"`
	IgnoredCount int           `json:"ignoredCount"`
}

type PlayerStatsResponse struct {
	ServerID         *string                     `json:"serverId"`
	ServerName       string                      `json:"serverName"`
	PlayerName       string                      `json:"playerName"`
	DigCount         int64                       `json:"digCount"`
	BrokenCount      int64                       `json:"brokenCount"`
	PlacedCount      int64                       `json:"placedCount"`
	TotalCount       int64                       `json:"totalCount"`
	FirstSeenAt      *apitype.LocalDateTime      `json:"firstSeenAt"`
	From             *apitype.Date               `json:"from"`
	To               *apitype.Date               `json:"to"`
	LatestLogDate    *apitype.Date               `json:"latestLogDate"`
	LatestImportedAt *time.Time                  `json:"latestImportedAt"`
	Servers          []PlayerServerStatsResponse `json:"servers"`
}

type PlayerServerStatsResponse struct {
	ServerID         string                 `json:"serverId"`
	ServerName       string                 `json:"serverName"`
	PlayerName       string                 `json:"playerName"`
	BrokenCount      int64                  `json:"brokenCount"`
	PlacedCount      int64                  `json:"placedCount"`
	TotalCount       int64                  `json:"totalCount"`
	FirstSeenAt      *apitype.LocalDateTime `json:"firstSeenAt"`
	LatestLogDate    *apitype.Date          `json:"latestLogDate"`
	LatestImportedAt *time.Time             `json:"latestImportedAt"`
}

type dateRange struct {
	From *time.Time
	To   *time.Time
}

func (s *Service) Overview(ctx context.Context, serverID string, from, to *time.Time) (OverviewResponse, error) {
	serverID = normalizeServerID(serverID)
	where, args := dailyWhere(serverID, from, to, "")
	query := `select count(distinct player_name), coalesce(sum(broken_count), 0), coalesce(sum(placed_count), 0) from player_server_daily_stats` + where
	var response OverviewResponse
	if err := s.db.QueryRowContext(ctx, query, args...).Scan(&response.PlayerCount, &response.BrokenCount, &response.PlacedCount); err != nil {
		return response, err
	}
	response.TotalCount = response.BrokenCount + response.PlacedCount
	response.ImportedFileCount = s.importedFileCount(ctx, serverID)
	response.LastImportedAt = s.latestImportedAt(ctx, serverID)
	return response, nil
}

func (s *Service) Players(ctx context.Context, serverID string, from, to *time.Time, player string, limit int) ([]PlayerSummary, error) {
	serverID = normalizeServerID(serverID)
	limit = clamp(limit, 1, 200)
	where, args := dailyWhere(serverID, from, to, strings.TrimSpace(player))
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, `
		select player_name, coalesce(sum(broken_count), 0), coalesce(sum(placed_count), 0)
		from player_server_daily_stats`+where+`
		group by player_name
		order by (coalesce(sum(broken_count), 0) + coalesce(sum(placed_count), 0)) desc
		limit ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []PlayerSummary
	for rows.Next() {
		var item PlayerSummary
		if err := rows.Scan(&item.PlayerName, &item.BrokenCount, &item.PlacedCount); err != nil {
			return nil, err
		}
		item.TotalCount = item.BrokenCount + item.PlacedCount
		item.FirstSeenAt = s.firstSeen(ctx, serverID, item.PlayerName)
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Service) Player(ctx context.Context, serverID, playerName string, from, to *time.Time) (PlayerStatsResponse, bool, error) {
	playerName = strings.TrimSpace(playerName)
	if playerName == "" {
		return PlayerStatsResponse{}, false, nil
	}
	serverID = normalizeServerID(serverID)
	if serverID != "" {
		return s.singleServerPlayer(ctx, serverID, playerName, from, to)
	}
	return s.aggregatePlayer(ctx, playerName, from, to)
}

func (s *Service) Daily(ctx context.Context, serverID string, from, to *time.Time, player string) ([]DailySummary, error) {
	serverID = normalizeServerID(serverID)
	where, args := dailyWhere(serverID, from, to, strings.TrimSpace(player))
	rows, err := s.db.QueryContext(ctx, `
		select stat_date, coalesce(sum(broken_count), 0), coalesce(sum(placed_count), 0)
		from player_server_daily_stats`+where+`
		group by stat_date
		order by stat_date`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []DailySummary
	for rows.Next() {
		var statDate time.Time
		var item DailySummary
		if err := rows.Scan(&statDate, &item.BrokenCount, &item.PlacedCount); err != nil {
			return nil, err
		}
		item.StatDate = apitype.NewDate(statDate)
		item.TotalCount = item.BrokenCount + item.PlacedCount
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Service) Servers(ctx context.Context, from, to *time.Time) ([]ServerSummary, error) {
	where, args := dailyWhere("", from, to, "")
	rows, err := s.db.QueryContext(ctx, `
		select server_id, server_name, count(distinct player_name), coalesce(sum(broken_count), 0), coalesce(sum(placed_count), 0)
		from player_server_daily_stats`+where+`
		group by server_id, server_name
		order by server_id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []ServerSummary
	for rows.Next() {
		var item ServerSummary
		if err := rows.Scan(&item.ServerID, &item.ServerName, &item.PlayerCount, &item.BrokenCount, &item.PlacedCount); err != nil {
			return nil, err
		}
		item.TotalCount = item.BrokenCount + item.PlacedCount
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Service) ServerOptions() []ServerOption {
	sources := s.cfg.Sources()
	result := make([]ServerOption, 0, len(sources))
	for _, source := range sources {
		result = append(result, ServerOption{ServerID: source.ID, ServerName: source.Name})
	}
	return result
}

func (s *Service) ImportedFiles(ctx context.Context, serverID string, limit int) ([]ImportedServerLogFileView, error) {
	serverID = normalizeServerID(serverID)
	limit = clamp(limit, 1, 200)
	query := `
		select id, server_id, server_name, remote_path, file_name, log_date, file_size, last_modified, content_hash, imported_at, row_count, ignored_count
		from imported_server_log_files`
	args := []any{}
	if serverID != "" {
		query += ` where server_id = ?`
		args = append(args, serverID)
	}
	query += ` order by imported_at desc limit ?`
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []ImportedServerLogFileView
	for rows.Next() {
		var item ImportedServerLogFileView
		var logDate sql.NullTime
		if err := rows.Scan(&item.ID, &item.ServerID, &item.ServerName, &item.RemotePath, &item.FileName,
			&logDate, &item.FileSize, &item.LastModified, &item.ContentHash, &item.ImportedAt, &item.RowCount, &item.IgnoredCount); err != nil {
			return nil, err
		}
		if logDate.Valid {
			date := apitype.NewDate(logDate.Time)
			item.LogDate = &date
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Service) singleServerPlayer(ctx context.Context, serverID, playerName string, from, to *time.Time) (PlayerStatsResponse, bool, error) {
	summary, found, err := s.exactPlayerSummary(ctx, serverID, playerName, from, to)
	if err != nil {
		return PlayerStatsResponse{}, false, err
	}
	firstSeen := s.firstSeen(ctx, serverID, playerName)
	if !found && firstSeen == nil {
		return PlayerStatsResponse{}, false, nil
	}
	freshnessDate, freshnessImported := s.dataFreshness(ctx, serverID)
	responseName := summary.PlayerName
	if responseName == "" {
		responseName = s.profilePlayerName(ctx, serverID, playerName)
	}
	if responseName == "" {
		responseName = playerName
	}
	fromDate, toDate := apiDatePtr(from), apiDatePtr(to)
	return PlayerStatsResponse{
		ServerID:         &serverID,
		ServerName:       s.cfg.SourceName(serverID),
		PlayerName:       responseName,
		DigCount:         summary.BrokenCount,
		BrokenCount:      summary.BrokenCount,
		PlacedCount:      summary.PlacedCount,
		TotalCount:       summary.BrokenCount + summary.PlacedCount,
		FirstSeenAt:      firstSeen,
		From:             fromDate,
		To:               toDate,
		LatestLogDate:    freshnessDate,
		LatestImportedAt: freshnessImported,
		Servers:          []PlayerServerStatsResponse{},
	}, true, nil
}

func (s *Service) aggregatePlayer(ctx context.Context, playerName string, from, to *time.Time) (PlayerStatsResponse, bool, error) {
	counts, err := s.exactPlayerSummaryByServer(ctx, playerName, from, to)
	if err != nil {
		return PlayerStatsResponse{}, false, err
	}
	profiles, err := s.profilesByServer(ctx, playerName)
	if err != nil {
		return PlayerStatsResponse{}, false, err
	}
	if len(counts) == 0 && len(profiles) == 0 {
		return PlayerStatsResponse{}, false, nil
	}

	var servers []PlayerServerStatsResponse
	var brokenTotal, placedTotal int64
	var firstSeen *apitype.LocalDateTime
	responseName := playerName
	for _, source := range s.cfg.Sources() {
		summary := counts[source.ID]
		profile := profiles[source.ID]
		serverPlayerName := playerName
		if summary.PlayerName != "" {
			serverPlayerName = summary.PlayerName
			responseName = summary.PlayerName
		} else if profile.PlayerName != "" {
			serverPlayerName = profile.PlayerName
			responseName = profile.PlayerName
		}
		brokenTotal += summary.BrokenCount
		placedTotal += summary.PlacedCount
		if profile.FirstSeenAt != nil && (firstSeen == nil || profile.FirstSeenAt.Before(firstSeen.Time)) {
			value := *profile.FirstSeenAt
			firstSeen = &value
		}
		freshDate, freshImported := s.dataFreshness(ctx, source.ID)
		servers = append(servers, PlayerServerStatsResponse{
			ServerID:         source.ID,
			ServerName:       source.Name,
			PlayerName:       serverPlayerName,
			BrokenCount:      summary.BrokenCount,
			PlacedCount:      summary.PlacedCount,
			TotalCount:       summary.BrokenCount + summary.PlacedCount,
			FirstSeenAt:      profile.FirstSeenAt,
			LatestLogDate:    freshDate,
			LatestImportedAt: freshImported,
		})
	}
	freshDate, freshImported := s.dataFreshness(ctx, "")
	fromDate, toDate := apiDatePtr(from), apiDatePtr(to)
	return PlayerStatsResponse{
		ServerID:         nil,
		ServerName:       "合计",
		PlayerName:       responseName,
		DigCount:         brokenTotal,
		BrokenCount:      brokenTotal,
		PlacedCount:      placedTotal,
		TotalCount:       brokenTotal + placedTotal,
		FirstSeenAt:      firstSeen,
		From:             fromDate,
		To:               toDate,
		LatestLogDate:    freshDate,
		LatestImportedAt: freshImported,
		Servers:          servers,
	}, true, nil
}

func (s *Service) exactPlayerSummary(ctx context.Context, serverID, playerName string, from, to *time.Time) (PlayerSummary, bool, error) {
	where, args := dailyExactWhere(serverID, playerName, from, to)
	var summary PlayerSummary
	err := s.db.QueryRowContext(ctx, `
		select player_name, coalesce(sum(broken_count), 0), coalesce(sum(placed_count), 0)
		from player_server_daily_stats`+where+`
		group by player_name
		limit 1`, args...).Scan(&summary.PlayerName, &summary.BrokenCount, &summary.PlacedCount)
	if err == sql.ErrNoRows {
		return PlayerSummary{}, false, nil
	}
	if err != nil {
		return PlayerSummary{}, false, err
	}
	summary.TotalCount = summary.BrokenCount + summary.PlacedCount
	return summary, true, nil
}

func (s *Service) exactPlayerSummaryByServer(ctx context.Context, playerName string, from, to *time.Time) (map[string]PlayerSummary, error) {
	where, args := dailyExactWhere("", playerName, from, to)
	rows, err := s.db.QueryContext(ctx, `
		select server_id, player_name, coalesce(sum(broken_count), 0), coalesce(sum(placed_count), 0)
		from player_server_daily_stats`+where+`
		group by server_id, server_name, player_name
		order by server_id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[string]PlayerSummary{}
	for rows.Next() {
		var serverID string
		var summary PlayerSummary
		if err := rows.Scan(&serverID, &summary.PlayerName, &summary.BrokenCount, &summary.PlacedCount); err != nil {
			return nil, err
		}
		summary.TotalCount = summary.BrokenCount + summary.PlacedCount
		result[serverID] = summary
	}
	return result, rows.Err()
}

type profileSummary struct {
	PlayerName  string
	FirstSeenAt *apitype.LocalDateTime
}

func (s *Service) profilesByServer(ctx context.Context, playerName string) (map[string]profileSummary, error) {
	rows, err := s.db.QueryContext(ctx, `
		select server_id, player_name, first_seen_at
		from player_server_profiles
		where lower(player_name) = lower(?)
		order by server_id`, playerName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[string]profileSummary{}
	for rows.Next() {
		var serverID string
		var profile profileSummary
		var firstSeen time.Time
		if err := rows.Scan(&serverID, &profile.PlayerName, &firstSeen); err != nil {
			return nil, err
		}
		value := apitype.NewLocalDateTime(firstSeen)
		profile.FirstSeenAt = &value
		result[serverID] = profile
	}
	return result, rows.Err()
}

func (s *Service) firstSeen(ctx context.Context, serverID, playerName string) *apitype.LocalDateTime {
	query := `select min(first_seen_at) from player_server_profiles where lower(player_name) = lower(?)`
	args := []any{playerName}
	if serverID != "" {
		query += ` and server_id = ?`
		args = append(args, serverID)
	}
	var value sql.NullTime
	if err := s.db.QueryRowContext(ctx, query, args...).Scan(&value); err != nil || !value.Valid {
		return nil
	}
	result := apitype.NewLocalDateTime(value.Time)
	return &result
}

func (s *Service) profilePlayerName(ctx context.Context, serverID, playerName string) string {
	var value string
	err := s.db.QueryRowContext(ctx, `
		select player_name
		from player_server_profiles
		where server_id = ? and lower(player_name) = lower(?)
		limit 1`, serverID, playerName).Scan(&value)
	if err != nil {
		return ""
	}
	return value
}

func (s *Service) dataFreshness(ctx context.Context, serverID string) (*apitype.Date, *time.Time) {
	var latestFileDate sql.NullTime
	query := `select max(log_date) from imported_server_log_files`
	args := []any{}
	if serverID != "" {
		query += ` where server_id = ?`
		args = append(args, serverID)
	}
	_ = s.db.QueryRowContext(ctx, query, args...).Scan(&latestFileDate)

	var latestSeen sql.NullTime
	query = `select max(first_seen_at) from player_server_profiles`
	args = []any{}
	if serverID != "" {
		query += ` where server_id = ?`
		args = append(args, serverID)
	}
	_ = s.db.QueryRowContext(ctx, query, args...).Scan(&latestSeen)

	var latestDate *apitype.Date
	if latestFileDate.Valid {
		value := apitype.NewDate(latestFileDate.Time)
		latestDate = &value
	}
	if latestSeen.Valid {
		seenDate := apitype.NewDate(latestSeen.Time)
		if latestDate == nil || seenDate.After(latestDate.Time) {
			latestDate = &seenDate
		}
	}
	return latestDate, s.latestImportedAt(ctx, serverID)
}

func (s *Service) importedFileCount(ctx context.Context, serverID string) int64 {
	query := `select count(*) from imported_server_log_files`
	args := []any{}
	if serverID != "" {
		query += ` where server_id = ?`
		args = append(args, serverID)
	}
	var count int64
	_ = s.db.QueryRowContext(ctx, query, args...).Scan(&count)
	return count
}

func (s *Service) latestImportedAt(ctx context.Context, serverID string) *time.Time {
	query := `select max(imported_at) from imported_server_log_files`
	args := []any{}
	if serverID != "" {
		query += ` where server_id = ?`
		args = append(args, serverID)
	}
	var value sql.NullTime
	if err := s.db.QueryRowContext(ctx, query, args...).Scan(&value); err != nil || !value.Valid {
		return nil
	}
	return &value.Time
}

func dailyWhere(serverID string, from, to *time.Time, player string) (string, []any) {
	clauses := []string{"1 = 1"}
	args := []any{}
	if serverID != "" {
		clauses = append(clauses, "server_id = ?")
		args = append(args, serverID)
	}
	if from != nil {
		clauses = append(clauses, "stat_date >= ?")
		args = append(args, *from)
	}
	if to != nil {
		clauses = append(clauses, "stat_date <= ?")
		args = append(args, *to)
	}
	if player != "" {
		clauses = append(clauses, "lower(player_name) like lower(?)")
		args = append(args, "%"+player+"%")
	}
	return " where " + strings.Join(clauses, " and "), args
}

func dailyExactWhere(serverID, playerName string, from, to *time.Time) (string, []any) {
	where, args := dailyWhere(serverID, from, to, "")
	where += " and lower(player_name) = lower(?)"
	args = append(args, playerName)
	return where, args
}

func normalizeServerID(serverID string) string {
	value := strings.TrimSpace(serverID)
	if value == "" || strings.EqualFold(value, "all") || strings.EqualFold(value, "total") {
		return ""
	}
	return value
}

func apiDatePtr(value *time.Time) *apitype.Date {
	if value == nil {
		return nil
	}
	date := apitype.NewDate(*value)
	return &date
}

func clamp(value, minValue, maxValue int) int {
	return max(minValue, min(value, maxValue))
}

func ParseDate(raw string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	value, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return nil, fmt.Errorf("日期格式应为 YYYY-MM-DD")
	}
	return &value, nil
}
