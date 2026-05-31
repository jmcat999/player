package importer

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/hirochachacha/go-smb2"

	"player-stats-backend-go/internal/apitype"
	"player-stats-backend-go/internal/auth"
	"player-stats-backend-go/internal/config"
)

func (s *Service) listRemoteSMBFiles(ctx context.Context, requestedServerID string) ([]ImportFileStatus, error) {
	result := make([]ImportFileStatus, 0)
	err := s.withSMBShare(ctx, func(share *smb2.Share) error {
		for _, source := range s.selectedSources(ctx, requestedServerID) {
			sourceConfig, err := s.sourceSyncSettings(ctx, source)
			if err != nil {
				return err
			}
			files, err := listSMBFiles(share, sourceConfig.directory, "", false)
			if err != nil {
				result = append(result, ImportFileStatus{
					ServerID:     source.ID,
					ServerName:   source.Name,
					RemotePath:   sourceConfig.directory,
					FileName:     sourceConfig.directory,
					LastModified: time.Unix(0, 0).UTC(),
					Status:       "FAILED",
					Message:      err.Error(),
				})
				continue
			}
			for _, file := range limitFiles(files, s.cfg.MaxFilesPerRun) {
				result = append(result, s.remoteFileStatus(source, file, toArchivedSMBFile(source, file)))
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Service) syncFilesFromSMBSource(ctx context.Context, requestedServerID string, skipToday bool, listener ProgressListener) (ImportRunResult, error) {
	startedAt := time.Now().UTC()
	results := make([]FileImportResult, 0)
	err := s.withSMBShare(ctx, func(share *smb2.Share) error {
		for _, source := range s.selectedSources(ctx, requestedServerID) {
			sourceConfig, err := s.sourceSyncSettings(ctx, source)
			if err != nil {
				return err
			}
			remoteFiles, err := listSMBFiles(share, sourceConfig.directory, sourceConfig.fileGlob, sourceConfig.recursive)
			if err != nil {
				results = appendResult(results, listener, failedResult(source.ID, source.Name, sourceConfig.directory, err.Error()))
				continue
			}
			archivedFiles := make([]RemoteLogFile, 0, len(remoteFiles))
			for _, file := range remoteFiles {
				archivedFiles = append(archivedFiles, toArchivedSMBFile(source, file))
			}
			sort.Slice(archivedFiles, func(i, j int) bool { return archivedFiles[i].Path < archivedFiles[j].Path })
			for _, file := range limitFiles(archivedFiles, s.cfg.MaxFilesPerRun) {
				if listener != nil {
					listener.FileStarted(source.ID, source.Name, file)
				}
				logDate := extractLogDate(file.FileName, s.cfg.Location)
				effectiveDate := logDate
				if effectiveDate == nil {
					value := dateOnly(file.LastModified.In(s.cfg.Location), s.cfg.Location)
					effectiveDate = &value
				}
				today := dateOnly(time.Now().In(s.cfg.Location), s.cfg.Location)
				if skipToday && !effectiveDate.Before(today) {
					results = appendResult(results, listener, skippedResult(source.ID, source.Name, file.Path, "跳过当天或未来的日志文件"))
					continue
				}
				copied, err := copySMBFileToArchive(share, file)
				if err != nil {
					results = appendResult(results, listener, failedResult(source.ID, source.Name, file.Path, err.Error()))
					continue
				}
				if copied {
					results = appendResult(results, listener, copiedResult(source.ID, source.Name, file.Path))
				} else {
					results = appendResult(results, listener, skippedResult(source.ID, source.Name, file.Path, "本地 CSV 文件未变化"))
				}
			}
		}
		return nil
	})
	if err != nil {
		results = append(results, failedResult(requestedServerID, s.cfg.SourceName(requestedServerID), "", err.Error()))
	}
	return summarizeRun(startedAt, results), nil
}

type smbSourceSettings struct {
	directory string
	fileGlob  string
	recursive bool
}

func (s *Service) sourceSyncSettings(ctx context.Context, source config.Source) (smbSourceSettings, error) {
	result := smbSourceSettings{
		directory: strings.TrimSpace(s.cfg.SMBDirectory),
		fileGlob:  firstNonBlank(s.cfg.SMBFileGlob, source.FileGlob),
		recursive: s.cfg.SMBRecursive,
	}
	if sourceConfig, ok, err := s.settings.SourceByID(ctx, source.ID); err != nil {
		return result, err
	} else if ok {
		if strings.TrimSpace(sourceConfig.SMBDirectory) != "" {
			result.directory = strings.TrimSpace(sourceConfig.SMBDirectory)
		}
		if strings.TrimSpace(sourceConfig.SMBFileGlob) != "" {
			result.fileGlob = sourceConfig.SMBFileGlob
		}
		result.recursive = sourceConfig.SMBRecursive
	}
	result.directory = normalizeSMBPath(result.directory)
	if strings.TrimSpace(result.fileGlob) == "" {
		result.fileGlob = "player_actions_*.csv"
	}
	return result, nil
}

func (s *Service) withSMBShare(ctx context.Context, callback func(*smb2.Share) error) error {
	cfg, err := s.settings.SMBConnectionConfig(ctx)
	if err != nil {
		return err
	}
	if strings.TrimSpace(cfg.Host) == "" {
		return auth.NewHTTPError(400, "SMB 主机未配置，请在设置页面填写")
	}
	if strings.TrimSpace(cfg.Share) == "" {
		return auth.NewHTTPError(400, "SMB 共享名未配置，请在设置页面填写")
	}
	address := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	conn, err := (&net.Dialer{Timeout: 15 * time.Second}).DialContext(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("SMB 连接失败: %w", err)
	}
	defer conn.Close()

	dialer := &smb2.Dialer{Initiator: &smb2.NTLMInitiator{
		User:     cfg.Username,
		Password: cfg.Password,
		Domain:   cfg.Domain,
	}}
	session, err := dialer.DialContext(ctx, conn)
	if err != nil {
		return fmt.Errorf("SMB 认证失败: %w", err)
	}
	defer session.Logoff()

	share, err := session.WithContext(ctx).Mount(cfg.Share)
	if err != nil {
		return fmt.Errorf("SMB 共享挂载失败: %w", err)
	}
	defer share.Umount()
	return callback(share.WithContext(ctx))
}

func (s *Service) TestSMBConnection(ctx context.Context) (string, error) {
	cfg, err := s.settings.SMBConnectionConfig(ctx)
	if err != nil {
		return "", err
	}
	if err := s.withSMBShare(ctx, func(share *smb2.Share) error {
		_, err := share.ReadDir("")
		return err
	}); err != nil {
		return "", err
	}
	return fmt.Sprintf("连接成功：%s:%d/%s", cfg.Host, cfg.Port, cfg.Share), nil
}

func listSMBFiles(share *smb2.Share, directory, glob string, recursive bool) ([]RemoteLogFile, error) {
	directory = normalizeSMBPath(directory)
	infos, err := share.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("列出 SMB 目录失败 [%s]: %w", directory, err)
	}
	files := make([]RemoteLogFile, 0)
	for _, info := range infos {
		name := info.Name()
		if name == "." || name == ".." {
			continue
		}
		remotePath := joinSMBPath(directory, name)
		if info.IsDir() {
			if recursive {
				children, err := listSMBFiles(share, remotePath, glob, recursive)
				if err != nil {
					return nil, err
				}
				files = append(files, children...)
			}
			continue
		}
		if strings.TrimSpace(glob) != "" && !matchesGlob(name, glob) {
			continue
		}
		files = append(files, RemoteLogFile{
			Path:         remotePath,
			FileName:     name,
			Size:         info.Size(),
			LastModified: info.ModTime().UTC(),
			SourcePath:   remotePath,
		})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

func toArchivedSMBFile(source config.Source, remoteFile RemoteLogFile) RemoteLogFile {
	safeName := filepath.Base(remoteFile.FileName)
	localPath := filepath.Join(source.Directory, safeName)
	return RemoteLogFile{
		Path:         filepath.Clean(localPath),
		FileName:     safeName,
		Size:         remoteFile.Size,
		LastModified: remoteFile.LastModified,
		SourcePath:   remoteFile.SourcePath,
	}
}

func (s *Service) remoteFileStatus(source config.Source, remoteFile, archivedFile RemoteLogFile) ImportFileStatus {
	logDate := extractLogDate(remoteFile.FileName, s.cfg.Location)
	effectiveDate := logDate
	if effectiveDate == nil {
		value := dateOnly(remoteFile.LastModified.In(s.cfg.Location), s.cfg.Location)
		effectiveDate = &value
	}
	var apiLogDate *apitype.Date
	if logDate != nil {
		value := apitype.NewDate(*logDate)
		apiLogDate = &value
	}

	status := "REMOTE"
	message := "远程文件，待复制"
	today := dateOnly(time.Now().In(s.cfg.Location), s.cfg.Location)
	if s.cfg.SkipToday && !effectiveDate.Before(today) {
		status = "SKIPPED_TODAY"
		message = "跳过当天或未来的日志文件"
	} else if currentArchiveFile(archivedFile.Path, archivedFile) {
		status = "COPIED"
		message = "本地 CSV 文件未变化"
	} else if archiveExists(archivedFile.Path) {
		status = "CHANGED"
		message = "远程文件与本地 CSV 不一致，可重新复制"
	}

	return ImportFileStatus{
		ServerID:     source.ID,
		ServerName:   source.Name,
		RemotePath:   remoteFile.Path,
		LocalPath:    archivedFile.Path,
		FileName:     remoteFile.FileName,
		FileSize:     remoteFile.Size,
		LastModified: remoteFile.LastModified,
		LogDate:      apiLogDate,
		Status:       status,
		Message:      message,
	}
}

func copySMBFileToArchive(share *smb2.Share, file RemoteLogFile) (bool, error) {
	localPath, err := filepath.Abs(file.Path)
	if err != nil {
		return false, err
	}
	if currentArchiveFile(localPath, file) {
		return false, nil
	}
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return false, err
	}
	sourcePath := file.SourcePath
	if strings.TrimSpace(sourcePath) == "" {
		sourcePath = file.Path
	}
	remote, err := share.Open(sourcePath)
	if err != nil {
		return false, err
	}
	defer remote.Close()

	if err := writeArchiveFile(remote, localPath, file.LastModified); err != nil {
		return false, err
	}
	return true, nil
}

func writeArchiveFile(reader io.Reader, localPath string, lastModified time.Time) error {
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return err
	}
	_ = os.Remove(localPath + ".tmp")

	out, err := os.CreateTemp(filepath.Dir(localPath), filepath.Base(localPath)+".*.tmp")
	if err != nil {
		return err
	}
	tempPath := out.Name()
	renamed := false
	defer func() {
		if !renamed {
			_ = os.Remove(tempPath)
		}
	}()

	_, copyErr := io.Copy(out, reader)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	if err := replaceFile(tempPath, localPath); err != nil {
		return err
	}
	renamed = true
	_ = os.Chmod(localPath, 0644)
	_ = os.Chtimes(localPath, lastModified, lastModified)
	return nil
}

func replaceFile(tempPath, localPath string) error {
	if err := os.Rename(tempPath, localPath); err != nil {
		if runtime.GOOS != "windows" {
			return err
		}
		_ = os.Remove(localPath)
		return os.Rename(tempPath, localPath)
	}
	return nil
}

func currentArchiveFile(localPath string, file RemoteLogFile) bool {
	info, err := os.Stat(localPath)
	if err != nil || info.IsDir() {
		return false
	}
	return info.Size() == file.Size && info.ModTime().UTC().UnixMilli() == file.LastModified.UTC().UnixMilli()
}

func archiveExists(localPath string) bool {
	info, err := os.Stat(localPath)
	return err == nil && !info.IsDir()
}

func normalizeSMBPath(value string) string {
	value = strings.ReplaceAll(strings.TrimSpace(value), "\\", "/")
	value = strings.Trim(value, "/")
	if value == "" {
		return ""
	}
	return path.Clean(value)
}

func joinSMBPath(directory, name string) string {
	directory = normalizeSMBPath(directory)
	name = strings.Trim(name, "/\\")
	if directory == "." || directory == "" {
		return name
	}
	return directory + "/" + name
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
