package importer

import (
	"time"

	"player-stats-backend-go/internal/apitype"
)

type RemoteLogFile struct {
	Path         string    `json:"path"`
	FileName     string    `json:"fileName"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"lastModified"`
	SourcePath   string    `json:"-"`
}

type ImportFileStatus struct {
	ServerID     string        `json:"serverId"`
	ServerName   string        `json:"serverName"`
	RemotePath   string        `json:"remotePath"`
	FileName     string        `json:"fileName"`
	FileSize     int64         `json:"fileSize"`
	LastModified time.Time     `json:"lastModified"`
	LogDate      *apitype.Date `json:"logDate"`
	Imported     bool          `json:"imported"`
	ImportedAt   *time.Time    `json:"importedAt"`
	RowCount     int           `json:"rowCount"`
	IgnoredCount int           `json:"ignoredCount"`
	Status       string        `json:"status"`
	Message      string        `json:"message"`
}

type FileImportResult struct {
	ServerID     string `json:"serverId"`
	ServerName   string `json:"serverName"`
	RemotePath   string `json:"remotePath"`
	Status       string `json:"status"`
	Success      bool   `json:"success"`
	RowCount     int    `json:"rowCount"`
	IgnoredCount int    `json:"ignoredCount"`
	Message      string `json:"message"`
}

type ImportRunResult struct {
	StartedAt     time.Time          `json:"startedAt"`
	FinishedAt    time.Time          `json:"finishedAt"`
	ScannedFiles  int                `json:"scannedFiles"`
	ImportedFiles int                `json:"importedFiles"`
	SkippedFiles  int                `json:"skippedFiles"`
	FailedFiles   int                `json:"failedFiles"`
	Files         []FileImportResult `json:"files"`
}

type ImportJobView struct {
	JobID         string              `json:"jobId"`
	Status        string              `json:"status"`
	StartedAt     time.Time           `json:"startedAt"`
	FinishedAt    *time.Time          `json:"finishedAt"`
	ScannedFiles  int                 `json:"scannedFiles"`
	ImportedFiles int                 `json:"importedFiles"`
	SkippedFiles  int                 `json:"skippedFiles"`
	FailedFiles   int                 `json:"failedFiles"`
	Message       string              `json:"message"`
	Files         []ImportJobFileView `json:"files"`
}

type ImportJobFileView struct {
	ServerID     string     `json:"serverId"`
	ServerName   string     `json:"serverName"`
	RemotePath   string     `json:"remotePath"`
	FileName     string     `json:"fileName"`
	FileSize     int64      `json:"fileSize"`
	Status       string     `json:"status"`
	Success      bool       `json:"success"`
	RowCount     int        `json:"rowCount"`
	IgnoredCount int        `json:"ignoredCount"`
	Message      string     `json:"message"`
	StartedAt    *time.Time `json:"startedAt"`
	FinishedAt   *time.Time `json:"finishedAt"`
}

type DeleteImportRecordResult struct {
	ServerID   string `json:"serverId"`
	RemotePath string `json:"remotePath"`
	Deleted    bool   `json:"deleted"`
	Message    string `json:"message"`
}

type DeleteImportRecordsRequest struct {
	Files []ImportRecordKey `json:"files"`
}

type AutoTaskLogView struct {
	ID           int64     `json:"id"`
	CreatedAt    time.Time `json:"createdAt"`
	ServerID     string    `json:"serverId"`
	ServerName   string    `json:"serverName"`
	TaskType     string    `json:"taskType"`
	TaskLabel    string    `json:"taskLabel"`
	Status       string    `json:"status"`
	Message      string    `json:"message"`
	FileDetails  string    `json:"fileDetails"`
	ScannedFiles int       `json:"scannedFiles"`
	SuccessFiles int       `json:"successFiles"`
	SkippedFiles int       `json:"skippedFiles"`
	FailedFiles  int       `json:"failedFiles"`
}

type ImportRecordKey struct {
	ServerID   string `json:"serverId"`
	RemotePath string `json:"remotePath"`
}

func importedResult(serverID, serverName, remotePath string, rowCount, ignoredCount int) FileImportResult {
	return FileImportResult{ServerID: serverID, ServerName: serverName, RemotePath: remotePath, Status: "IMPORTED", Success: true, RowCount: rowCount, IgnoredCount: ignoredCount}
}

func copiedResult(serverID, serverName, remotePath string) FileImportResult {
	return FileImportResult{ServerID: serverID, ServerName: serverName, RemotePath: remotePath, Status: "COPIED", Success: true}
}

func skippedResult(serverID, serverName, remotePath, message string) FileImportResult {
	return FileImportResult{ServerID: serverID, ServerName: serverName, RemotePath: remotePath, Status: "SKIPPED", Success: true, Message: message}
}

func failedResult(serverID, serverName, remotePath, message string) FileImportResult {
	return FileImportResult{ServerID: serverID, ServerName: serverName, RemotePath: remotePath, Status: "FAILED", Success: false, Message: message}
}
