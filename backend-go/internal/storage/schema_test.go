package storage

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
)

const innodbMaxIndexBytes = 3072

func TestImportedLogFilePathIndexUsesHash(t *testing.T) {
	schema := strings.Join(schemaStatements, "\n")
	if !strings.Contains(schema, "remote_path_hash CHAR(64) NOT NULL") {
		t.Fatal("imported_server_log_files should store a fixed-size remote_path_hash")
	}
	if strings.Contains(schema, "remote_path(768)") {
		t.Fatal("remote_path(768) exceeds MySQL utf8mb4 key length limits when combined with server_id")
	}
	if !strings.Contains(schema, "UNIQUE KEY uk_imported_server_log_files_source_path_hash (server_id, remote_path_hash)") {
		t.Fatal("imported_server_log_files should deduplicate by server_id and remote_path_hash")
	}
}

func TestSchemaIndexesFitMySQLUtf8mb4Limit(t *testing.T) {
	for _, statement := range schemaStatements {
		table := tableName(statement)
		columns := columnByteLengths(statement)
		for _, index := range indexParts(statement) {
			total := 0
			for _, part := range index.parts {
				length, ok := columns[part.column]
				if !ok {
					t.Fatalf("%s.%s references unknown column %s", table, index.name, part.column)
				}
				if part.prefix > 0 {
					length = part.prefix * 4
				}
				total += length
			}
			if total > innodbMaxIndexBytes {
				t.Fatalf("%s.%s uses %d bytes, over MySQL InnoDB limit %d", table, index.name, total, innodbMaxIndexBytes)
			}
		}
	}
}

type schemaIndex struct {
	name  string
	parts []indexPart
}

type indexPart struct {
	column string
	prefix int
}

func tableName(statement string) string {
	match := regexp.MustCompile(`(?i)CREATE TABLE IF NOT EXISTS\s+([a-z0-9_]+)`).FindStringSubmatch(statement)
	if len(match) < 2 {
		return "unknown"
	}
	return match[1]
}

func columnByteLengths(statement string) map[string]int {
	result := map[string]int{}
	for _, raw := range strings.Split(statement, "\n") {
		line := strings.TrimSpace(strings.TrimSuffix(raw, ","))
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		column := strings.Trim(fields[0], "`")
		if strings.EqualFold(column, "PRIMARY") || strings.EqualFold(column, "UNIQUE") || strings.EqualFold(column, "KEY") {
			continue
		}
		result[column] = mysqlTypeBytes(fields[1])
	}
	return result
}

func mysqlTypeBytes(typ string) int {
	upper := strings.ToUpper(typ)
	switch {
	case strings.HasPrefix(upper, "VARCHAR("), strings.HasPrefix(upper, "CHAR("):
		return typeLength(upper) * 4
	case strings.HasPrefix(upper, "BIGINT"):
		return 8
	case strings.HasPrefix(upper, "INT"):
		return 4
	case strings.HasPrefix(upper, "BOOLEAN"):
		return 1
	case strings.HasPrefix(upper, "DATE"):
		return 3
	case strings.HasPrefix(upper, "DATETIME"):
		return 8
	default:
		return 0
	}
}

func typeLength(typ string) int {
	match := regexp.MustCompile(`\((\d+)\)`).FindStringSubmatch(typ)
	if len(match) < 2 {
		return 0
	}
	value, _ := strconv.Atoi(match[1])
	return value
}

func indexParts(statement string) []schemaIndex {
	var indexes []schemaIndex
	for _, raw := range strings.Split(statement, "\n") {
		line := strings.TrimSpace(strings.TrimSuffix(raw, ","))
		upper := strings.ToUpper(line)
		if !strings.HasPrefix(upper, "PRIMARY KEY") && !strings.HasPrefix(upper, "UNIQUE KEY") && !strings.HasPrefix(upper, "KEY ") {
			continue
		}
		name := "PRIMARY"
		if !strings.HasPrefix(upper, "PRIMARY KEY") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				name = strings.Trim(fields[1], "`")
			}
		}
		open := strings.Index(line, "(")
		close := strings.LastIndex(line, ")")
		if open < 0 || close <= open {
			continue
		}
		var parts []indexPart
		for _, rawPart := range strings.Split(line[open+1:close], ",") {
			part := strings.TrimSpace(rawPart)
			match := regexp.MustCompile("`?([a-zA-Z0-9_]+)`?(?:\\((\\d+)\\))?").FindStringSubmatch(part)
			if len(match) < 2 {
				continue
			}
			prefix := 0
			if len(match) >= 3 && match[2] != "" {
				prefix, _ = strconv.Atoi(match[2])
			}
			parts = append(parts, indexPart{column: match[1], prefix: prefix})
		}
		indexes = append(indexes, schemaIndex{name: name, parts: parts})
	}
	return indexes
}
