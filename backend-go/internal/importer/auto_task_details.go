package importer

import (
	"encoding/json"
	"path"
	"regexp"
	"strconv"
	"strings"
)

func normalizeAutoTaskFileDetails(details string) string {
	details = strings.TrimSpace(details)
	if details == "" || !strings.HasPrefix(details, "[") {
		return details
	}
	var files []FileImportResult
	if err := json.Unmarshal([]byte(details), &files); err != nil {
		return details
	}
	return formatAutoTaskFileDetails(files)
}

func formatAutoTaskFileDetails(files []FileImportResult) string {
	if len(files) == 0 {
		return ""
	}
	var details strings.Builder
	appendAutoTaskSummary(&details, files, "IMPORTED", "入库", 8)
	appendAutoTaskSummary(&details, files, "COPIED", "复制", 8)
	appendAutoTaskSummary(&details, files, "FAILED", "失败", 6)
	summary := strings.TrimSpace(details.String())
	if summary == "" {
		return "没有新处理或失败文件"
	}
	return summary
}

func appendAutoTaskSummary(details *strings.Builder, files []FileImportResult, status, label string, limit int) {
	matched := make([]FileImportResult, 0)
	for _, file := range files {
		if file.Status == status {
			matched = append(matched, file)
		}
	}
	if len(matched) == 0 {
		return
	}
	if details.Len() > 0 {
		details.WriteString("\n")
	}
	details.WriteString(label)
	details.WriteString("：")
	for index, file := range matched {
		if index >= limit {
			break
		}
		if index > 0 {
			details.WriteString("，")
		}
		details.WriteString(shortAutoTaskFileResult(file))
	}
	if len(matched) > limit {
		details.WriteString("；省略 ")
		details.WriteString(strconv.Itoa(len(matched) - limit))
		details.WriteString(" 个")
	}
}

func shortAutoTaskFileResult(file FileImportResult) string {
	name := fileNameFromPath(file.RemotePath)
	datedName := fileDateFromName(name) + name
	if file.Status == "IMPORTED" && file.RowCount > 0 {
		return datedName + "(" + strconv.Itoa(file.RowCount) + "行)"
	}
	if file.Status == "FAILED" && strings.TrimSpace(file.Message) != "" {
		return datedName + "(" + truncateAutoTaskText(file.Message, 24) + ")"
	}
	return datedName
}

func fileNameFromPath(filePath string) string {
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return "-"
	}
	return path.Base(strings.ReplaceAll(filePath, "\\", "/"))
}

func fileDateFromName(fileName string) string {
	match := regexp.MustCompile(`.*?(\d{4}-\d{2}-\d{2}).*`).FindStringSubmatch(fileName)
	if len(match) < 2 {
		return ""
	}
	return match[1] + " "
}

func truncateAutoTaskText(value string, maxLength int) string {
	if maxLength <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= maxLength {
		return value
	}
	return string(runes[:maxLength])
}
