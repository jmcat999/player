package importer

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"player-stats-backend-go/internal/apitype"
	"player-stats-backend-go/internal/auth"
	"player-stats-backend-go/internal/config"
	"player-stats-backend-go/internal/settings"
)

type Service struct {
	db       *sql.DB
	cfg      config.Config
	settings *settings.Service
	parser   parser
}

type ProgressListener interface {
	FileStarted(serverID, serverName string, file RemoteLogFile)
	FileFinished(result FileImportResult)
}

func NewService(db *sql.DB, cfg config.Config, settingsService *settings.Service) *Service {
	return &Service{db: db, cfg: cfg, settings: settingsService, parser: newParser(cfg.Location)}
}

func (s *Service) ListLocalImportFiles(ctx context.Context, requestedServerID string) ([]ImportFileStatus, error) {
	result := make([]ImportFileStatus, 0)
	for _, source := range s.selectedSources(ctx, requestedServerID) {
		files, err := s.localFiles(source)
		if err != nil {
			result = append(result, ImportFileStatus{
				ServerID:   source.ID,
				ServerName: source.Name,
				RemotePath: source.Directory,
				FileName:   source.Directory,
				Status:     "FAILED",
				Message:    err.Error(),
			})
			continue
		}
		for _, file := range limitFiles(files, s.cfg.MaxFilesPerRun) {
			status, err := s.importFileStatus(ctx, source, file, s.cfg.SkipToday)
			if err != nil {
				return nil, err
			}
			result = append(result, status)
		}
	}
	return result, nil
}

func (s *Service) ListRemoteSMBFiles(ctx context.Context, requestedServerID string) ([]ImportFileStatus, error) {
	return s.listRemoteSMBFiles(ctx, requestedServerID)
}

func (s *Service) ImportFromConfiguredSource(ctx context.Context, requestedServerID string, skipToday bool, listener ProgressListener) (ImportRunResult, error) {
	startedAt := time.Now().UTC()
	files := make([]FileImportResult, 0)
	for _, source := range s.selectedSources(ctx, requestedServerID) {
		localFiles, err := s.localFiles(source)
		if err != nil {
			files = appendResult(files, listener, failedResult(source.ID, source.Name, source.Directory, err.Error()))
			continue
		}
		for _, file := range limitFiles(localFiles, s.cfg.MaxFilesPerRun) {
			if listener != nil {
				listener.FileStarted(source.ID, source.Name, file)
			}
			result := s.importOne(ctx, source, file, skipToday)
			files = appendResult(files, listener, result)
		}
	}
	return summarizeRun(startedAt, files), nil
}

func (s *Service) SyncFilesFromConfiguredSource(ctx context.Context, requestedServerID string, skipToday bool, listener ProgressListener) (ImportRunResult, error) {
	return s.syncFilesFromSMBSource(ctx, requestedServerID, skipToday, listener)
}

func (s *Service) DeleteImportRecord(ctx context.Context, serverID, remotePath string) (DeleteImportRecordResult, error) {
	serverID = strings.TrimSpace(serverID)
	remotePath = strings.TrimSpace(remotePath)
	if serverID == "" || remotePath == "" {
		return DeleteImportRecordResult{}, auth.NewHTTPError(400, "serverId and remotePath are required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return DeleteImportRecordResult{}, err
	}
	defer rollbackQuietly(tx)

	record, found, err := findImportFileTx(ctx, tx, serverID, remotePath)
	if err != nil {
		return DeleteImportRecordResult{}, err
	}
	if !found {
		return DeleteImportRecordResult{}, auth.NewHTTPError(404, "Import record not found")
	}
	previous, err := loadFileStats(ctx, tx, record.id)
	if err != nil {
		return DeleteImportRecordResult{}, err
	}
	affectedPlayers, err := loadFileSeenPlayers(ctx, tx, record.id)
	if err != nil {
		return DeleteImportRecordResult{}, err
	}
	if err := subtractDeltas(ctx, tx, record.serverID, record.serverName, previous); err != nil {
		return DeleteImportRecordResult{}, err
	}
	if err := deleteFileDetails(ctx, tx, record.id); err != nil {
		return DeleteImportRecordResult{}, err
	}
	if _, err := tx.ExecContext(ctx, `delete from imported_server_log_files where id = ?`, record.id); err != nil {
		return DeleteImportRecordResult{}, err
	}
	if err := s.syncProfiles(ctx, tx, record.serverID, record.serverName, affectedPlayers); err != nil {
		return DeleteImportRecordResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return DeleteImportRecordResult{}, err
	}
	return DeleteImportRecordResult{ServerID: serverID, RemotePath: remotePath, Deleted: true, Message: "Import record deleted"}, nil
}

func (s *Service) DeleteImportRecords(ctx context.Context, request DeleteImportRecordsRequest) []DeleteImportRecordResult {
	if len(request.Files) == 0 {
		return []DeleteImportRecordResult{{Deleted: false, Message: "files are required"}}
	}
	results := make([]DeleteImportRecordResult, 0, len(request.Files))
	for _, file := range request.Files {
		result, err := s.DeleteImportRecord(ctx, file.ServerID, file.RemotePath)
		if err != nil {
			result = DeleteImportRecordResult{ServerID: file.ServerID, RemotePath: file.RemotePath, Deleted: false, Message: err.Error()}
		}
		results = append(results, result)
	}
	return results
}

func (s *Service) DeleteLocalFile(ctx context.Context, serverID, remotePath string) (DeleteImportRecordResult, error) {
	serverID = strings.TrimSpace(serverID)
	remotePath = strings.TrimSpace(remotePath)
	if serverID == "" || remotePath == "" {
		return DeleteImportRecordResult{}, auth.NewHTTPError(400, "serverId and remotePath are required")
	}
	if strings.EqualFold(serverID, "all") || strings.EqualFold(serverID, "total") {
		return DeleteImportRecordResult{}, auth.NewHTTPError(400, "请指定单个服务器后再删除本地 CSV")
	}
	source, ok := s.sourceByID(ctx, serverID)
	if !ok {
		return DeleteImportRecordResult{}, auth.NewHTTPError(404, "Server not found")
	}
	base, err := filepath.Abs(source.Directory)
	if err != nil {
		return DeleteImportRecordResult{}, err
	}
	target, err := filepath.Abs(remotePath)
	if err != nil {
		return DeleteImportRecordResult{}, err
	}
	rel, err := filepath.Rel(base, target)
	if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return DeleteImportRecordResult{}, auth.NewHTTPError(400, "只能删除当前服务器本地目录内的 CSV 文件")
	}
	if !matchesGlob(target, source.FileGlob) {
		return DeleteImportRecordResult{}, auth.NewHTTPError(400, "只能删除匹配当前日志规则的 CSV 文件")
	}
	info, err := os.Stat(target)
	if errors.Is(err, os.ErrNotExist) {
		return DeleteImportRecordResult{ServerID: serverID, RemotePath: remotePath, Deleted: false, Message: "Local CSV not found"}, nil
	}
	if err != nil {
		return DeleteImportRecordResult{}, err
	}
	if info.IsDir() {
		return DeleteImportRecordResult{}, auth.NewHTTPError(400, "只能删除本地 CSV 文件")
	}
	if err := os.Remove(target); err != nil {
		return DeleteImportRecordResult{}, err
	}
	return DeleteImportRecordResult{ServerID: serverID, RemotePath: remotePath, Deleted: true, Message: "Local CSV deleted"}, nil
}

func (s *Service) DeleteLocalFiles(ctx context.Context, request DeleteImportRecordsRequest) []DeleteImportRecordResult {
	if len(request.Files) == 0 {
		return []DeleteImportRecordResult{{Deleted: false, Message: "files are required"}}
	}
	results := make([]DeleteImportRecordResult, 0, len(request.Files))
	for _, file := range request.Files {
		result, err := s.DeleteLocalFile(ctx, file.ServerID, file.RemotePath)
		if err != nil {
			result = DeleteImportRecordResult{ServerID: file.ServerID, RemotePath: file.RemotePath, Deleted: false, Message: err.Error()}
		}
		results = append(results, result)
	}
	return results
}

func (s *Service) LatestAutoTaskLogs(ctx context.Context, limit int) ([]AutoTaskLogView, error) {
	limit = max(1, min(limit, 200))
	rows, err := s.db.QueryContext(ctx, `
		select id, created_at, coalesce(server_id, ''), coalesce(server_name, ''), coalesce(task_type, ''),
		       coalesce(task_label, ''), coalesce(status, ''), coalesce(message, ''), coalesce(file_details, ''),
		       scanned_files, success_files, skipped_files, failed_files
		from auto_task_logs
		order by created_at desc
		limit ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]AutoTaskLogView, 0)
	for rows.Next() {
		var item AutoTaskLogView
		if err := rows.Scan(&item.ID, &item.CreatedAt, &item.ServerID, &item.ServerName, &item.TaskType,
			&item.TaskLabel, &item.Status, &item.Message, &item.FileDetails, &item.ScannedFiles,
			&item.SuccessFiles, &item.SkippedFiles, &item.FailedFiles); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Service) ClearAutoTaskLogs(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `delete from auto_task_logs`)
	return err
}

func (s *Service) importOne(ctx context.Context, source config.Source, file RemoteLogFile, skipToday bool) FileImportResult {
	logDate := extractLogDate(file.FileName, s.cfg.Location)
	effectiveDate := logDate
	if effectiveDate == nil {
		value := dateOnly(file.LastModified.In(s.cfg.Location), s.cfg.Location)
		effectiveDate = &value
	}
	today := dateOnly(time.Now().In(s.cfg.Location), s.cfg.Location)
	if skipToday && !effectiveDate.Before(today) {
		return skippedResult(source.ID, source.Name, file.Path, "跳过当天或未来的日志文件")
	}
	status, err := s.importFileStatus(ctx, source, file, skipToday)
	if err != nil {
		return failedResult(source.ID, source.Name, file.Path, err.Error())
	}
	if status.Status == "IMPORTED" {
		return skippedResult(source.ID, source.Name, file.Path, status.Message)
	}
	opened, err := os.Open(file.Path)
	if err != nil {
		return failedResult(source.ID, source.Name, file.Path, err.Error())
	}
	defer opened.Close()
	parsed, err := s.parser.parse(opened)
	if err != nil && err != io.EOF {
		return failedResult(source.ID, source.Name, file.Path, err.Error())
	}
	if err := s.applyImport(ctx, source, file, logDate, parsed); err != nil {
		return failedResult(source.ID, source.Name, file.Path, err.Error())
	}
	return importedResult(source.ID, source.Name, file.Path, parsed.rowCount, parsed.ignoredCount)
}

func (s *Service) applyImport(ctx context.Context, source config.Source, file RemoteLogFile, logDate *time.Time, parsed parsedLogFile) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollbackQuietly(tx)

	existing, found, err := findImportFileTx(ctx, tx, source.ID, file.Path)
	if err != nil {
		return err
	}
	previousStats := map[statKey]actionCounts{}
	affectedPlayers := map[string]struct{}{}
	if found {
		previousStats, err = loadFileStats(ctx, tx, existing.id)
		if err != nil {
			return err
		}
		affectedPlayers, err = loadFileSeenPlayers(ctx, tx, existing.id)
		if err != nil {
			return err
		}
	}
	for player := range parsed.firstSeenByPlayer {
		affectedPlayers[player] = struct{}{}
	}

	importFileID, err := upsertImportFile(ctx, tx, existing.id, source, file, logDate, parsed)
	if err != nil {
		return err
	}
	if err := applyDeltas(ctx, tx, source.ID, source.Name, previousStats, parsed.stats); err != nil {
		return err
	}
	if err := deleteFileDetails(ctx, tx, importFileID); err != nil {
		return err
	}
	if err := insertParsedDetails(ctx, tx, importFileID, parsed); err != nil {
		return err
	}
	if err := s.syncProfiles(ctx, tx, source.ID, source.Name, affectedPlayers); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Service) importFileStatus(ctx context.Context, source config.Source, file RemoteLogFile, skipToday bool) (ImportFileStatus, error) {
	logDate := extractLogDate(file.FileName, s.cfg.Location)
	effectiveDate := logDate
	if effectiveDate == nil {
		value := dateOnly(file.LastModified.In(s.cfg.Location), s.cfg.Location)
		effectiveDate = &value
	}
	today := dateOnly(time.Now().In(s.cfg.Location), s.cfg.Location)
	var datePtr *apitype.Date
	if logDate != nil {
		value := apitype.NewDate(*logDate)
		datePtr = &value
	}
	record, found, err := s.findImportFile(ctx, source.ID, file.Path)
	if err != nil {
		return ImportFileStatus{}, err
	}
	if skipToday && !effectiveDate.Before(today) {
		return s.statusFromRecord(source, file, datePtr, record, found, "SKIPPED_TODAY", "跳过当天或未来的日志文件"), nil
	}
	if !found {
		return s.statusFromRecord(source, file, datePtr, record, false, "PENDING", "未导入"), nil
	}
	sameMetadata := record.fileSize == file.Size && record.lastModified.UnixMilli() == file.LastModified.UnixMilli()
	hasRows, err := s.hasImportedDetailRows(ctx, record.id)
	if err != nil {
		return ImportFileStatus{}, err
	}
	if sameMetadata && hasRows {
		return s.statusFromRecord(source, file, datePtr, record, true, "IMPORTED", "文件大小和修改时间未变化"), nil
	}
	if sameMetadata {
		return s.statusFromRecord(source, file, datePtr, record, true, "NEEDS_IMPORT", "导入记录不完整，可重新解析"), nil
	}
	return s.statusFromRecord(source, file, datePtr, record, true, "CHANGED", "本地 CSV 已变更，可重新解析"), nil
}

func (s *Service) statusFromRecord(source config.Source, file RemoteLogFile, logDate *apitype.Date, record importedFileRecord, found bool, status, message string) ImportFileStatus {
	var importedAt *time.Time
	rowCount := 0
	ignoredCount := 0
	if found {
		importedAt = &record.importedAt
		rowCount = record.rowCount
		ignoredCount = record.ignoredCount
	}
	return ImportFileStatus{
		ServerID:     source.ID,
		ServerName:   source.Name,
		RemotePath:   file.Path,
		FileName:     file.FileName,
		FileSize:     file.Size,
		LastModified: file.LastModified,
		LogDate:      logDate,
		Imported:     found,
		ImportedAt:   importedAt,
		RowCount:     rowCount,
		IgnoredCount: ignoredCount,
		Status:       status,
		Message:      message,
	}
}

func (s *Service) selectedSources(ctx context.Context, requestedServerID string) []config.Source {
	requestedServerID = strings.TrimSpace(requestedServerID)
	all := requestedServerID == "" || strings.EqualFold(requestedServerID, "all") || strings.EqualFold(requestedServerID, "total")
	result := make([]config.Source, 0)
	for _, source := range s.cfg.Sources() {
		if !all && source.ID != requestedServerID {
			continue
		}
		if dbSource, ok, err := s.settings.SourceByID(ctx, source.ID); err == nil && ok {
			if !dbSource.Enabled {
				continue
			}
			if strings.TrimSpace(dbSource.SourceName) != "" {
				source.Name = dbSource.SourceName
			}
		}
		result = append(result, source)
	}
	return result
}

func (s *Service) sourceByID(ctx context.Context, serverID string) (config.Source, bool) {
	for _, source := range s.selectedSources(ctx, serverID) {
		if source.ID == serverID {
			return source, true
		}
	}
	return config.Source{}, false
}

func (s *Service) localFiles(source config.Source) ([]RemoteLogFile, error) {
	directory, err := filepath.Abs(source.Directory)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(directory, 0755); err != nil {
		return nil, err
	}
	files := make([]RemoteLogFile, 0)
	err = filepath.WalkDir(directory, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			if path != directory {
				return filepath.SkipDir
			}
			return nil
		}
		if !matchesGlob(path, source.FileGlob) {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return nil
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return nil
		}
		files = append(files, RemoteLogFile{
			Path:         filepath.Clean(abs),
			FileName:     info.Name(),
			Size:         info.Size(),
			LastModified: info.ModTime().UTC(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

type importedFileRecord struct {
	id           int64
	serverID     string
	serverName   string
	remotePath   string
	fileName     string
	logDate      sql.NullTime
	fileSize     int64
	lastModified time.Time
	contentHash  string
	importedAt   time.Time
	rowCount     int
	ignoredCount int
}

func (s *Service) findImportFile(ctx context.Context, serverID, remotePath string) (importedFileRecord, bool, error) {
	return scanImportFile(s.db.QueryRowContext(ctx, importFileSelectQuery()+` where server_id = ? and remote_path_hash = ? and remote_path = ?`, serverID, remotePathHash(remotePath), remotePath))
}

func findImportFileTx(ctx context.Context, tx *sql.Tx, serverID, remotePath string) (importedFileRecord, bool, error) {
	return scanImportFile(tx.QueryRowContext(ctx, importFileSelectQuery()+` where server_id = ? and remote_path_hash = ? and remote_path = ? for update`, serverID, remotePathHash(remotePath), remotePath))
}

func scanImportFile(row *sql.Row) (importedFileRecord, bool, error) {
	var record importedFileRecord
	err := row.Scan(&record.id, &record.serverID, &record.serverName, &record.remotePath, &record.fileName,
		&record.logDate, &record.fileSize, &record.lastModified, &record.contentHash, &record.importedAt,
		&record.rowCount, &record.ignoredCount)
	if errors.Is(err, sql.ErrNoRows) {
		return importedFileRecord{}, false, nil
	}
	if err != nil {
		return importedFileRecord{}, false, err
	}
	return record, true, nil
}

func importFileSelectQuery() string {
	return `select id, server_id, server_name, remote_path, file_name, log_date, file_size, last_modified, content_hash, imported_at, row_count, ignored_count from imported_server_log_files`
}

func upsertImportFile(ctx context.Context, tx *sql.Tx, existingID int64, source config.Source, file RemoteLogFile, logDate *time.Time, parsed parsedLogFile) (int64, error) {
	importedAt := time.Now().UTC()
	pathHash := remotePathHash(file.Path)
	if existingID > 0 {
		_, err := tx.ExecContext(ctx, `
			update imported_server_log_files
			set server_id = ?, server_name = ?, remote_path = ?, remote_path_hash = ?, file_name = ?, log_date = ?,
			    file_size = ?, last_modified = ?, content_hash = ?, imported_at = ?, row_count = ?, ignored_count = ?
			where id = ?
		`, source.ID, source.Name, file.Path, pathHash, file.FileName, nullableDate(logDate), file.Size, file.LastModified, parsed.contentHash, importedAt, parsed.rowCount, parsed.ignoredCount, existingID)
		return existingID, err
	}
	result, err := tx.ExecContext(ctx, `
		insert into imported_server_log_files
		    (server_id, server_name, remote_path, remote_path_hash, file_name, log_date, file_size, last_modified, content_hash, imported_at, row_count, ignored_count)
		values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, source.ID, source.Name, file.Path, pathHash, file.FileName, nullableDate(logDate), file.Size, file.LastModified, parsed.contentHash, importedAt, parsed.rowCount, parsed.ignoredCount)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func remotePathHash(remotePath string) string {
	sum := sha256.Sum256([]byte(remotePath))
	return hex.EncodeToString(sum[:])
}

func loadFileStats(ctx context.Context, tx *sql.Tx, importFileID int64) (map[statKey]actionCounts, error) {
	rows, err := tx.QueryContext(ctx, `
		select stat_date, player_name, broken_count, placed_count
		from player_server_log_file_stats
		where import_file_id = ?
	`, importFileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[statKey]actionCounts{}
	for rows.Next() {
		var statDate time.Time
		var playerName string
		var counts actionCounts
		if err := rows.Scan(&statDate, &playerName, &counts.broken, &counts.placed); err != nil {
			return nil, err
		}
		result[statKey{statDate: statDate, playerName: playerName}] = counts
	}
	return result, rows.Err()
}

func loadFileSeenPlayers(ctx context.Context, tx *sql.Tx, importFileID int64) (map[string]struct{}, error) {
	rows, err := tx.QueryContext(ctx, `select player_name from player_server_log_file_seen where import_file_id = ?`, importFileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[string]struct{}{}
	for rows.Next() {
		var playerName string
		if err := rows.Scan(&playerName); err != nil {
			return nil, err
		}
		result[playerName] = struct{}{}
	}
	return result, rows.Err()
}

func applyDeltas(ctx context.Context, tx *sql.Tx, serverID, serverName string, previous, next map[statKey]actionCounts) error {
	allKeys := map[statKey]struct{}{}
	for key := range previous {
		allKeys[key] = struct{}{}
	}
	for key := range next {
		allKeys[key] = struct{}{}
	}
	for key := range allKeys {
		prev := previous[key]
		nxt := next[key]
		brokenDelta := nxt.broken - prev.broken
		placedDelta := nxt.placed - prev.placed
		if brokenDelta == 0 && placedDelta == 0 {
			continue
		}
		if err := applyDailyDelta(ctx, tx, serverID, serverName, key, brokenDelta, placedDelta); err != nil {
			return err
		}
	}
	return nil
}

func subtractDeltas(ctx context.Context, tx *sql.Tx, serverID, serverName string, previous map[statKey]actionCounts) error {
	empty := map[statKey]actionCounts{}
	return applyDeltas(ctx, tx, serverID, serverName, previous, empty)
}

func applyDailyDelta(ctx context.Context, tx *sql.Tx, serverID, serverName string, key statKey, brokenDelta, placedDelta int64) error {
	var id int64
	var currentBroken, currentPlaced int64
	err := tx.QueryRowContext(ctx, `
		select id, broken_count, placed_count
		from player_server_daily_stats
		where server_id = ? and stat_date = ? and player_name = ?
		for update
	`, serverID, key.statDate, key.playerName).Scan(&id, &currentBroken, &currentPlaced)
	if errors.Is(err, sql.ErrNoRows) {
		if brokenDelta <= 0 && placedDelta <= 0 {
			return nil
		}
		_, err = tx.ExecContext(ctx, `
			insert into player_server_daily_stats (server_id, server_name, stat_date, player_name, broken_count, placed_count)
			values (?, ?, ?, ?, ?, ?)
		`, serverID, serverName, key.statDate, key.playerName, maxInt64(0, brokenDelta), maxInt64(0, placedDelta))
		return err
	}
	if err != nil {
		return err
	}
	nextBroken := maxInt64(0, currentBroken+brokenDelta)
	nextPlaced := maxInt64(0, currentPlaced+placedDelta)
	if nextBroken == 0 && nextPlaced == 0 {
		_, err = tx.ExecContext(ctx, `delete from player_server_daily_stats where id = ?`, id)
		return err
	}
	_, err = tx.ExecContext(ctx, `
		update player_server_daily_stats
		set server_name = ?, broken_count = ?, placed_count = ?
		where id = ?
	`, serverName, nextBroken, nextPlaced, id)
	return err
}

func deleteFileDetails(ctx context.Context, tx *sql.Tx, importFileID int64) error {
	tables := []string{
		"player_server_log_file_stats",
		"player_server_log_file_seen",
		"player_server_log_file_ore_stats",
		"player_server_log_file_wood_stats",
		"player_server_log_file_sapling_stats",
		"player_server_log_file_milestones",
	}
	for _, table := range tables {
		if _, err := tx.ExecContext(ctx, `delete from `+table+` where import_file_id = ?`, importFileID); err != nil {
			return err
		}
	}
	return nil
}

func insertParsedDetails(ctx context.Context, tx *sql.Tx, importFileID int64, parsed parsedLogFile) error {
	for key, counts := range parsed.stats {
		if counts.broken == 0 && counts.placed == 0 {
			continue
		}
		if _, err := tx.ExecContext(ctx, `
			insert into player_server_log_file_stats (import_file_id, stat_date, player_name, broken_count, placed_count)
			values (?, ?, ?, ?, ?)
		`, importFileID, key.statDate, key.playerName, counts.broken, counts.placed); err != nil {
			return err
		}
	}
	for playerName, firstSeenAt := range parsed.firstSeenByPlayer {
		if _, err := tx.ExecContext(ctx, `
			insert into player_server_log_file_seen (import_file_id, player_name, first_seen_at)
			values (?, ?, ?)
		`, importFileID, playerName, firstSeenAt); err != nil {
			return err
		}
	}
	if err := insertTypedCounts(ctx, tx, importFileID, parsed.oreCounts, "player_server_log_file_ore_stats", "ore_type", "ore_count"); err != nil {
		return err
	}
	if err := insertTypedCounts(ctx, tx, importFileID, parsed.woodCounts, "player_server_log_file_wood_stats", "wood_type", "wood_count"); err != nil {
		return err
	}
	if err := insertTypedCounts(ctx, tx, importFileID, parsed.saplingCounts, "player_server_log_file_sapling_stats", "sapling_type", "sapling_count"); err != nil {
		return err
	}
	for key, value := range parsed.milestones {
		if _, err := tx.ExecContext(ctx, `
			insert into player_server_log_file_milestones (import_file_id, player_name, milestone_type, first_seen_at, detail)
			values (?, ?, ?, ?, ?)
		`, importFileID, key.playerName, key.typ, value.firstSeenAt, value.detail); err != nil {
			return err
		}
	}
	return nil
}

func insertTypedCounts(ctx context.Context, tx *sql.Tx, importFileID int64, counts map[typedPlayerKey]int64, table, typeColumn, countColumn string) error {
	for key, count := range counts {
		if count <= 0 {
			continue
		}
		query := `insert into ` + table + ` (import_file_id, player_name, ` + typeColumn + `, ` + countColumn + `) values (?, ?, ?, ?)`
		if _, err := tx.ExecContext(ctx, query, importFileID, key.playerName, key.typ, count); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) syncProfiles(ctx context.Context, tx *sql.Tx, serverID, serverName string, playerNames map[string]struct{}) error {
	for playerName := range playerNames {
		var firstSeen sql.NullTime
		err := tx.QueryRowContext(ctx, `
			select min(s.first_seen_at)
			from player_server_log_file_seen s
			join imported_server_log_files f on f.id = s.import_file_id
			where f.server_id = ? and lower(s.player_name) = lower(?)
		`, serverID, playerName).Scan(&firstSeen)
		if err != nil {
			return err
		}
		if !firstSeen.Valid {
			if _, err := tx.ExecContext(ctx, `delete from player_server_profiles where server_id = ? and lower(player_name) = lower(?)`, serverID, playerName); err != nil {
				return err
			}
			continue
		}
		if _, err := tx.ExecContext(ctx, `
			insert into player_server_profiles (server_id, server_name, player_name, first_seen_at)
			values (?, ?, ?, ?)
			on duplicate key update server_name = values(server_name), player_name = values(player_name), first_seen_at = values(first_seen_at)
		`, serverID, serverName, playerName, firstSeen.Time); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) hasImportedDetailRows(ctx context.Context, importFileID int64) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		select (
			(select count(*) from player_server_log_file_seen where import_file_id = ?) +
			(select count(*) from player_server_log_file_stats where import_file_id = ?)
		)
	`, importFileID, importFileID).Scan(&count)
	return count > 0, err
}

func extractLogDate(fileName string, location *time.Location) *time.Time {
	re := regexp.MustCompile(`.*?(\d{4}-\d{2}-\d{2}).*\.csv$`)
	match := re.FindStringSubmatch(fileName)
	if len(match) < 2 {
		return nil
	}
	value, err := time.ParseInLocation("2006-01-02", match[1], location)
	if err != nil {
		return nil
	}
	return &value
}

func nullableDate(value *time.Time) any {
	if value == nil {
		return nil
	}
	return *value
}

func dateOnly(value time.Time, location *time.Location) time.Time {
	value = value.In(location)
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, location)
}

func matchesGlob(path, glob string) bool {
	if strings.TrimSpace(glob) == "" {
		glob = "player_actions_*.csv"
	}
	matched, err := filepath.Match(glob, filepath.Base(path))
	return err == nil && matched
}

func limitFiles(files []RemoteLogFile, limit int) []RemoteLogFile {
	if limit <= 0 {
		limit = 1000
	}
	if len(files) <= limit {
		return files
	}
	return files[:limit]
}

func summarizeRun(startedAt time.Time, files []FileImportResult) ImportRunResult {
	result := ImportRunResult{StartedAt: startedAt, FinishedAt: time.Now().UTC(), ScannedFiles: len(files), Files: files}
	for _, file := range files {
		switch file.Status {
		case "IMPORTED", "COPIED":
			result.ImportedFiles++
		case "SKIPPED":
			result.SkippedFiles++
		case "FAILED":
			result.FailedFiles++
		}
	}
	return result
}

func appendResult(files []FileImportResult, listener ProgressListener, result FileImportResult) []FileImportResult {
	files = append(files, result)
	if listener != nil {
		listener.FileFinished(result)
	}
	return files
}

func rollbackQuietly(tx *sql.Tx) {
	_ = tx.Rollback()
}

func maxInt64(left, right int64) int64 {
	if left > right {
		return left
	}
	return right
}
