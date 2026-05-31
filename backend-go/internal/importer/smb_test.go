package importer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"player-stats-backend-go/internal/config"
)

func TestRemoteFileStatusComparesLocalArchive(t *testing.T) {
	location := time.FixedZone("CST", 8*60*60)
	dir := t.TempDir()
	service := NewService(nil, config.Config{Location: location, SkipToday: true}, nil)
	source := config.Source{ID: "sub", Name: "2服", Directory: dir, FileGlob: "player_actions_*.csv", Enabled: true}
	modified := time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)
	remote := RemoteLogFile{
		Path:         "logs/sub/player_actions_2026-04-24.csv",
		FileName:     "player_actions_2026-04-24.csv",
		Size:         6,
		LastModified: modified,
		SourcePath:   "logs/sub/player_actions_2026-04-24.csv",
	}
	archived := toArchivedSMBFile(source, remote)

	pending := service.remoteFileStatus(source, remote, archived)
	if pending.Status != "REMOTE" {
		t.Fatalf("empty archive status = %s, want REMOTE", pending.Status)
	}
	if pending.RemotePath != remote.Path {
		t.Fatalf("remote path = %q, want %q", pending.RemotePath, remote.Path)
	}
	if pending.LocalPath != archived.Path {
		t.Fatalf("local path = %q, want %q", pending.LocalPath, archived.Path)
	}

	if err := os.WriteFile(archived.Path, []byte("abc"), 0644); err != nil {
		t.Fatal(err)
	}
	changed := service.remoteFileStatus(source, remote, archived)
	if changed.Status != "CHANGED" {
		t.Fatalf("size mismatch status = %s, want CHANGED", changed.Status)
	}

	if err := os.WriteFile(archived.Path, []byte("abcdef"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(archived.Path, modified, modified); err != nil {
		t.Fatal(err)
	}
	copied := service.remoteFileStatus(source, remote, archived)
	if copied.Status != "COPIED" {
		t.Fatalf("matching archive status = %s, want COPIED", copied.Status)
	}
}

func TestWriteArchiveFileIgnoresStaleFixedTmpFile(t *testing.T) {
	dir := t.TempDir()
	localPath := filepath.Join(dir, "player_actions_2026-04-24.csv")
	staleTmp := localPath + ".tmp"
	if err := os.WriteFile(staleTmp, []byte("stale"), 0400); err != nil {
		t.Fatal(err)
	}
	modified := time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)
	if err := writeArchiveFile(strings.NewReader("fresh"), localPath, modified); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(staleTmp); !os.IsNotExist(err) {
		t.Fatalf("stale fixed tmp still exists or stat failed: %v", err)
	}
	content, err := os.ReadFile(localPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "fresh" {
		t.Fatalf("content = %q, want fresh", content)
	}
	info, err := os.Stat(localPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.ModTime().UTC().UnixMilli() != modified.UnixMilli() {
		t.Fatalf("mtime = %s, want %s", info.ModTime().UTC(), modified)
	}
}
