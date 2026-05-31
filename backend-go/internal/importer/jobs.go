package importer

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"path/filepath"
	"sync"
	"time"

	"player-stats-backend-go/internal/auth"
)

type JobService struct {
	importer    *Service
	mu          sync.Mutex
	running     bool
	syncRunning bool
	jobs        map[string]*jobState
	syncJobs    map[string]*jobState
}

func NewJobService(importer *Service) *JobService {
	return &JobService{
		importer: importer,
		jobs:     map[string]*jobState{},
		syncJobs: map[string]*jobState{},
	}
}

func (s *JobService) StartImport(ctx context.Context, serverID string, skipToday bool) (ImportJobView, error) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return ImportJobView{}, auth.NewHTTPError(409, "已有解析任务正在运行，请等待完成后再开始新的任务")
	}
	s.running = true
	s.mu.Unlock()

	files, err := s.importer.ListLocalImportFiles(ctx, serverID)
	if err != nil {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
		return ImportJobView{}, err
	}
	state := newJobState(files)
	s.mu.Lock()
	s.jobs[state.jobID] = state
	s.mu.Unlock()

	go s.runImport(state, serverID, skipToday)
	return state.view(), nil
}

func (s *JobService) GetJob(jobID string) (ImportJobView, error) {
	s.mu.Lock()
	state := s.jobs[jobID]
	s.mu.Unlock()
	if state == nil {
		return ImportJobView{}, auth.NewHTTPError(404, "解析任务不存在")
	}
	return state.view(), nil
}

func (s *JobService) StartSync(ctx context.Context, serverID string, skipToday bool) (ImportJobView, error) {
	s.mu.Lock()
	if s.syncRunning {
		s.mu.Unlock()
		return ImportJobView{}, auth.NewHTTPError(409, "已有复制任务正在运行，请等待完成后再开始新的任务")
	}
	s.syncRunning = true
	s.mu.Unlock()

	state := newJobState(nil)
	s.mu.Lock()
	s.syncJobs[state.jobID] = state
	s.mu.Unlock()

	go s.runSync(state, serverID, skipToday)
	return state.view(), nil
}

func (s *JobService) GetSyncJob(jobID string) (ImportJobView, error) {
	s.mu.Lock()
	state := s.syncJobs[jobID]
	s.mu.Unlock()
	if state == nil {
		return ImportJobView{}, auth.NewHTTPError(404, "复制任务不存在")
	}
	return state.view(), nil
}

func (s *JobService) runImport(state *jobState, serverID string, skipToday bool) {
	state.markRunning()
	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()
	result, err := s.importer.ImportFromConfiguredSource(context.Background(), serverID, skipToday, state)
	if err != nil {
		state.markFailed(err.Error())
		return
	}
	state.markFinished(result)
}

func (s *JobService) runSync(state *jobState, serverID string, skipToday bool) {
	state.markRunning()
	defer func() {
		s.mu.Lock()
		s.syncRunning = false
		s.mu.Unlock()
	}()
	result, err := s.importer.SyncFilesFromConfiguredSource(context.Background(), serverID, skipToday, state)
	if err != nil {
		state.markFailed(err.Error())
		return
	}
	state.markFinished(result)
}

type jobState struct {
	mu            sync.Mutex
	jobID         string
	startedAt     time.Time
	status        string
	finishedAt    *time.Time
	scannedFiles  int
	importedFiles int
	skippedFiles  int
	failedFiles   int
	message       string
	order         []string
	files         map[string]*mutableFile
}

func newJobState(initialFiles []ImportFileStatus) *jobState {
	state := &jobState{
		jobID:     newJobID(),
		startedAt: time.Now().UTC(),
		status:    "PENDING",
		files:     map[string]*mutableFile{},
	}
	for _, file := range initialFiles {
		key := fileKey(file.ServerID, file.RemotePath)
		state.order = append(state.order, key)
		state.files[key] = pendingMutableFile(file)
	}
	state.scannedFiles = len(initialFiles)
	return state
}

func (s *jobState) FileStarted(serverID, serverName string, file RemoteLogFile) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := fileKey(serverID, file.Path)
	item := s.ensureFile(key, func() *mutableFile {
		return &mutableFile{
			serverID:   serverID,
			serverName: serverName,
			remotePath: file.Path,
			fileName:   file.FileName,
			fileSize:   file.Size,
			status:     "PENDING",
			success:    true,
		}
	})
	now := time.Now().UTC()
	item.status = "RUNNING"
	item.message = "正在解析"
	item.startedAt = &now
	item.finishedAt = nil
}

func (s *jobState) FileFinished(result FileImportResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := fileKey(result.ServerID, result.RemotePath)
	item := s.ensureFile(key, func() *mutableFile {
		return &mutableFile{
			serverID:   result.ServerID,
			serverName: result.ServerName,
			remotePath: result.RemotePath,
			fileName:   filepath.Base(result.RemotePath),
			status:     "PENDING",
			success:    true,
		}
	})
	now := time.Now().UTC()
	item.status = result.Status
	item.success = result.Success
	item.rowCount = result.RowCount
	item.ignoredCount = result.IgnoredCount
	item.message = result.Message
	if item.startedAt == nil {
		item.startedAt = &now
	}
	item.finishedAt = &now
}

func (s *jobState) markRunning() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status = "RUNNING"
}

func (s *jobState) markFinished(result ImportRunResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	status := "FINISHED"
	if result.FailedFiles > 0 {
		status = "FINISHED_WITH_ERRORS"
	}
	s.status = status
	s.finishedAt = &result.FinishedAt
	s.scannedFiles = result.ScannedFiles
	s.importedFiles = result.ImportedFiles
	s.skippedFiles = result.SkippedFiles
	s.failedFiles = result.FailedFiles
}

func (s *jobState) markFailed(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	s.status = "FAILED"
	s.message = message
	s.finishedAt = &now
}

func (s *jobState) view() ImportJobView {
	s.mu.Lock()
	defer s.mu.Unlock()
	files := make([]ImportJobFileView, 0, len(s.order))
	for _, key := range s.order {
		files = append(files, s.files[key].view())
	}
	return ImportJobView{
		JobID:         s.jobID,
		Status:        s.status,
		StartedAt:     s.startedAt,
		FinishedAt:    s.finishedAt,
		ScannedFiles:  s.scannedFiles,
		ImportedFiles: s.importedFiles,
		SkippedFiles:  s.skippedFiles,
		FailedFiles:   s.failedFiles,
		Message:       s.message,
		Files:         files,
	}
}

func (s *jobState) ensureFile(key string, create func() *mutableFile) *mutableFile {
	item := s.files[key]
	if item != nil {
		return item
	}
	item = create()
	s.files[key] = item
	s.order = append(s.order, key)
	return item
}

type mutableFile struct {
	serverID     string
	serverName   string
	remotePath   string
	fileName     string
	fileSize     int64
	status       string
	success      bool
	rowCount     int
	ignoredCount int
	message      string
	startedAt    *time.Time
	finishedAt   *time.Time
}

func pendingMutableFile(file ImportFileStatus) *mutableFile {
	return &mutableFile{
		serverID:   file.ServerID,
		serverName: file.ServerName,
		remotePath: file.RemotePath,
		fileName:   file.FileName,
		fileSize:   file.FileSize,
		status:     "PENDING",
		success:    true,
	}
}

func (f *mutableFile) view() ImportJobFileView {
	return ImportJobFileView{
		ServerID:     f.serverID,
		ServerName:   f.serverName,
		RemotePath:   f.remotePath,
		FileName:     f.fileName,
		FileSize:     f.fileSize,
		Status:       f.status,
		Success:      f.success,
		RowCount:     f.rowCount,
		IgnoredCount: f.ignoredCount,
		Message:      f.message,
		StartedAt:    f.startedAt,
		FinishedAt:   f.finishedAt,
	}
}

func fileKey(serverID, remotePath string) string {
	return serverID + "\n" + remotePath
}

func newJobID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return hex.EncodeToString([]byte(time.Now().Format(time.RFC3339Nano)))
	}
	return hex.EncodeToString(bytes[:])
}
