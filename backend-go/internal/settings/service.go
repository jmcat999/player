package settings

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"io/fs"
	"path/filepath"
	"strings"

	"player-stats-backend-go/internal/config"
)

const AstrBotAPIKeyHeader = "X-Player-Stats-Key"

type Service struct {
	db  *sql.DB
	cfg config.Config
}

func NewService(db *sql.DB, cfg config.Config) *Service {
	return &Service{db: db, cfg: cfg}
}

type SyncConfigResponse struct {
	SMBHost         string            `json:"smbHost"`
	SMBPort         int               `json:"smbPort"`
	SMBDomain       string            `json:"smbDomain"`
	SMBUsername     string            `json:"smbUsername"`
	SMBPassword     string            `json:"smbPassword"`
	SMBShare        string            `json:"smbShare"`
	AutoRun         bool              `json:"autoRun"`
	SyncCron        string            `json:"syncCron"`
	SkipToday       bool              `json:"skipToday"`
	ShareTTLMinutes int               `json:"shareTtlMinutes"`
	Sources         []SyncSource      `json:"sources"`
	AutoTasks       []AutoTaskSetting `json:"autoTasks"`
}

type SyncSource struct {
	ID           *int64 `json:"id"`
	SourceID     string `json:"sourceId"`
	SourceName   string `json:"sourceName"`
	SMBDirectory string `json:"smbDirectory"`
	SMBFileGlob  string `json:"smbFileGlob"`
	SMBRecursive bool   `json:"smbRecursive"`
	Enabled      bool   `json:"enabled"`
}

type SMBConnectionConfig struct {
	Host     string
	Port     int
	Domain   string
	Username string
	Password string
	Share    string
}

type AutoTaskSetting struct {
	ServerID      string `json:"serverId"`
	ServerName    string `json:"serverName"`
	SyncEnabled   bool   `json:"syncEnabled"`
	SyncTime      string `json:"syncTime"`
	ImportEnabled bool   `json:"importEnabled"`
	ImportTime    string `json:"importTime"`
}

type AstrBotKeyResponse struct {
	APIKey     string `json:"apiKey"`
	HeaderName string `json:"headerName"`
}

type RemoteLogFile struct {
	FileName     string `json:"fileName"`
	Path         string `json:"path"`
	Size         int64  `json:"size"`
	LastModified any    `json:"lastModified"`
}

func (s *Service) GetSyncConfig(ctx context.Context) (SyncConfigResponse, error) {
	cfg, err := s.readSyncConfig(ctx)
	if err != nil {
		return SyncConfigResponse{}, err
	}
	sources, err := s.Sources(ctx)
	if err != nil {
		return SyncConfigResponse{}, err
	}
	cfg.Sources = sources
	cfg.AutoTasks = s.autoTasks(cfg, sources)
	if cfg.SMBPassword != "" {
		cfg.SMBPassword = "********"
	}
	return cfg, nil
}

func (s *Service) SaveSyncConfig(ctx context.Context, request SyncConfigResponse) (SyncConfigResponse, error) {
	current, err := s.readSyncConfig(ctx)
	if err != nil {
		return SyncConfigResponse{}, err
	}
	password := current.SMBPassword
	if strings.TrimSpace(request.SMBPassword) != "" && strings.TrimSpace(request.SMBPassword) != "********" {
		password = request.SMBPassword
	}
	shareTTL := NormalizeShareTTL(request.ShareTTLMinutes)
	_, err = s.db.ExecContext(ctx, `
		update sync_config
		set smb_host = ?, smb_port = ?, smb_domain = ?, smb_username = ?, smb_password = ?, smb_share = ?,
		    auto_run = ?, sync_cron = ?, skip_today = ?, share_ttl_minutes = ?,
		    auto_sync_main_enabled = ?, auto_sync_main_time = ?, auto_import_main_enabled = ?, auto_import_main_time = ?,
		    auto_sync_sub_enabled = ?, auto_sync_sub_time = ?, auto_import_sub_enabled = ?, auto_import_sub_time = ?
		where id = 1
	`, request.SMBHost, defaultPort(request.SMBPort), request.SMBDomain, request.SMBUsername, password, request.SMBShare,
		request.AutoRun, defaultString(request.SyncCron, "0 30 0 * * *"), request.SkipToday, shareTTL,
		autoBool(request.AutoTasks, "main", "sync", request.AutoRun), autoTime(request.AutoTasks, "main", "sync", "00:20"),
		autoBool(request.AutoTasks, "main", "import", request.AutoRun), autoTime(request.AutoTasks, "main", "import", "00:30"),
		autoBool(request.AutoTasks, "sub", "sync", request.AutoRun), autoTime(request.AutoTasks, "sub", "sync", "00:40"),
		autoBool(request.AutoTasks, "sub", "import", request.AutoRun), autoTime(request.AutoTasks, "sub", "import", "00:50"),
	)
	if err != nil {
		return SyncConfigResponse{}, err
	}
	if len(request.Sources) > 0 {
		if err := s.SaveSources(ctx, request.Sources); err != nil {
			return SyncConfigResponse{}, err
		}
	}
	return s.GetSyncConfig(ctx)
}

func (s *Service) AstrBotKey(ctx context.Context) (AstrBotKeyResponse, error) {
	var key sql.NullString
	if err := s.db.QueryRowContext(ctx, `select astrbot_api_key from sync_config where id = 1`).Scan(&key); err != nil {
		return AstrBotKeyResponse{}, err
	}
	return AstrBotKeyResponse{APIKey: key.String, HeaderName: AstrBotAPIKeyHeader}, nil
}

func (s *Service) ResetAstrBotKey(ctx context.Context) (AstrBotKeyResponse, error) {
	key, err := generateAPIKey()
	if err != nil {
		return AstrBotKeyResponse{}, err
	}
	if _, err := s.db.ExecContext(ctx, `update sync_config set astrbot_api_key = ? where id = 1`, key); err != nil {
		return AstrBotKeyResponse{}, err
	}
	return AstrBotKeyResponse{APIKey: key, HeaderName: AstrBotAPIKeyHeader}, nil
}

func (s *Service) ShareTTLMinutes(ctx context.Context) int {
	var minutes int
	if err := s.db.QueryRowContext(ctx, `select share_ttl_minutes from sync_config where id = 1`).Scan(&minutes); err != nil {
		return 60
	}
	return NormalizeShareTTL(minutes)
}

func (s *Service) SMBConnectionConfig(ctx context.Context) (SMBConnectionConfig, error) {
	cfg, err := s.readSyncConfig(ctx)
	if err != nil {
		return SMBConnectionConfig{}, err
	}
	return SMBConnectionConfig{
		Host:     strings.TrimSpace(cfg.SMBHost),
		Port:     defaultPort(cfg.SMBPort),
		Domain:   strings.TrimSpace(cfg.SMBDomain),
		Username: strings.TrimSpace(cfg.SMBUsername),
		Password: cfg.SMBPassword,
		Share:    strings.TrimSpace(cfg.SMBShare),
	}, nil
}

func (s *Service) Sources(ctx context.Context) ([]SyncSource, error) {
	rows, err := s.db.QueryContext(ctx, `
		select id, source_id, source_name, coalesce(smb_directory, ''), coalesce(smb_file_glob, ''), smb_recursive, enabled
		from sync_source_config
		order by source_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []SyncSource
	for rows.Next() {
		var source SyncSource
		var id int64
		if err := rows.Scan(&id, &source.SourceID, &source.SourceName, &source.SMBDirectory, &source.SMBFileGlob, &source.SMBRecursive, &source.Enabled); err != nil {
			return nil, err
		}
		source.ID = &id
		result = append(result, source)
	}
	return result, rows.Err()
}

func (s *Service) SaveSources(ctx context.Context, sources []SyncSource) error {
	for _, source := range sources {
		if source.ID != nil {
			_, err := s.db.ExecContext(ctx, `
				update sync_source_config
				set source_name = ?, smb_directory = ?, smb_file_glob = ?, smb_recursive = ?, enabled = ?
				where id = ?
			`, source.SourceName, source.SMBDirectory, defaultString(source.SMBFileGlob, "player_actions_*.csv"), source.SMBRecursive, source.Enabled, *source.ID)
			if err != nil {
				return err
			}
			continue
		}
		if strings.TrimSpace(source.SourceID) == "" {
			continue
		}
		_, err := s.db.ExecContext(ctx, `
			insert into sync_source_config (source_id, source_name, smb_directory, smb_file_glob, smb_recursive, enabled)
			values (?, ?, ?, ?, ?, ?)
			on duplicate key update source_name = values(source_name), smb_directory = values(smb_directory),
			    smb_file_glob = values(smb_file_glob), smb_recursive = values(smb_recursive), enabled = values(enabled)
		`, source.SourceID, source.SourceName, source.SMBDirectory, defaultString(source.SMBFileGlob, "player_actions_*.csv"), source.SMBRecursive, source.Enabled)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) SourceByID(ctx context.Context, sourceID string) (SyncSource, bool, error) {
	var source SyncSource
	var id int64
	err := s.db.QueryRowContext(ctx, `
		select id, source_id, source_name, coalesce(smb_directory, ''), coalesce(smb_file_glob, ''), smb_recursive, enabled
		from sync_source_config
		where source_id = ?
	`, sourceID).Scan(&id, &source.SourceID, &source.SourceName, &source.SMBDirectory, &source.SMBFileGlob, &source.SMBRecursive, &source.Enabled)
	if errors.Is(err, sql.ErrNoRows) {
		return SyncSource{}, false, nil
	}
	if err != nil {
		return SyncSource{}, false, err
	}
	source.ID = &id
	return source, true, nil
}

func (s *Service) LocalSourceFiles(ctx context.Context, sourceID string) ([]RemoteLogFile, error) {
	source, ok := s.cfgSource(sourceID)
	if !ok {
		return nil, errors.New("找不到数据源：" + sourceID)
	}
	pattern := defaultString(source.FileGlob, "player_actions_*.csv")
	var files []RemoteLogFile
	err := filepath.WalkDir(source.Directory, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			if path != source.Directory {
				return filepath.SkipDir
			}
			return nil
		}
		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err != nil || !matched {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return nil
		}
		files = append(files, RemoteLogFile{
			FileName:     info.Name(),
			Path:         path,
			Size:         info.Size(),
			LastModified: info.ModTime(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

func (s *Service) readSyncConfig(ctx context.Context) (SyncConfigResponse, error) {
	var cfg SyncConfigResponse
	err := s.db.QueryRowContext(ctx, `
		select coalesce(smb_host, ''), smb_port, coalesce(smb_domain, ''), coalesce(smb_username, ''),
		       coalesce(smb_password, ''), coalesce(smb_share, ''), auto_run, coalesce(sync_cron, ''),
		       skip_today, share_ttl_minutes
		from sync_config where id = 1
	`).Scan(&cfg.SMBHost, &cfg.SMBPort, &cfg.SMBDomain, &cfg.SMBUsername, &cfg.SMBPassword, &cfg.SMBShare,
		&cfg.AutoRun, &cfg.SyncCron, &cfg.SkipToday, &cfg.ShareTTLMinutes)
	if err != nil {
		return SyncConfigResponse{}, err
	}
	cfg.ShareTTLMinutes = NormalizeShareTTL(cfg.ShareTTLMinutes)
	return cfg, nil
}

func (s *Service) autoTasks(cfg SyncConfigResponse, sources []SyncSource) []AutoTaskSetting {
	result := make([]AutoTaskSetting, 0, len(sources))
	for _, source := range sources {
		switch strings.ToLower(strings.TrimSpace(source.SourceID)) {
		case "main":
			result = append(result, s.readAutoTask("main", source.SourceName, "00:20", "00:30", cfg.AutoRun))
		case "sub":
			result = append(result, s.readAutoTask("sub", source.SourceName, "00:40", "00:50", cfg.AutoRun))
		default:
			result = append(result, AutoTaskSetting{ServerID: source.SourceID, ServerName: source.SourceName, SyncTime: "01:00", ImportTime: "01:10"})
		}
	}
	return result
}

func (s *Service) readAutoTask(serverID, serverName, defaultSync, defaultImport string, defaultEnabled bool) AutoTaskSetting {
	var syncEnabled, importEnabled sql.NullBool
	var syncTime, importTime sql.NullString
	if serverID == "main" {
		_ = s.db.QueryRow(`
			select auto_sync_main_enabled, auto_sync_main_time, auto_import_main_enabled, auto_import_main_time from sync_config where id = 1
		`).Scan(&syncEnabled, &syncTime, &importEnabled, &importTime)
	} else {
		_ = s.db.QueryRow(`
			select auto_sync_sub_enabled, auto_sync_sub_time, auto_import_sub_enabled, auto_import_sub_time from sync_config where id = 1
		`).Scan(&syncEnabled, &syncTime, &importEnabled, &importTime)
	}
	return AutoTaskSetting{
		ServerID:      serverID,
		ServerName:    serverName,
		SyncEnabled:   nullBool(syncEnabled, defaultEnabled),
		SyncTime:      normalizeTime(syncTime.String, defaultSync),
		ImportEnabled: nullBool(importEnabled, defaultEnabled),
		ImportTime:    normalizeTime(importTime.String, defaultImport),
	}
}

func (s *Service) cfgSource(sourceID string) (config.Source, bool) {
	for _, source := range s.cfg.Sources() {
		if source.ID == sourceID {
			return source, true
		}
	}
	return config.Source{}, false
}

func NormalizeShareTTL(minutes int) int {
	if minutes <= 0 {
		return 60
	}
	if minutes < 5 {
		return 5
	}
	if minutes > 10080 {
		return 10080
	}
	return minutes
}

func generateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "psk_" + base64.RawURLEncoding.EncodeToString(bytes), nil
}

func defaultPort(port int) int {
	if port <= 0 {
		return 445
	}
	return port
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func nullBool(value sql.NullBool, fallback bool) bool {
	if value.Valid {
		return value.Bool
	}
	return fallback
}

func autoBool(tasks []AutoTaskSetting, serverID, kind string, fallback bool) bool {
	for _, task := range tasks {
		if strings.EqualFold(task.ServerID, serverID) {
			if kind == "sync" {
				return task.SyncEnabled
			}
			return task.ImportEnabled
		}
	}
	return fallback
}

func autoTime(tasks []AutoTaskSetting, serverID, kind, fallback string) string {
	for _, task := range tasks {
		if strings.EqualFold(task.ServerID, serverID) {
			if kind == "sync" {
				return normalizeTime(task.SyncTime, fallback)
			}
			return normalizeTime(task.ImportTime, fallback)
		}
	}
	return fallback
}

func normalizeTime(value, fallback string) string {
	value = strings.TrimSpace(value)
	if len(value) == 5 && value[2] == ':' {
		return value
	}
	return fallback
}
