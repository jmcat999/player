package share

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"player-stats-backend-go/internal/apitype"
	"player-stats-backend-go/internal/auth"
	"player-stats-backend-go/internal/catalog"
	"player-stats-backend-go/internal/config"
	"player-stats-backend-go/internal/importer"
	"player-stats-backend-go/internal/settings"
	"player-stats-backend-go/internal/stats"
)

type Service struct {
	db                *sql.DB
	cfg               config.Config
	settings          *settings.Service
	stats             *stats.Service
	xray              *importer.XrayAnalysisService
	mu                sync.Mutex
	xraySubs          map[chan XrayGroupMessageResponse]struct{}
	xrayMessages      map[int64]queuedXrayGroupMessage
	nextXrayMessageID int64
}

func NewService(db *sql.DB, cfg config.Config, settingsService *settings.Service, statsService *stats.Service, xrayService *importer.XrayAnalysisService) *Service {
	return &Service{
		db:           db,
		cfg:          cfg,
		settings:     settingsService,
		stats:        statsService,
		xray:         xrayService,
		xraySubs:     map[chan XrayGroupMessageResponse]struct{}{},
		xrayMessages: map[int64]queuedXrayGroupMessage{},
	}
}

const xrayGroupMessageTTL = 10 * time.Minute

type queuedXrayGroupMessage struct {
	message   XrayGroupMessageResponse
	createdAt time.Time
}

type ShareTokenResponse struct {
	Token      string    `json:"token"`
	SharePath  string    `json:"sharePath"`
	ExpiresAt  time.Time `json:"expiresAt"`
	TTLMinutes int       `json:"ttlMinutes"`
}

type PlayerDetailsResponse struct {
	PlayerName    string                  `json:"playerName"`
	LatestLogDate *apitype.Date           `json:"latestLogDate"`
	ExpiresAt     time.Time               `json:"expiresAt"`
	TTLMinutes    int                     `json:"ttlMinutes"`
	Servers       []ServerDetailsResponse `json:"servers"`
}

type ServerDetailsResponse struct {
	ServerID      string                 `json:"serverId"`
	ServerName    string                 `json:"serverName"`
	PlayerName    string                 `json:"playerName"`
	BrokenCount   int64                  `json:"brokenCount"`
	PlacedCount   int64                  `json:"placedCount"`
	TotalCount    int64                  `json:"totalCount"`
	FirstSeenAt   *apitype.LocalDateTime `json:"firstSeenAt"`
	LatestLogDate *apitype.Date          `json:"latestLogDate"`
	Milestones    []MilestoneItem        `json:"milestones"`
	Ores          []OreItem              `json:"ores"`
	Woods         []CountItem            `json:"woods"`
	Saplings      []CountItem            `json:"saplings"`
}

type MilestoneItem struct {
	Type        string                 `json:"type"`
	Label       string                 `json:"label"`
	FoundText   string                 `json:"foundText"`
	MissingText string                 `json:"missingText"`
	FirstSeenAt *apitype.LocalDateTime `json:"firstSeenAt"`
	Detail      string                 `json:"detail"`
}

type OreItem struct {
	Type  string `json:"type"`
	Label string `json:"label"`
	Count int64  `json:"count"`
}

type CountItem struct {
	Type  string `json:"type"`
	Label string `json:"label"`
	Count int64  `json:"count"`
}

type RankingDetailsResponse struct {
	RankingType string                  `json:"rankingType"`
	Limit       int                     `json:"limit"`
	FromDate    *apitype.Date           `json:"fromDate"`
	ToDate      *apitype.Date           `json:"toDate"`
	ExpiresAt   time.Time               `json:"expiresAt"`
	TTLMinutes  int                     `json:"ttlMinutes"`
	Servers     []RankingServerResponse `json:"servers"`
}

type RankingServerResponse struct {
	ServerID   string                `json:"serverId"`
	ServerName string                `json:"serverName"`
	Players    []stats.PlayerSummary `json:"players"`
}

type XrayGroupSendRequest struct {
	ServerID   string                       `json:"serverId"`
	ServerName string                       `json:"serverName"`
	FromTime   string                       `json:"fromTime"`
	ToTime     string                       `json:"toTime"`
	PlayerName string                       `json:"playerName"`
	TTLMinutes *int                         `json:"ttlMinutes"`
	Player     *importer.XrayPlayerRiskView `json:"player"`
}

type XrayGroupSendResponse struct {
	MessageID  int64     `json:"messageId"`
	SharePath  string    `json:"sharePath"`
	Status     string    `json:"status"`
	ServerID   string    `json:"serverId"`
	ServerName string    `json:"serverName"`
	PlayerName string    `json:"playerName"`
	RiskScore  int       `json:"riskScore"`
	RiskLevel  string    `json:"riskLevel"`
	ExpiresAt  time.Time `json:"expiresAt"`
	TTLMinutes int       `json:"ttlMinutes"`
}

type XrayShareDetailsResponse struct {
	ServerID   string                      `json:"serverId"`
	ServerName string                      `json:"serverName"`
	FromTime   *apitype.LocalDateTime      `json:"fromTime"`
	ToTime     *apitype.LocalDateTime      `json:"toTime"`
	PlayerName string                      `json:"playerName"`
	CreatedAt  time.Time                   `json:"createdAt"`
	ExpiresAt  time.Time                   `json:"expiresAt"`
	TTLMinutes int                         `json:"ttlMinutes"`
	Player     importer.XrayPlayerRiskView `json:"player"`
}

type XrayGroupMessageResponse struct {
	ID                         int64                  `json:"id"`
	SharePath                  string                 `json:"sharePath"`
	ServerID                   string                 `json:"serverId"`
	ServerName                 string                 `json:"serverName"`
	PlayerName                 string                 `json:"playerName"`
	RiskScore                  int                    `json:"riskScore"`
	RiskLevel                  string                 `json:"riskLevel"`
	MiningSessionRareOreBreaks int64                  `json:"miningSessionRareOreBreaks"`
	TrackingEvidenceCount      int                    `json:"trackingEvidenceCount"`
	PeakRareOreWindowCount     int                    `json:"peakRareOreWindowCount"`
	FromTime                   *apitype.LocalDateTime `json:"fromTime"`
	ToTime                     *apitype.LocalDateTime `json:"toTime"`
	ExpiresAt                  time.Time              `json:"expiresAt"`
	TTLMinutes                 int                    `json:"ttlMinutes"`
}

type XrayGroupDeliveryRequest struct {
	Success      bool   `json:"success"`
	ErrorMessage string `json:"errorMessage"`
}

type XrayGroupDeliveryResponse struct {
	MessageID int64  `json:"messageId"`
	Status    string `json:"status"`
}

func (s *Service) CreatePlayerToken(ctx context.Context, playerName string) (ShareTokenResponse, error) {
	playerName = strings.TrimSpace(playerName)
	if playerName == "" {
		return ShareTokenResponse{}, auth.NewHTTPError(httpStatusBadRequest, "playerName is required")
	}
	playerStats, found, err := s.stats.Player(ctx, "", playerName, nil, nil)
	if err != nil {
		return ShareTokenResponse{}, err
	}
	if !found {
		return ShareTokenResponse{}, auth.NewHTTPError(httpStatusNotFound, "玩家没有统计数据")
	}
	if err := s.deleteExpiredTokens(ctx); err != nil {
		return ShareTokenResponse{}, err
	}

	now := time.Now().UTC()
	ttl := s.settings.ShareTTLMinutes(ctx)
	token, err := s.generateUniqueToken(ctx, "player_share_tokens")
	if err != nil {
		return ShareTokenResponse{}, err
	}
	expiresAt := now.Add(time.Duration(ttl) * time.Minute)
	_, err = s.db.ExecContext(ctx, `
		insert into player_share_tokens (token, player_name, created_at, expires_at)
		values (?, ?, ?, ?)
	`, token, playerStats.PlayerName, now, expiresAt)
	if err != nil {
		return ShareTokenResponse{}, err
	}
	return ShareTokenResponse{Token: token, SharePath: "/share/" + token, ExpiresAt: expiresAt, TTLMinutes: ttl}, nil
}

func (s *Service) PlayerDetails(ctx context.Context, token string) (PlayerDetailsResponse, error) {
	tokenRow, err := s.playerToken(ctx, token)
	if err != nil {
		return PlayerDetailsResponse{}, err
	}
	playerStats, found, err := s.stats.Player(ctx, "", tokenRow.playerName, nil, nil)
	if err != nil {
		return PlayerDetailsResponse{}, err
	}
	if !found {
		return PlayerDetailsResponse{}, auth.NewHTTPError(httpStatusNotFound, "玩家没有统计数据")
	}

	servers := make([]ServerDetailsResponse, 0, len(playerStats.Servers))
	for _, serverStats := range playerStats.Servers {
		details, err := s.serverDetails(ctx, serverStats, tokenRow.playerName)
		if err != nil {
			return PlayerDetailsResponse{}, err
		}
		servers = append(servers, details)
	}

	return PlayerDetailsResponse{
		PlayerName:    playerStats.PlayerName,
		LatestLogDate: playerStats.LatestLogDate,
		ExpiresAt:     tokenRow.expiresAt,
		TTLMinutes:    ttlMinutes(tokenRow.createdAt, tokenRow.expiresAt),
		Servers:       servers,
	}, nil
}

func (s *Service) CreateRankingToken(ctx context.Context, rankingType string, limit int, from, to *time.Time) (ShareTokenResponse, error) {
	normalizedType, err := normalizeRankingType(rankingType)
	if err != nil {
		return ShareTokenResponse{}, err
	}
	limit = max(1, min(limit, 20))
	if err := s.deleteExpiredTokens(ctx); err != nil {
		return ShareTokenResponse{}, err
	}

	now := time.Now().UTC()
	ttl := s.settings.ShareTTLMinutes(ctx)
	token, err := s.generateUniqueToken(ctx, "ranking_share_tokens")
	if err != nil {
		return ShareTokenResponse{}, err
	}
	expiresAt := now.Add(time.Duration(ttl) * time.Minute)
	_, err = s.db.ExecContext(ctx, `
		insert into ranking_share_tokens (token, ranking_type, limit_count, from_date, to_date, created_at, expires_at)
		values (?, ?, ?, ?, ?, ?, ?)
	`, token, normalizedType, limit, dateArg(from), dateArg(to), now, expiresAt)
	if err != nil {
		return ShareTokenResponse{}, err
	}
	return ShareTokenResponse{Token: token, SharePath: "/share/ranking/" + token, ExpiresAt: expiresAt, TTLMinutes: ttl}, nil
}

func (s *Service) RankingDetails(ctx context.Context, token string) (RankingDetailsResponse, error) {
	tokenRow, err := s.rankingToken(ctx, token)
	if err != nil {
		return RankingDetailsResponse{}, err
	}
	servers := make([]RankingServerResponse, 0)
	for _, option := range s.stats.ServerOptions() {
		players, err := s.stats.Players(ctx, option.ServerID, tokenRow.fromDate, tokenRow.toDate, "", tokenRow.limit)
		if err != nil {
			return RankingDetailsResponse{}, err
		}
		servers = append(servers, RankingServerResponse{
			ServerID:   option.ServerID,
			ServerName: option.ServerName,
			Players:    players,
		})
	}
	return RankingDetailsResponse{
		RankingType: tokenRow.rankingType,
		Limit:       tokenRow.limit,
		FromDate:    apiDatePtr(tokenRow.fromDate),
		ToDate:      apiDatePtr(tokenRow.toDate),
		ExpiresAt:   tokenRow.expiresAt,
		TTLMinutes:  ttlMinutes(tokenRow.createdAt, tokenRow.expiresAt),
		Servers:     servers,
	}, nil
}

func (s *Service) SendXrayToGroup(ctx context.Context, request XrayGroupSendRequest) (XrayGroupSendResponse, error) {
	snapshot, err := s.xraySnapshot(ctx, request)
	if err != nil {
		return XrayGroupSendResponse{}, err
	}
	if err := s.deleteExpiredTokens(ctx); err != nil {
		return XrayGroupSendResponse{}, err
	}
	now := time.Now().UTC()
	ttl := normalizeXrayTTL(request.TTLMinutes)
	expiresAt := now.Add(time.Duration(ttl) * time.Minute)
	token, err := s.generateUniqueToken(ctx, "xray_share_tokens")
	if err != nil {
		return XrayGroupSendResponse{}, err
	}
	sharePath := "/xray-share/" + token
	details := XrayShareDetailsResponse{
		ServerID:   snapshot.serverID,
		ServerName: snapshot.serverName,
		FromTime:   localDateTimePtr(&snapshot.fromTime),
		ToTime:     localDateTimePtr(&snapshot.toTime),
		PlayerName: snapshot.player.PlayerName,
		CreatedAt:  now,
		ExpiresAt:  expiresAt,
		TTLMinutes: ttl,
		Player:     snapshot.player,
	}
	payload, err := json.Marshal(details)
	if err != nil {
		return XrayGroupSendResponse{}, err
	}
	_, err = s.db.ExecContext(ctx, `
		insert into xray_share_tokens (token, server_id, player_name, created_at, expires_at, payload_json)
		values (?, ?, ?, ?, ?, ?)
	`, token, snapshot.serverID, snapshot.player.PlayerName, now, expiresAt, string(payload))
	if err != nil {
		return XrayGroupSendResponse{}, err
	}
	messageID := s.nextXrayGroupMessageID()
	response := XrayGroupSendResponse{
		MessageID:  messageID,
		SharePath:  sharePath,
		Status:     "PENDING",
		ServerID:   snapshot.serverID,
		ServerName: snapshot.serverName,
		PlayerName: snapshot.player.PlayerName,
		RiskScore:  snapshot.player.RiskScore,
		RiskLevel:  snapshot.player.RiskLevel,
		ExpiresAt:  expiresAt,
		TTLMinutes: ttl,
	}
	s.enqueueXrayGroupMessage(XrayGroupMessageResponse{
		ID:                         messageID,
		SharePath:                  sharePath,
		ServerID:                   response.ServerID,
		ServerName:                 response.ServerName,
		PlayerName:                 response.PlayerName,
		RiskScore:                  response.RiskScore,
		RiskLevel:                  response.RiskLevel,
		MiningSessionRareOreBreaks: snapshot.player.MiningSessionRareOreBreaks,
		TrackingEvidenceCount:      snapshot.player.TrackingEvidenceCount,
		PeakRareOreWindowCount:     snapshot.player.PeakRareOreWindowCount,
		FromTime:                   localDateTimePtr(&snapshot.fromTime),
		ToTime:                     localDateTimePtr(&snapshot.toTime),
		ExpiresAt:                  expiresAt,
		TTLMinutes:                 ttl,
	})
	return response, nil
}

func (s *Service) SubscribeXrayGroupMessages(buffer int) (<-chan XrayGroupMessageResponse, func()) {
	if buffer < 1 {
		buffer = 1
	}
	ch := make(chan XrayGroupMessageResponse, buffer)
	s.mu.Lock()
	s.xraySubs[ch] = struct{}{}
	s.mu.Unlock()
	unsubscribe := func() {
		s.mu.Lock()
		if _, ok := s.xraySubs[ch]; ok {
			delete(s.xraySubs, ch)
			close(ch)
		}
		s.mu.Unlock()
	}
	return ch, unsubscribe
}

func (s *Service) nextXrayGroupMessageID() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextXrayMessageID++
	return s.nextXrayMessageID
}

func (s *Service) enqueueXrayGroupMessage(message XrayGroupMessageResponse) {
	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneExpiredXrayMessagesLocked(now)
	s.xrayMessages[message.ID] = queuedXrayGroupMessage{message: message, createdAt: now}
	for ch := range s.xraySubs {
		select {
		case ch <- message:
		default:
		}
	}
}

func (s *Service) pendingXrayGroupMessages(limit int) []XrayGroupMessageResponse {
	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneExpiredXrayMessagesLocked(now)

	items := make([]queuedXrayGroupMessage, 0, len(s.xrayMessages))
	for _, item := range s.xrayMessages {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].createdAt.Equal(items[j].createdAt) {
			return items[i].message.ID < items[j].message.ID
		}
		return items[i].createdAt.Before(items[j].createdAt)
	})

	result := make([]XrayGroupMessageResponse, 0, min(limit, len(items)))
	for _, item := range items {
		if len(result) >= limit {
			break
		}
		result = append(result, item.message)
	}
	return result
}

func (s *Service) removeXrayGroupMessage(messageID int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneExpiredXrayMessagesLocked(time.Now().UTC())
	if _, ok := s.xrayMessages[messageID]; !ok {
		return false
	}
	delete(s.xrayMessages, messageID)
	return true
}

func (s *Service) pruneExpiredXrayMessagesLocked(now time.Time) {
	for messageID, item := range s.xrayMessages {
		if !item.createdAt.Add(xrayGroupMessageTTL).After(now) {
			delete(s.xrayMessages, messageID)
		}
	}
}

func (s *Service) XrayDetails(ctx context.Context, token string) (XrayShareDetailsResponse, error) {
	token = strings.TrimSpace(token)
	var payload string
	var expiresAt time.Time
	err := s.db.QueryRowContext(ctx, `select payload_json, expires_at from xray_share_tokens where token = ?`, token).Scan(&payload, &expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return XrayShareDetailsResponse{}, auth.NewHTTPError(httpStatusNotFound, "链接不存在")
	}
	if err != nil {
		return XrayShareDetailsResponse{}, err
	}
	if !expiresAt.After(time.Now().UTC()) {
		return XrayShareDetailsResponse{}, auth.NewHTTPError(httpStatusGone, "链接已过期，请重新生成")
	}
	var details XrayShareDetailsResponse
	if err := json.Unmarshal([]byte(payload), &details); err != nil {
		return XrayShareDetailsResponse{}, err
	}
	return details, nil
}

func (s *Service) PendingXrayGroupMessages(ctx context.Context, limit int) ([]XrayGroupMessageResponse, error) {
	limit = max(1, min(limit, 10))
	return s.pendingXrayGroupMessages(limit), nil
}

func (s *Service) MarkXrayGroupDelivery(ctx context.Context, messageID int64, success bool, errorMessage string) (XrayGroupDeliveryResponse, error) {
	status := "FAILED"
	if success {
		status = "SENT"
	}
	if !s.removeXrayGroupMessage(messageID) {
		return XrayGroupDeliveryResponse{}, auth.NewHTTPError(httpStatusNotFound, "发送任务不存在或已过期")
	}
	return XrayGroupDeliveryResponse{MessageID: messageID, Status: status}, nil
}

type xrayShareSnapshot struct {
	serverID   string
	serverName string
	fromTime   time.Time
	toTime     time.Time
	player     importer.XrayPlayerRiskView
}

func (s *Service) xraySnapshot(ctx context.Context, request XrayGroupSendRequest) (xrayShareSnapshot, error) {
	serverID := strings.TrimSpace(request.ServerID)
	if serverID == "" {
		serverID = "main"
	}
	playerName := strings.TrimSpace(request.PlayerName)
	if playerName == "" && request.Player != nil {
		playerName = strings.TrimSpace(request.Player.PlayerName)
	}
	if playerName == "" {
		return xrayShareSnapshot{}, auth.NewHTTPError(httpStatusBadRequest, "playerName is required")
	}

	fromTime, ok := parseShareLocalDateTime(request.FromTime, s.cfg.Location)
	if !ok {
		fromTime = time.Now().In(s.cfg.Location)
	}
	toTime, ok := parseShareLocalDateTime(request.ToTime, s.cfg.Location)
	if !ok {
		toTime = fromTime
	}
	serverName := strings.TrimSpace(request.ServerName)
	if serverName == "" {
		serverName = s.cfg.SourceName(serverID)
	}

	if request.Player != nil {
		player := *request.Player
		if strings.TrimSpace(player.PlayerName) == "" {
			player.PlayerName = playerName
		}
		return xrayShareSnapshot{
			serverID:   serverID,
			serverName: serverName,
			fromTime:   fromTime,
			toTime:     toTime,
			player:     player,
		}, nil
	}

	if s.xray == nil {
		return xrayShareSnapshot{}, auth.NewHTTPError(httpStatusBadRequest, "矿透分析服务未初始化")
	}
	analysis, err := s.xray.Latest(ctx, serverID)
	if err != nil {
		return xrayShareSnapshot{}, err
	}
	if analysis.Status != "FINISHED" && analysis.Status != "FINISHED_WITH_ERRORS" {
		return xrayShareSnapshot{}, auth.NewHTTPError(httpStatusBadRequest, "请先完成矿透分析后再发送")
	}
	if analysis.FromTime != nil {
		fromTime = analysis.FromTime.Time
	}
	if analysis.ToTime != nil {
		toTime = analysis.ToTime.Time
	}
	if strings.TrimSpace(analysis.ServerName) != "" {
		serverName = analysis.ServerName
	}
	for _, player := range analysis.Players {
		if strings.EqualFold(player.PlayerName, playerName) {
			return xrayShareSnapshot{
				serverID:   analysis.ServerID,
				serverName: serverName,
				fromTime:   fromTime,
				toTime:     toTime,
				player:     player,
			}, nil
		}
	}
	return xrayShareSnapshot{}, auth.NewHTTPError(httpStatusNotFound, "找不到该玩家的矿透分析结果")
}

func (s *Service) serverDetails(ctx context.Context, server stats.PlayerServerStatsResponse, playerName string) (ServerDetailsResponse, error) {
	milestones, err := s.milestoneItems(ctx, server.ServerID, playerName)
	if err != nil {
		return ServerDetailsResponse{}, err
	}
	ores, err := s.oreItems(ctx, server.ServerID, playerName)
	if err != nil {
		return ServerDetailsResponse{}, err
	}
	woods, err := s.countItems(ctx, server.ServerID, playerName, "player_server_log_file_wood_stats", "wood_type", "wood_count", catalog.WoodTypes)
	if err != nil {
		return ServerDetailsResponse{}, err
	}
	saplings, err := s.countItems(ctx, server.ServerID, playerName, "player_server_log_file_sapling_stats", "sapling_type", "sapling_count", catalog.SaplingTypes)
	if err != nil {
		return ServerDetailsResponse{}, err
	}
	return ServerDetailsResponse{
		ServerID:      server.ServerID,
		ServerName:    server.ServerName,
		PlayerName:    server.PlayerName,
		BrokenCount:   server.BrokenCount,
		PlacedCount:   server.PlacedCount,
		TotalCount:    server.TotalCount,
		FirstSeenAt:   server.FirstSeenAt,
		LatestLogDate: server.LatestLogDate,
		Milestones:    milestones,
		Ores:          ores,
		Woods:         woods,
		Saplings:      saplings,
	}, nil
}

func (s *Service) milestoneItems(ctx context.Context, serverID, playerName string) ([]MilestoneItem, error) {
	rows, err := s.db.QueryContext(ctx, `
		select m.milestone_type, m.first_seen_at, coalesce(m.detail, '')
		from player_server_log_file_milestones m
		join imported_server_log_files f on f.id = m.import_file_id
		where f.server_id = ? and lower(m.player_name) = lower(?)
		order by m.first_seen_at
	`, serverID, playerName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type value struct {
		firstSeenAt apitype.LocalDateTime
		detail      string
	}
	firstByType := map[string]value{}
	for rows.Next() {
		var milestoneType, detail string
		var firstSeen time.Time
		if err := rows.Scan(&milestoneType, &firstSeen, &detail); err != nil {
			return nil, err
		}
		if _, exists := firstByType[milestoneType]; exists {
			continue
		}
		firstByType[milestoneType] = value{firstSeenAt: apitype.NewLocalDateTime(firstSeen), detail: detail}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := make([]MilestoneItem, 0, len(catalog.MilestoneTypes))
	for _, typ := range catalog.MilestoneTypes {
		item := MilestoneItem{
			Type:        typ.Type,
			Label:       typ.Label,
			FoundText:   typ.FoundText,
			MissingText: typ.MissingText,
		}
		if seen, ok := firstByType[typ.Type]; ok {
			value := seen.firstSeenAt
			item.FirstSeenAt = &value
			item.Detail = seen.detail
		}
		result = append(result, item)
	}
	return result, nil
}

func (s *Service) oreItems(ctx context.Context, serverID, playerName string) ([]OreItem, error) {
	rows, err := s.db.QueryContext(ctx, `
		select o.ore_type, coalesce(sum(o.ore_count), 0)
		from player_server_log_file_ore_stats o
		join imported_server_log_files f on f.id = o.import_file_id
		where f.server_id = ? and lower(o.player_name) = lower(?)
		group by o.ore_type
	`, serverID, playerName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	countByType := map[string]int64{}
	for rows.Next() {
		var oreType string
		var count int64
		if err := rows.Scan(&oreType, &count); err != nil {
			return nil, err
		}
		countByType[oreType] = count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	result := make([]OreItem, 0, len(catalog.OreTypes))
	for _, typ := range catalog.OreTypes {
		result = append(result, OreItem{Type: typ.Type, Label: typ.Label, Count: countByType[typ.Type]})
	}
	return result, nil
}

func (s *Service) countItems(ctx context.Context, serverID, playerName, table, typeColumn, countColumn string, types []catalog.BlockType) ([]CountItem, error) {
	query := `
		select t.` + typeColumn + `, coalesce(sum(t.` + countColumn + `), 0)
		from ` + table + ` t
		join imported_server_log_files f on f.id = t.import_file_id
		where f.server_id = ? and lower(t.player_name) = lower(?)
		group by t.` + typeColumn
	rows, err := s.db.QueryContext(ctx, query, serverID, playerName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	countByType := map[string]int64{}
	for rows.Next() {
		var typ string
		var count int64
		if err := rows.Scan(&typ, &count); err != nil {
			return nil, err
		}
		countByType[typ] = count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	result := make([]CountItem, 0, len(types))
	for _, typ := range types {
		result = append(result, CountItem{Type: typ.Type, Label: typ.Label, Count: countByType[typ.Type]})
	}
	return result, nil
}

type playerTokenRow struct {
	playerName string
	createdAt  time.Time
	expiresAt  time.Time
}

func (s *Service) playerToken(ctx context.Context, token string) (playerTokenRow, error) {
	token = strings.TrimSpace(token)
	var row playerTokenRow
	err := s.db.QueryRowContext(ctx, `
		select player_name, created_at, expires_at
		from player_share_tokens
		where token = ?
	`, token).Scan(&row.playerName, &row.createdAt, &row.expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return playerTokenRow{}, auth.NewHTTPError(httpStatusNotFound, "链接不存在")
	}
	if err != nil {
		return playerTokenRow{}, err
	}
	if !row.expiresAt.After(time.Now().UTC()) {
		return playerTokenRow{}, auth.NewHTTPError(httpStatusGone, "链接已过期，请重新在群里查询")
	}
	return row, nil
}

type rankingTokenRow struct {
	rankingType string
	limit       int
	fromDate    *time.Time
	toDate      *time.Time
	createdAt   time.Time
	expiresAt   time.Time
}

func (s *Service) rankingToken(ctx context.Context, token string) (rankingTokenRow, error) {
	token = strings.TrimSpace(token)
	var row rankingTokenRow
	var fromDate, toDate sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		select ranking_type, limit_count, from_date, to_date, created_at, expires_at
		from ranking_share_tokens
		where token = ?
	`, token).Scan(&row.rankingType, &row.limit, &fromDate, &toDate, &row.createdAt, &row.expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return rankingTokenRow{}, auth.NewHTTPError(httpStatusNotFound, "链接不存在")
	}
	if err != nil {
		return rankingTokenRow{}, err
	}
	if !row.expiresAt.After(time.Now().UTC()) {
		return rankingTokenRow{}, auth.NewHTTPError(httpStatusGone, "链接已过期，请重新在群里查询")
	}
	if fromDate.Valid {
		row.fromDate = &fromDate.Time
	}
	if toDate.Valid {
		row.toDate = &toDate.Time
	}
	return row, nil
}

func (s *Service) deleteExpiredTokens(ctx context.Context) error {
	now := time.Now().UTC()
	if _, err := s.db.ExecContext(ctx, `delete from player_share_tokens where expires_at < ?`, now); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `delete from ranking_share_tokens where expires_at < ?`, now); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `delete from xray_share_tokens where expires_at < ?`, now); err != nil {
		return err
	}
	return nil
}

func (s *Service) generateUniqueToken(ctx context.Context, table string) (string, error) {
	for attempt := 0; attempt < 5; attempt++ {
		token, err := randomToken(24)
		if err != nil {
			return "", err
		}
		var exists int
		if err := s.db.QueryRowContext(ctx, `select count(*) from `+table+` where token = ?`, token).Scan(&exists); err != nil {
			return "", err
		}
		if exists == 0 {
			return token, nil
		}
	}
	return "", errors.New("failed to generate share token")
}

func randomToken(size int) (string, error) {
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func normalizeRankingType(value string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		normalized = "total"
	}
	if normalized != "total" && normalized != "active" {
		return "", auth.NewHTTPError(httpStatusBadRequest, "rankingType must be 'total' or 'active'")
	}
	return normalized, nil
}

func ttlMinutes(createdAt, expiresAt time.Time) int {
	minutes := int(expiresAt.Sub(createdAt).Minutes())
	if minutes < 1 {
		return 1
	}
	return minutes
}

func dateArg(value *time.Time) any {
	if value == nil {
		return nil
	}
	return *value
}

func apiDatePtr(value *time.Time) *apitype.Date {
	if value == nil {
		return nil
	}
	date := apitype.NewDate(*value)
	return &date
}

func localDateTimePtr(value *time.Time) *apitype.LocalDateTime {
	if value == nil {
		return nil
	}
	local := apitype.NewLocalDateTime(*value)
	return &local
}

func normalizeXrayTTL(value *int) int {
	if value == nil {
		return 1440
	}
	ttl := *value
	if ttl < 5 {
		return 5
	}
	if ttl > 10080 {
		return 10080
	}
	return ttl
}

func parseShareLocalDateTime(raw string, location *time.Location) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}
	if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
		return parsed.In(location), true
	}
	layouts := []string{
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04",
		"2006-01-02 15:04",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if parsed, err := time.ParseInLocation(layout, raw, location); err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

func truncate(value string, maxLength int) string {
	value = strings.TrimSpace(value)
	if maxLength <= 0 || len(value) <= maxLength {
		return value
	}
	return value[:maxLength]
}

const (
	httpStatusBadRequest = 400
	httpStatusNotFound   = 404
	httpStatusGone       = 410
)
