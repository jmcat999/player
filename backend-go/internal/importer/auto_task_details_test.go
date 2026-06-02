package importer

import "testing"

func TestFormatAutoTaskFileDetailsMatchesJavaStyleSummary(t *testing.T) {
	files := []FileImportResult{
		{RemotePath: "/data/synced-logs/sub/player_actions_2025-10-24.csv", Status: "COPIED", Success: true},
		{RemotePath: "/data/synced-logs/sub/player_actions_2025-10-25.csv", Status: "COPIED", Success: true},
		{RemotePath: "/data/synced-logs/sub/player_actions_2025-10-26.csv", Status: "FAILED", Success: false, Message: "permission denied when opening remote file"},
	}

	got := formatAutoTaskFileDetails(files)
	want := "复制：2025-10-24 player_actions_2025-10-24.csv，2025-10-25 player_actions_2025-10-25.csv\n失败：2025-10-26 player_actions_2025-10-26.csv(permission denied when o)"
	if got != want {
		t.Fatalf("formatAutoTaskFileDetails() = %q, want %q", got, want)
	}
}

func TestFormatAutoTaskFileDetailsHidesSkippedOnlyRuns(t *testing.T) {
	files := []FileImportResult{
		{ServerID: "sub", ServerName: "2服", RemotePath: "/data/synced-logs/sub/player_actions_2025-10-24.csv", Status: "SKIPPED", Success: true, Message: "文件大小和修改时间未变化"},
		{ServerID: "sub", ServerName: "2服", RemotePath: "/data/synced-logs/sub/player_actions_2025-10-25.csv", Status: "SKIPPED", Success: true, Message: "文件大小和修改时间未变化"},
	}

	got := formatAutoTaskFileDetails(files)
	if got != "没有新处理或失败文件" {
		t.Fatalf("formatAutoTaskFileDetails() = %q", got)
	}
}

func TestNormalizeAutoTaskFileDetailsConvertsLegacyJSON(t *testing.T) {
	legacy := `[{"serverId":"sub","serverName":"2服","remotePath":"/data/synced-logs/sub/player_actions_2025-10-24.csv","status":"SKIPPED","success":true,"rowCount":0,"ignoredCount":0,"message":"文件大小和修改时间未变化"},{"serverId":"sub","serverName":"2服","remotePath":"/data/synced-logs/sub/player_actions_2025-10-25.csv","status":"COPIED","success":true,"rowCount":0,"ignoredCount":0,"message":""}]`

	got := normalizeAutoTaskFileDetails(legacy)
	want := "复制：2025-10-25 player_actions_2025-10-25.csv"
	if got != want {
		t.Fatalf("normalizeAutoTaskFileDetails() = %q, want %q", got, want)
	}
}
