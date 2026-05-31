package importer

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"strings"
	"time"

	"player-stats-backend-go/internal/catalog"
)

type statKey struct {
	statDate   time.Time
	playerName string
}

type actionCounts struct {
	broken int64
	placed int64
}

type typedPlayerKey struct {
	playerName string
	typ        string
}

type milestoneValue struct {
	firstSeenAt time.Time
	detail      string
}

type parsedLogFile struct {
	stats             map[statKey]actionCounts
	firstSeenByPlayer map[string]time.Time
	oreCounts         map[typedPlayerKey]int64
	woodCounts        map[typedPlayerKey]int64
	saplingCounts     map[typedPlayerKey]int64
	milestones        map[typedPlayerKey]milestoneValue
	rowCount          int
	ignoredCount      int
	contentHash       string
}

type parser struct {
	location *time.Location
}

func newParser(location *time.Location) parser {
	return parser{location: location}
}

func (p parser) parse(reader io.Reader) (parsedLogFile, error) {
	digest := sha256.New()
	tee := io.TeeReader(reader, digest)
	buffered := bufio.NewReaderSize(tee, 1024*1024)

	parsed := parsedLogFile{
		stats:             map[statKey]actionCounts{},
		firstSeenByPlayer: map[string]time.Time{},
		oreCounts:         map[typedPlayerKey]int64{},
		woodCounts:        map[typedPlayerKey]int64{},
		saplingCounts:     map[typedPlayerKey]int64{},
		milestones:        map[typedPlayerKey]milestoneValue{},
	}

	for {
		line, err := buffered.ReadString('\n')
		if len(line) > 0 {
			p.parseLine(strings.TrimRight(line, "\r\n"), &parsed)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return parsedLogFile{}, err
		}
	}
	parsed.contentHash = hex.EncodeToString(digest.Sum(nil))
	return parsed, nil
}

func (p parser) parseLine(line string, parsed *parsedLogFile) {
	if strings.TrimSpace(line) == "" {
		return
	}
	columns := splitPrefix(line)
	if isHeader(columns) {
		return
	}
	parsed.rowCount++

	happenedAt, ok := parseDateTime(columns, p.location)
	if ok {
		player := playerName(columns)
		if player != "" {
			if current, exists := parsed.firstSeenByPlayer[player]; !exists || happenedAt.Before(current) {
				parsed.firstSeenByPlayer[player] = happenedAt
			}
		}
	}
	p.collectBlockDetails(columns, parsed)

	record, ok := p.parseActionRecord(columns)
	if !ok {
		parsed.ignoredCount++
		return
	}
	key := statKey{statDate: record.statDate, playerName: record.playerName}
	counts := parsed.stats[key]
	if record.action == "DESTROY_BLOCK" {
		counts.broken++
	} else if record.action == "PLACE_BLOCK" {
		counts.placed++
	}
	parsed.stats[key] = counts
}

type actionRecord struct {
	statDate   time.Time
	playerName string
	action     string
}

func (p parser) parseActionRecord(columns []string) (actionRecord, bool) {
	if len(columns) < 4 {
		return actionRecord{}, false
	}
	statDate, ok := parseDate(stripBOM(columns[0]), p.location)
	if !ok {
		return actionRecord{}, false
	}
	player := playerName(columns)
	if player == "" {
		return actionRecord{}, false
	}
	action := parseAction(columns[3])
	if action == "" {
		return actionRecord{}, false
	}
	return actionRecord{statDate: statDate, playerName: player, action: action}, true
}

func (p parser) collectBlockDetails(columns []string, parsed *parsedLogFile) {
	if len(columns) < 13 {
		return
	}
	action := parseAction(columns[3])
	if action == "" {
		return
	}
	happenedAt, ok := parseDateTime(columns, p.location)
	if !ok {
		return
	}
	player := playerName(columns)
	if player == "" {
		return
	}
	blockID := columns[12]
	if action == "DESTROY_BLOCK" {
		if oreType, ok := catalog.OreTypeFromBlock(blockID); ok {
			parsed.oreCounts[typedPlayerKey{playerName: player, typ: oreType}]++
		}
		if woodType, ok := catalog.WoodTypeFromBlock(blockID); ok {
			parsed.woodCounts[typedPlayerKey{playerName: player, typ: woodType}]++
			mergeMilestone(parsed.milestones, typedPlayerKey{playerName: player, typ: "FIRST_WOOD"}, milestoneValue{
				firstSeenAt: happenedAt,
				detail:      catalog.WoodLabel(woodType),
			})
		}
		if milestoneType, ok := catalog.MilestoneTypeFromDestroyedBlock(blockID); ok && milestoneType != "FIRST_WOOD" {
			mergeMilestone(parsed.milestones, typedPlayerKey{playerName: player, typ: milestoneType}, milestoneValue{firstSeenAt: happenedAt})
		}
		return
	}
	if action == "PLACE_BLOCK" {
		if saplingType, ok := catalog.SaplingTypeFromBlock(blockID); ok {
			parsed.saplingCounts[typedPlayerKey{playerName: player, typ: saplingType}]++
		}
	}
}

func mergeMilestone(items map[typedPlayerKey]milestoneValue, key typedPlayerKey, value milestoneValue) {
	current, exists := items[key]
	if !exists || value.firstSeenAt.Before(current.firstSeenAt) {
		items[key] = value
	}
}

func splitPrefix(line string) []string {
	parts := strings.SplitN(line, ",", 14)
	for index := range parts {
		parts[index] = strings.TrimSpace(parts[index])
	}
	return parts
}

func isHeader(columns []string) bool {
	if len(columns) < 4 {
		return false
	}
	first := strings.TrimSpace(stripBOM(columns[0]))
	if _, ok := parseDate(first, time.Local); ok {
		return false
	}
	third := strings.TrimSpace(columns[2])
	fourth := strings.TrimSpace(columns[3])
	return strings.EqualFold(first, "date") ||
		strings.Contains(first, "日期") ||
		strings.Contains(third, "玩家") ||
		strings.Contains(third, "鍚") ||
		strings.Contains(fourth, "行为") ||
		strings.Contains(fourth, "琛")
}

func parseDateTime(columns []string, location *time.Location) (time.Time, bool) {
	if len(columns) < 2 {
		return time.Time{}, false
	}
	date, ok := parseDate(stripBOM(columns[0]), location)
	if !ok {
		return time.Time{}, false
	}
	clock, err := time.Parse("15:04:05", strings.TrimSpace(columns[1]))
	if err != nil {
		return time.Time{}, false
	}
	return time.Date(date.Year(), date.Month(), date.Day(), clock.Hour(), clock.Minute(), clock.Second(), 0, location), true
}

func parseDate(value string, location *time.Location) (time.Time, bool) {
	parsed, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(value), location)
	if err != nil {
		return time.Time{}, false
	}
	return parsed, true
}

func playerName(columns []string) string {
	if len(columns) < 3 {
		return ""
	}
	player := strings.TrimSpace(columns[2])
	if player == "" || player == "-" {
		return ""
	}
	return player
}

func parseAction(raw string) string {
	normalized := strings.TrimSpace(raw)
	switch normalized {
	case "破坏方块", "鐮村潖鏂瑰潡":
		return "DESTROY_BLOCK"
	case "放置方块", "鏀剧疆鏂瑰潡":
		return "PLACE_BLOCK"
	default:
		return ""
	}
}

func stripBOM(value string) string {
	return strings.TrimPrefix(value, "\ufeff")
}
