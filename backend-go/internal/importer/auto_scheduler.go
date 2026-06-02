package importer

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"player-stats-backend-go/internal/settings"
)

type AutoTaskScheduler struct {
	settings *settings.Service
	importer *Service
	logger   *slog.Logger
	location *time.Location

	mu      sync.Mutex
	lastRun map[string]string
	running map[string]bool
	stop    chan struct{}
	done    chan struct{}
}

func NewAutoTaskScheduler(settingsService *settings.Service, importService *Service, logger *slog.Logger, location *time.Location) *AutoTaskScheduler {
	return &AutoTaskScheduler{
		settings: settingsService,
		importer: importService,
		logger:   logger,
		location: location,
		lastRun:  map[string]string{},
		running:  map[string]bool{},
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}
}

func (s *AutoTaskScheduler) Start() {
	go s.loop()
}

func (s *AutoTaskScheduler) Stop() {
	close(s.stop)
	<-s.done
}

func (s *AutoTaskScheduler) loop() {
	defer close(s.done)
	s.recordConfig(context.Background(), "自动任务调度器已启动")
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	s.check(time.Now().In(s.location))
	for {
		select {
		case <-ticker.C:
			s.check(time.Now().In(s.location))
		case <-s.stop:
			s.recordConfig(context.Background(), "自动任务调度器已停止")
			return
		}
	}
}

func (s *AutoTaskScheduler) check(now time.Time) {
	cfg, err := s.settings.GetSyncConfig(context.Background())
	if err != nil {
		s.logger.Warn("auto task config load failed", "error", err)
		return
	}
	currentTime := now.Format("15:04")
	currentDate := now.Format("2006-01-02")
	for _, task := range cfg.AutoTasks {
		if task.SyncEnabled && task.SyncTime == currentTime {
			s.launch(currentDate, task, "SYNC", cfg.SkipToday)
		}
		if task.ImportEnabled && task.ImportTime == currentTime {
			s.launch(currentDate, task, "IMPORT", cfg.SkipToday)
		}
	}
}

func (s *AutoTaskScheduler) launch(date string, task settings.AutoTaskSetting, kind string, skipToday bool) {
	scheduledTime := task.ImportTime
	if kind == "SYNC" {
		scheduledTime = task.SyncTime
	}
	key := date + "|" + task.ServerID + "|" + kind + "|" + scheduledTime
	s.mu.Lock()
	if s.lastRun[key] == scheduledTime || s.running[key] {
		s.mu.Unlock()
		return
	}
	s.running[key] = true
	s.lastRun[key] = scheduledTime
	s.mu.Unlock()

	go func() {
		defer func() {
			s.mu.Lock()
			delete(s.running, key)
			s.mu.Unlock()
		}()
		if kind == "SYNC" {
			s.runSync(task, skipToday)
			return
		}
		s.runImport(task, skipToday)
	}()
}

func (s *AutoTaskScheduler) runSync(task settings.AutoTaskSetting, skipToday bool) {
	label := task.ServerName + "：自动复制 CSV"
	ctx := context.Background()
	s.recordTask(ctx, task.ServerID, task.ServerName, "SYNC", label, "RUNNING", label+" 开始", nil)
	result, err := s.importer.SyncFilesFromConfiguredSource(ctx, task.ServerID, skipToday, nil)
	if err != nil {
		s.recordTask(ctx, task.ServerID, task.ServerName, "SYNC", label, "FAILED", label+" 失败："+err.Error(), nil)
		return
	}
	message := label + " 完成"
	status := "FINISHED"
	if result.FailedFiles > 0 {
		message = label + " 完成，但存在失败文件"
		status = "FINISHED_WITH_ERRORS"
	}
	s.recordTask(ctx, task.ServerID, task.ServerName, "SYNC", label, status, message, &result)
}

func (s *AutoTaskScheduler) runImport(task settings.AutoTaskSetting, skipToday bool) {
	label := task.ServerName + "：自动解析入库"
	ctx := context.Background()
	s.recordTask(ctx, task.ServerID, task.ServerName, "IMPORT", label, "RUNNING", label+" 开始", nil)
	result, err := s.importer.ImportFromConfiguredSource(ctx, task.ServerID, skipToday, nil)
	if err != nil {
		s.recordTask(ctx, task.ServerID, task.ServerName, "IMPORT", label, "FAILED", label+" 失败："+err.Error(), nil)
		return
	}
	message := label + " 完成"
	status := "FINISHED"
	if result.FailedFiles > 0 {
		message = label + " 完成，但存在失败文件"
		status = "FINISHED_WITH_ERRORS"
	}
	s.recordTask(ctx, task.ServerID, task.ServerName, "IMPORT", label, status, message, &result)
}

func (s *AutoTaskScheduler) recordConfig(ctx context.Context, message string) {
	s.recordTask(ctx, "", "系统", "CONFIG", "自动任务配置", "INFO", message, nil)
}

func (s *AutoTaskScheduler) recordTask(ctx context.Context, serverID, serverName, taskType, label, status, message string, result *ImportRunResult) {
	var details string
	scanned, success, skipped, failed := 0, 0, 0, 0
	if result != nil {
		scanned = result.ScannedFiles
		success = result.ImportedFiles
		skipped = result.SkippedFiles
		failed = result.FailedFiles
		details = formatAutoTaskFileDetails(result.Files)
	}
	_, err := s.importer.db.ExecContext(ctx, `
		insert into auto_task_logs (
			created_at, server_id, server_name, task_type, task_label, status, message, file_details,
			scanned_files, success_files, skipped_files, failed_files
		)
		values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, time.Now().UTC(), serverID, serverName, taskType, label, status, message, details, scanned, success, skipped, failed)
	if err != nil {
		s.logger.Warn("auto task log write failed", "error", err)
	}
}
