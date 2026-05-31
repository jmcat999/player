package importer

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"math"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"player-stats-backend-go/internal/apitype"
	"player-stats-backend-go/internal/auth"
	"player-stats-backend-go/internal/catalog"
	"player-stats-backend-go/internal/config"
)

const (
	minTurnMainTunnelBlocks  = 8
	minTurnSideTunnelBlocks  = 2
	maxTurnSideTunnelBlocks  = 5
	turnLookbackSeconds      = 180
	minTurnCorridorFit       = 0.95
	maxTurnDirectionDot      = 0.08715574274765817
	minDirectVeinDistance    = 8.0
	maxDirectVeinDistance    = 72.0
	minDirectVeinSeconds     = 10
	maxDirectVeinSeconds     = 120
	maxDirectVeinGapBlocks   = 6
	sameVeinDistance         = 4.25
	minShortStraightBlocks   = 5
	maxShortStraightBlocks   = 24
	maxShortStraightSeconds  = 90
	minShortStraightness     = 0.93
	miningSessionGapSeconds  = 480
	miningSessionEvents      = 20
	shortRareOreWindowSecond = 600
)

var tunnelBlocks = map[string]struct{}{
	"minecraft:stone":           {},
	"minecraft:deepslate":       {},
	"minecraft:tuff":            {},
	"minecraft:netherrack":      {},
	"minecraft:basalt":          {},
	"minecraft:smooth_basalt":   {},
	"minecraft:blackstone":      {},
	"minecraft:dripstone_block": {},
	"minecraft:calcite":         {},
	"minecraft:granite":         {},
	"minecraft:diorite":         {},
	"minecraft:andesite":        {},
	"minecraft:gravel":          {},
	"minecraft:sandstone":       {},
	"minecraft:red_sandstone":   {},
}

type XrayAnalysisRequest struct {
	ServerID   string `json:"serverId"`
	FromTime   string `json:"fromTime"`
	ToTime     string `json:"toTime"`
	PlayerName string `json:"playerName"`
	Dimension  string `json:"dimension"`
}

type XrayAnalysisView struct {
	JobID        string                 `json:"jobId"`
	ServerID     string                 `json:"serverId"`
	ServerName   string                 `json:"serverName"`
	Status       string                 `json:"status"`
	StartedAt    time.Time              `json:"startedAt"`
	FinishedAt   *time.Time             `json:"finishedAt"`
	FromTime     *apitype.LocalDateTime `json:"fromTime"`
	ToTime       *apitype.LocalDateTime `json:"toTime"`
	PlayerName   string                 `json:"playerName"`
	Dimension    string                 `json:"dimension"`
	ScannedFiles int                    `json:"scannedFiles"`
	ScannedRows  int64                  `json:"scannedRows"`
	FailedFiles  int                    `json:"failedFiles"`
	PlayerCount  int                    `json:"playerCount"`
	FindingCount int                    `json:"findingCount"`
	MaxRiskScore int                    `json:"maxRiskScore"`
	CurrentFile  string                 `json:"currentFile"`
	Message      string                 `json:"message"`
	Players      []XrayPlayerRiskView   `json:"players"`
}

type XrayPlayerRiskView struct {
	PlayerName                       string                 `json:"playerName"`
	RiskScore                        int                    `json:"riskScore"`
	RiskLevel                        string                 `json:"riskLevel"`
	MiningSessionCount               int                    `json:"miningSessionCount"`
	MiningSessionStart               *apitype.LocalDateTime `json:"miningSessionStart"`
	MiningSessionEnd                 *apitype.LocalDateTime `json:"miningSessionEnd"`
	MiningSessionBreaks              int64                  `json:"miningSessionBreaks"`
	MiningSessionUndergroundBreaks   int64                  `json:"miningSessionUndergroundBreaks"`
	MiningSessionOreBreaks           int64                  `json:"miningSessionOreBreaks"`
	MiningSessionRareOreBreaks       int64                  `json:"miningSessionRareOreBreaks"`
	MiningSessionDiamondOreBreaks    int64                  `json:"miningSessionDiamondOreBreaks"`
	MiningSessionAncientDebrisBreaks int64                  `json:"miningSessionAncientDebrisBreaks"`
	MiningSessionRareVeins           int                    `json:"miningSessionRareVeins"`
	TrackingEvidenceCount            int                    `json:"trackingEvidenceCount"`
	PeakRareOreWindowCount           int                    `json:"peakRareOreWindowCount"`
	PeakRareOreWindowStart           *apitype.LocalDateTime `json:"peakRareOreWindowStart"`
	PeakRareOreWindowEnd             *apitype.LocalDateTime `json:"peakRareOreWindowEnd"`
	PeakRareVeinWindowCount          int                    `json:"peakRareVeinWindowCount"`
	UndergroundRareOreRatio          float64                `json:"undergroundRareOreRatio"`
	Reasons                          []string               `json:"reasons"`
	Ores                             []XrayOreCount         `json:"ores"`
	RareOres                         []XrayOreCount         `json:"rareOres"`
	Evidence                         []XrayEvidenceView     `json:"evidence"`
	RareOreRows                      []LogQueryRow          `json:"rareOreRows"`
	AnalysisBreaks                   int64                  `json:"analysisBreaks"`
	AnalysisUndergroundBreaks        int64                  `json:"analysisUndergroundBreaks"`
	AnalysisOreBreaks                int64                  `json:"analysisOreBreaks"`
	AnalysisRareOreBreaks            int64                  `json:"analysisRareOreBreaks"`
	AnalysisDiamondOreBreaks         int64                  `json:"analysisDiamondOreBreaks"`
	AnalysisAncientDebrisBreaks      int64                  `json:"analysisAncientDebrisBreaks"`
	AnalysisRareVeins                int                    `json:"analysisRareVeins"`
	AnalysisTrackingEvidenceCount    int                    `json:"analysisTrackingEvidenceCount"`
	AnalysisPeakRareOreWindowCount   int                    `json:"analysisPeakRareOreWindowCount"`
	AnalysisPeakRareOreWindowStart   *apitype.LocalDateTime `json:"analysisPeakRareOreWindowStart"`
	AnalysisPeakRareOreWindowEnd     *apitype.LocalDateTime `json:"analysisPeakRareOreWindowEnd"`
	AnalysisPeakRareVeinWindowCount  int                    `json:"analysisPeakRareVeinWindowCount"`
	AnalysisUndergroundRareOreRatio  float64                `json:"analysisUndergroundRareOreRatio"`
	AnalysisOres                     []XrayOreCount         `json:"analysisOres"`
	AnalysisRareOres                 []XrayOreCount         `json:"analysisRareOres"`
}

type XrayEvidenceView struct {
	Type      string                 `json:"type"`
	Score     int                    `json:"score"`
	StartedAt *apitype.LocalDateTime `json:"startedAt"`
	EndedAt   *apitype.LocalDateTime `json:"endedAt"`
	Summary   string                 `json:"summary"`
	Rows      []LogQueryRow          `json:"rows"`
}

type XrayOreCount struct {
	OreType     string `json:"oreType"`
	DisplayName string `json:"displayName"`
	Count       int64  `json:"count"`
}

type XrayAnalysisService struct {
	importer *Service
	mu       sync.Mutex
	latest   map[string]*xrayState
	running  map[string]bool
}

func NewXrayAnalysisService(importer *Service) *XrayAnalysisService {
	return &XrayAnalysisService{importer: importer, latest: map[string]*xrayState{}, running: map[string]bool{}}
}

func (s *XrayAnalysisService) Start(ctx context.Context, request XrayAnalysisRequest) (XrayAnalysisView, error) {
	criteria, err := s.validateCriteria(request)
	if err != nil {
		return XrayAnalysisView{}, err
	}
	source, ok := s.importer.sourceByID(ctx, criteria.serverID)
	if !ok {
		return XrayAnalysisView{}, auth.NewHTTPError(404, "找不到服务器："+criteria.serverID)
	}
	s.mu.Lock()
	if s.running[criteria.serverID] {
		s.mu.Unlock()
		return XrayAnalysisView{}, auth.NewHTTPError(409, source.Name+" 的矿透分析正在运行")
	}
	s.running[criteria.serverID] = true
	state := newXrayState(source, criteria)
	s.latest[criteria.serverID] = state
	s.mu.Unlock()

	go s.run(state, source)
	return state.view(), nil
}

func (s *XrayAnalysisService) Latest(ctx context.Context, serverID string) (XrayAnalysisView, error) {
	serverID, err := requireLogQueryServer(serverID)
	if err != nil {
		return XrayAnalysisView{}, err
	}
	s.mu.Lock()
	state := s.latest[serverID]
	s.mu.Unlock()
	if state != nil {
		return state.view(), nil
	}
	persisted, found, err := s.loadPersistedLatest(ctx, serverID)
	if err != nil {
		return XrayAnalysisView{}, err
	}
	if found {
		return persisted, nil
	}
	source, ok := s.importer.sourceByID(ctx, serverID)
	if !ok {
		return XrayAnalysisView{}, auth.NewHTTPError(404, "找不到服务器："+serverID)
	}
	return idleXrayView(source), nil
}

func (s *XrayAnalysisService) Clear(ctx context.Context, serverID string) (XrayAnalysisView, error) {
	serverID, err := requireLogQueryServer(serverID)
	if err != nil {
		return XrayAnalysisView{}, err
	}
	s.mu.Lock()
	if s.running[serverID] {
		s.mu.Unlock()
		return XrayAnalysisView{}, auth.NewHTTPError(409, "矿透分析正在运行，完成后再清空结果")
	}
	delete(s.latest, serverID)
	s.mu.Unlock()
	if _, err := s.importer.db.ExecContext(ctx, `delete from xray_analysis_jobs where server_id = ?`, serverID); err != nil {
		return XrayAnalysisView{}, err
	}
	return s.Latest(ctx, serverID)
}

func (s *XrayAnalysisService) saveState(ctx context.Context, state *xrayState) error {
	view := state.view()
	payload, err := json.Marshal(view)
	if err != nil {
		return err
	}
	_, err = s.importer.db.ExecContext(ctx, `
		insert into xray_analysis_jobs (
			job_id, server_id, server_name, status, started_at, finished_at, from_time, to_time, player_name,
			dimension, scanned_files, scanned_rows, failed_files, player_count, finding_count, max_risk_score,
			message, payload_json
		)
		values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		on duplicate key update
			server_id = values(server_id),
			server_name = values(server_name),
			status = values(status),
			finished_at = values(finished_at),
			from_time = values(from_time),
			to_time = values(to_time),
			player_name = values(player_name),
			dimension = values(dimension),
			scanned_files = values(scanned_files),
			scanned_rows = values(scanned_rows),
			failed_files = values(failed_files),
			player_count = values(player_count),
			finding_count = values(finding_count),
			max_risk_score = values(max_risk_score),
			message = values(message),
			payload_json = values(payload_json)
	`, view.JobID, view.ServerID, view.ServerName, view.Status, view.StartedAt, view.FinishedAt,
		xrayDateTimeArg(view.FromTime), xrayDateTimeArg(view.ToTime), view.PlayerName, view.Dimension,
		view.ScannedFiles, view.ScannedRows, view.FailedFiles, view.PlayerCount, view.FindingCount,
		view.MaxRiskScore, view.Message, string(payload))
	if err != nil {
		return err
	}
	return s.prunePersisted(ctx, view.ServerID)
}

func (s *XrayAnalysisService) loadPersistedLatest(ctx context.Context, serverID string) (XrayAnalysisView, bool, error) {
	var payload string
	err := s.importer.db.QueryRowContext(ctx, `
		select payload_json
		from xray_analysis_jobs
		where server_id = ?
		order by started_at desc
		limit 1
	`, serverID).Scan(&payload)
	if err == sql.ErrNoRows {
		return XrayAnalysisView{}, false, nil
	}
	if err != nil {
		return XrayAnalysisView{}, false, err
	}
	var view XrayAnalysisView
	if err := json.Unmarshal([]byte(payload), &view); err != nil {
		return XrayAnalysisView{}, false, err
	}
	return view, true, nil
}

func (s *XrayAnalysisService) prunePersisted(ctx context.Context, serverID string) error {
	retain := max(1, s.importer.cfg.XrayRetainedJobs)
	rows, err := s.importer.db.QueryContext(ctx, `
		select job_id
		from xray_analysis_jobs
		where server_id = ?
		order by started_at desc
	`, serverID)
	if err != nil {
		return err
	}
	defer rows.Close()
	index := 0
	var stale []string
	for rows.Next() {
		var jobID string
		if err := rows.Scan(&jobID); err != nil {
			return err
		}
		if index >= retain {
			stale = append(stale, jobID)
		}
		index++
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, jobID := range stale {
		if _, err := s.importer.db.ExecContext(ctx, `delete from xray_analysis_jobs where job_id = ?`, jobID); err != nil {
			return err
		}
	}
	return nil
}

func (s *XrayAnalysisService) validateCriteria(request XrayAnalysisRequest) (xrayCriteria, error) {
	serverID, err := requireLogQueryServer(request.ServerID)
	if err != nil {
		return xrayCriteria{}, err
	}
	from, err := parseLocalDateTime(request.FromTime, s.importer.cfg.Location)
	if err != nil {
		return xrayCriteria{}, err
	}
	to, err := parseLocalDateTime(request.ToTime, s.importer.cfg.Location)
	if err != nil {
		return xrayCriteria{}, err
	}
	if from == nil || to == nil {
		return xrayCriteria{}, auth.NewHTTPError(400, "请选择矿透分析开始时间和结束时间")
	}
	if from.After(*to) {
		return xrayCriteria{}, auth.NewHTTPError(400, "开始时间不能晚于结束时间")
	}
	return xrayCriteria{serverID: serverID, fromTime: *from, toTime: *to, playerName: strings.TrimSpace(request.PlayerName), dimension: strings.TrimSpace(request.Dimension)}, nil
}

func (s *XrayAnalysisService) run(state *xrayState, source config.Source) {
	state.markRunning()
	defer func() {
		s.mu.Lock()
		s.running[state.criteria.serverID] = false
		s.mu.Unlock()
	}()
	accumulator := newXrayAccumulator()
	files, err := s.importer.localFiles(source)
	if err != nil {
		state.markFailed(err.Error())
		_ = s.saveState(context.Background(), state)
		return
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	for _, file := range files {
		fileDate := extractLogDate(file.FileName, s.importer.cfg.Location)
		if fileDate != nil {
			if fileDate.Before(dateOnly(state.criteria.fromTime, s.importer.cfg.Location)) || fileDate.After(dateOnly(state.criteria.toTime, s.importer.cfg.Location)) {
				continue
			}
		}
		state.markFileStarted(file)
		rows, err := s.scanXrayFile(file, state.criteria, accumulator)
		state.addScannedRows(rows)
		if err != nil {
			state.markFileFailed(file, err.Error())
		}
	}
	state.markFinished(accumulator.results())
	_ = s.saveState(context.Background(), state)
}

func (s *XrayAnalysisService) scanXrayFile(file RemoteLogFile, criteria xrayCriteria, accumulator *xrayAccumulator) (int64, error) {
	opened, err := os.Open(file.Path)
	if err != nil {
		return 0, err
	}
	defer opened.Close()
	scanner := bufio.NewScanner(opened)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	var scannedRows int64
	var lineNumber int64
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		columns := splitPrefix(line)
		if isHeader(columns) {
			continue
		}
		scannedRows++
		event, ok := xrayEventFromRow(file, lineNumber, columns, s.importer.cfg.Location)
		if !ok || !criteria.matches(event) {
			continue
		}
		accumulator.accept(event)
	}
	return scannedRows, scanner.Err()
}

type xrayCriteria struct {
	serverID   string
	fromTime   time.Time
	toTime     time.Time
	playerName string
	dimension  string
}

func (c xrayCriteria) matches(event xrayEvent) bool {
	if event.happenedAt.Before(c.fromTime) || event.happenedAt.After(c.toTime) {
		return false
	}
	if c.playerName != "" && !strings.EqualFold(event.playerName, c.playerName) {
		return false
	}
	return c.dimension == "" || strings.TrimSpace(event.row.Dimension2) == c.dimension
}

type xrayAccumulator struct {
	players map[string][]xrayEvent
}

func newXrayAccumulator() *xrayAccumulator {
	return &xrayAccumulator{players: map[string][]xrayEvent{}}
}

func (a *xrayAccumulator) accept(event xrayEvent) {
	a.players[event.playerName] = append(a.players[event.playerName], event)
}

func (a *xrayAccumulator) results() []XrayPlayerRiskView {
	results := make([]XrayPlayerRiskView, 0, len(a.players))
	for playerName, events := range a.players {
		sort.Slice(events, func(i, j int) bool { return events[i].happenedAt.Before(events[j].happenedAt) })
		result := analyzePlayer(playerName, events)
		if result.RiskScore >= 20 || result.AnalysisTrackingEvidenceCount > 0 || result.AnalysisAncientDebrisBreaks >= 3 {
			results = append(results, result)
		}
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].RiskScore != results[j].RiskScore {
			return results[i].RiskScore > results[j].RiskScore
		}
		return results[i].PlayerName < results[j].PlayerName
	})
	if len(results) > 200 {
		return results[:200]
	}
	return results
}

type xrayEvent struct {
	row            LogQueryRow
	happenedAt     time.Time
	playerName     string
	blockID        string
	oreType        string
	rareOre        bool
	trackingTarget bool
	tunnelBlock    bool
	point          point3
	hasPoint       bool
}

type point3 struct {
	x float64
	y float64
	z float64
}

func xrayEventFromRow(file RemoteLogFile, lineNumber int64, columns []string, location *time.Location) (xrayEvent, bool) {
	if !isXrayDestroyAction(valueAt(columns, 3)) {
		return xrayEvent{}, false
	}
	happenedAt, ok := parseDateTime(columns, location)
	if !ok {
		return xrayEvent{}, false
	}
	player := playerName(columns)
	if player == "" {
		return xrayEvent{}, false
	}
	blockID := firstKnownXrayBlock(valueAt(columns, 12), valueAt(columns, 13))
	oreType, ore := catalog.OreTypeFromBlock(blockID)
	_, tunnel := tunnelBlocks[catalog.NormalizeBlockID(blockID)]
	point, hasPoint := parsePoint3(valueAt(columns, 8), valueAt(columns, 9), valueAt(columns, 10))
	rare := ore && hasPoint && catalog.IsRareOre(oreType) && suspiciousRareDepth(oreType, point)
	return xrayEvent{
		row:            logQueryRow(file, lineNumber, columns),
		happenedAt:     happenedAt,
		playerName:     player,
		blockID:        catalog.NormalizeBlockID(blockID),
		oreType:        oreType,
		rareOre:        rare,
		trackingTarget: rare && catalog.IsTrackingTargetOre(oreType),
		tunnelBlock:    tunnel,
		point:          point,
		hasPoint:       hasPoint,
	}, true
}

func analyzePlayer(playerName string, events []xrayEvent) XrayPlayerRiskView {
	analysisMetrics := collectMetrics(events)
	strongEvidence := append(detectTurnEvidence(events), detectDirectVeinEvidence(events)...)
	weakEvidence := detectShortStraightEvidence(events)
	bedEvidence := detectBedBlastEvidence(events, analysisMetrics)
	allEvidence := append([]XrayEvidenceView{}, strongEvidence...)
	allEvidence = append(allEvidence, bedEvidence...)
	allEvidence = append(allEvidence, weakEvidence...)

	sessions := splitMiningSessions(events)
	bestSessionEvents := events
	bestSessionScore := -1
	for _, session := range sessions {
		metrics := collectMetrics(session)
		score := rawRiskScore(metrics, len(strongEvidence), weakEvidenceQualified(weakEvidence), len(bedEvidence) > 0)
		if score > bestSessionScore {
			bestSessionScore = score
			bestSessionEvents = session
		}
	}
	sessionMetrics := collectMetrics(bestSessionEvents)
	qualifiedWeak := weakEvidenceQualified(weakEvidence)
	riskScore := rawRiskScore(sessionMetrics, len(strongEvidence), qualifiedWeak, len(bedEvidence) > 0)
	riskScore = applyEvidenceCap(riskScore, len(strongEvidence), qualifiedWeak, len(bedEvidence) > 0)

	reasons := riskReasons(sessionMetrics, analysisMetrics, len(strongEvidence), qualifiedWeak, len(bedEvidence) > 0)
	return XrayPlayerRiskView{
		PlayerName:                       playerName,
		RiskScore:                        riskScore,
		RiskLevel:                        riskLevel(riskScore),
		MiningSessionCount:               len(sessions),
		MiningSessionStart:               localDateTimePtr(sessionMetrics.start),
		MiningSessionEnd:                 localDateTimePtr(sessionMetrics.end),
		MiningSessionBreaks:              sessionMetrics.breaks,
		MiningSessionUndergroundBreaks:   sessionMetrics.undergroundBreaks,
		MiningSessionOreBreaks:           sessionMetrics.oreBreaks,
		MiningSessionRareOreBreaks:       sessionMetrics.rareOreBreaks,
		MiningSessionDiamondOreBreaks:    sessionMetrics.diamondOreBreaks,
		MiningSessionAncientDebrisBreaks: sessionMetrics.ancientDebrisBreaks,
		MiningSessionRareVeins:           len(sessionMetrics.rareVeins),
		TrackingEvidenceCount:            len(strongEvidence),
		PeakRareOreWindowCount:           sessionMetrics.peakRareOreCount,
		PeakRareOreWindowStart:           localDateTimePtr(sessionMetrics.peakRareOreStart),
		PeakRareOreWindowEnd:             localDateTimePtr(sessionMetrics.peakRareOreEnd),
		PeakRareVeinWindowCount:          sessionMetrics.peakRareVeinCount,
		UndergroundRareOreRatio:          sessionMetrics.rareRatio,
		Reasons:                          reasons,
		Ores:                             oreCounts(sessionMetrics.oreCounts, false),
		RareOres:                         oreCounts(sessionMetrics.oreCounts, true),
		Evidence:                         allEvidence,
		RareOreRows:                      analysisMetrics.rareRows,
		AnalysisBreaks:                   analysisMetrics.breaks,
		AnalysisUndergroundBreaks:        analysisMetrics.undergroundBreaks,
		AnalysisOreBreaks:                analysisMetrics.oreBreaks,
		AnalysisRareOreBreaks:            analysisMetrics.rareOreBreaks,
		AnalysisDiamondOreBreaks:         analysisMetrics.diamondOreBreaks,
		AnalysisAncientDebrisBreaks:      analysisMetrics.ancientDebrisBreaks,
		AnalysisRareVeins:                len(analysisMetrics.rareVeins),
		AnalysisTrackingEvidenceCount:    len(strongEvidence),
		AnalysisPeakRareOreWindowCount:   analysisMetrics.peakRareOreCount,
		AnalysisPeakRareOreWindowStart:   localDateTimePtr(analysisMetrics.peakRareOreStart),
		AnalysisPeakRareOreWindowEnd:     localDateTimePtr(analysisMetrics.peakRareOreEnd),
		AnalysisPeakRareVeinWindowCount:  analysisMetrics.peakRareVeinCount,
		AnalysisUndergroundRareOreRatio:  analysisMetrics.rareRatio,
		AnalysisOres:                     oreCounts(analysisMetrics.oreCounts, false),
		AnalysisRareOres:                 oreCounts(analysisMetrics.oreCounts, true),
	}
}

type playerMetrics struct {
	start               *time.Time
	end                 *time.Time
	breaks              int64
	undergroundBreaks   int64
	oreBreaks           int64
	rareOreBreaks       int64
	diamondOreBreaks    int64
	ancientDebrisBreaks int64
	rareRatio           float64
	oreCounts           map[string]int64
	rareRows            []LogQueryRow
	rareVeins           []rareVein
	peakRareOreCount    int
	peakRareOreStart    *time.Time
	peakRareOreEnd      *time.Time
	peakRareVeinCount   int
}

type rareVein struct {
	events []xrayEvent
}

func collectMetrics(events []xrayEvent) playerMetrics {
	metrics := playerMetrics{oreCounts: map[string]int64{}}
	if len(events) == 0 {
		return metrics
	}
	start, end := events[0].happenedAt, events[len(events)-1].happenedAt
	metrics.start, metrics.end = &start, &end
	var rareEvents []xrayEvent
	for _, event := range events {
		metrics.breaks++
		if event.hasPoint && event.point.y <= 32 {
			metrics.undergroundBreaks++
		}
		if event.oreType != "" {
			metrics.oreBreaks++
			metrics.oreCounts[event.oreType]++
		}
		if event.rareOre {
			metrics.rareOreBreaks++
			metrics.rareRows = append(metrics.rareRows, event.row)
			rareEvents = append(rareEvents, event)
			if catalog.IsDiamond(event.oreType) {
				metrics.diamondOreBreaks++
			}
			if catalog.IsAncientDebris(event.oreType) {
				metrics.ancientDebrisBreaks++
			}
		}
	}
	if metrics.undergroundBreaks > 0 {
		metrics.rareRatio = float64(metrics.rareOreBreaks) / float64(metrics.undergroundBreaks)
	}
	metrics.rareVeins = buildRareVeins(rareEvents)
	metrics.peakRareOreCount, metrics.peakRareOreStart, metrics.peakRareOreEnd = peakRareOreWindow(rareEvents)
	metrics.peakRareVeinCount = peakRareVeinWindow(metrics.rareVeins)
	return metrics
}

func splitMiningSessions(events []xrayEvent) [][]xrayEvent {
	if len(events) == 0 {
		return nil
	}
	var sessions [][]xrayEvent
	current := []xrayEvent{events[0]}
	for _, event := range events[1:] {
		gap := event.happenedAt.Sub(current[len(current)-1].happenedAt)
		if gap > miningSessionGapSeconds*time.Second {
			if len(current) >= miningSessionEvents {
				sessions = append(sessions, current)
			}
			current = []xrayEvent{event}
			continue
		}
		current = append(current, event)
	}
	if len(current) >= miningSessionEvents {
		sessions = append(sessions, current)
	}
	if len(sessions) == 0 {
		return [][]xrayEvent{events}
	}
	return sessions
}

func buildRareVeins(events []xrayEvent) []rareVein {
	var veins []rareVein
	for _, event := range events {
		accepted := false
		for i := range veins {
			last := veins[i].events[len(veins[i].events)-1]
			if last.hasPoint && event.hasPoint && last.row.Dimension2 == event.row.Dimension2 &&
				event.happenedAt.Sub(last.happenedAt) <= 30*time.Second &&
				distance(last.point, event.point) <= sameVeinDistance {
				veins[i].events = append(veins[i].events, event)
				accepted = true
				break
			}
		}
		if !accepted {
			veins = append(veins, rareVein{events: []xrayEvent{event}})
		}
	}
	return veins
}

func peakRareOreWindow(events []xrayEvent) (int, *time.Time, *time.Time) {
	best := 0
	var bestStart, bestEnd *time.Time
	left := 0
	for right, event := range events {
		for left <= right && event.happenedAt.Sub(events[left].happenedAt) > shortRareOreWindowSecond*time.Second {
			left++
		}
		count := right - left + 1
		if count > best {
			best = count
			start, end := events[left].happenedAt, event.happenedAt
			bestStart, bestEnd = &start, &end
		}
	}
	return best, bestStart, bestEnd
}

func peakRareVeinWindow(veins []rareVein) int {
	best := 0
	left := 0
	for right, vein := range veins {
		t := vein.events[0].happenedAt
		for left <= right && t.Sub(veins[left].events[0].happenedAt) > shortRareOreWindowSecond*time.Second {
			left++
		}
		if count := right - left + 1; count > best {
			best = count
		}
	}
	return best
}

func detectTurnEvidence(events []xrayEvent) []XrayEvidenceView {
	var evidence []XrayEvidenceView
	var usedTargets []xrayEvent
	for index, target := range events {
		if !target.trackingTarget || !target.hasPoint {
			continue
		}
		if sameTargetUsed(usedTargets, target) {
			continue
		}
		window := recentTunnelEvents(events[:index], target, turnLookbackSeconds)
		if len(window) < minTurnMainTunnelBlocks+minTurnSideTunnelBlocks {
			continue
		}
		if ev, ok := turnEvidenceFromWindow(window, target); ok {
			evidence = append(evidence, ev)
			usedTargets = append(usedTargets, target)
		}
	}
	return evidence
}

func turnEvidenceFromWindow(window []xrayEvent, target xrayEvent) (XrayEvidenceView, bool) {
	for sideLen := minTurnSideTunnelBlocks; sideLen <= maxTurnSideTunnelBlocks; sideLen++ {
		if len(window) < minTurnMainTunnelBlocks+sideLen {
			continue
		}
		side := window[len(window)-sideLen:]
		main := window[:len(window)-sideLen]
		if len(main) > 18 {
			main = main[len(main)-18:]
		}
		if len(main) < minTurnMainTunnelBlocks {
			continue
		}
		mainFit := straightness(main)
		sideFit := straightness(side)
		if mainFit < minTurnCorridorFit || sideFit < minTurnCorridorFit {
			continue
		}
		mainDir, ok1 := direction(main[0].point, main[len(main)-1].point)
		sideDir, ok2 := direction(side[0].point, side[len(side)-1].point)
		if !ok1 || !ok2 || math.Abs(dot(mainDir, sideDir)) > maxTurnDirectionDot {
			continue
		}
		if distance(side[len(side)-1].point, target.point) > 5.5 {
			continue
		}
		rows := rowsFromEvents(append(append([]xrayEvent{}, main...), append(side, target)...))
		return XrayEvidenceView{
			Type:      "TURN_TO_RARE_ORE",
			Score:     26,
			StartedAt: localDateTimeValue(main[0].happenedAt),
			EndedAt:   localDateTimeValue(target.happenedAt),
			Summary:   "拐弯追矿：直线通道 " + strconvI64(int64(len(main))) + " 格后近直角转向，" + strconvI64(int64(len(side))) + " 格内命中 " + catalog.OreLabel(target.oreType) + "，路径贴合度 ≥95%",
			Rows:      rows,
		}, true
	}
	return XrayEvidenceView{}, false
}

func detectDirectVeinEvidence(events []xrayEvent) []XrayEvidenceView {
	var targetEvents []xrayEvent
	for _, event := range events {
		if event.trackingTarget {
			targetEvents = append(targetEvents, event)
		}
	}
	veins := buildRareVeins(targetEvents)
	used := map[int]bool{}
	var evidence []XrayEvidenceView
	for i := 0; i < len(veins); i++ {
		if used[i] {
			continue
		}
		for j := i + 1; j < len(veins); j++ {
			if used[j] {
				continue
			}
			a := veins[i].events[len(veins[i].events)-1]
			b := veins[j].events[0]
			seconds := int(b.happenedAt.Sub(a.happenedAt).Seconds())
			if seconds < minDirectVeinSeconds || seconds > maxDirectVeinSeconds || !a.hasPoint || !b.hasPoint || a.row.Dimension2 != b.row.Dimension2 {
				continue
			}
			dist := distance(a.point, b.point)
			if dist < minDirectVeinDistance || dist > maxDirectVeinDistance {
				continue
			}
			gapBlocks := tunnelBlocksBetween(events, a.happenedAt, b.happenedAt)
			if gapBlocks > maxDirectVeinGapBlocks {
				continue
			}
			used[i], used[j] = true, true
			evidence = append(evidence, XrayEvidenceView{
				Type:      "DIRECT_VEIN_TO_VEIN",
				Score:     24,
				StartedAt: localDateTimeValue(a.happenedAt),
				EndedAt:   localDateTimeValue(b.happenedAt),
				Summary:   "矿脉间直达：两个稀有矿脉相距 " + formatDistance(dist) + " 格，中间仅连续挖掘 " + strconvI64(int64(gapBlocks)) + " 个普通方块",
				Rows:      rowsFromEvents([]xrayEvent{a, b}),
			})
			break
		}
	}
	return evidence
}

func detectShortStraightEvidence(events []xrayEvent) []XrayEvidenceView {
	var evidence []XrayEvidenceView
	for index, target := range events {
		if !target.trackingTarget || !target.hasPoint {
			continue
		}
		window := recentTunnelEvents(events[:index], target, maxShortStraightSeconds)
		if len(window) < minShortStraightBlocks {
			continue
		}
		if len(window) > maxShortStraightBlocks {
			window = window[len(window)-maxShortStraightBlocks:]
		}
		if straightness(window) < minShortStraightness {
			continue
		}
		evidence = append(evidence, XrayEvidenceView{
			Type:      "SHORT_STRAIGHT_TO_RARE_ORE",
			Score:     5,
			StartedAt: localDateTimeValue(window[0].happenedAt),
			EndedAt:   localDateTimeValue(target.happenedAt),
			Summary:   "短直线弱证据：" + strconvI64(int64(len(window))) + " 格普通方块后命中 " + catalog.OreLabel(target.oreType),
			Rows:      rowsFromEvents(append(append([]xrayEvent{}, window...), target)),
		})
	}
	return evidence
}

func detectBedBlastEvidence(events []xrayEvent, metrics playerMetrics) []XrayEvidenceView {
	if metrics.ancientDebrisBreaks < 3 {
		return nil
	}
	tunnelCount := 0
	var rows []LogQueryRow
	var start, end *time.Time
	for _, event := range events {
		if strings.Contains(event.row.Dimension2, "下界") || strings.EqualFold(event.row.Dimension2, "nether") {
			if event.tunnelBlock {
				tunnelCount++
			}
			if event.oreType == "ANCIENT_DEBRIS" {
				rows = append(rows, event.row)
				if start == nil || event.happenedAt.Before(*start) {
					t := event.happenedAt
					start = &t
				}
				if end == nil || event.happenedAt.After(*end) {
					t := event.happenedAt
					end = &t
				}
			}
		}
	}
	allowed := int(metrics.ancientDebrisBreaks*8) + 12
	if tunnelCount > allowed {
		return nil
	}
	return []XrayEvidenceView{{
		Type:      "BED_BLAST_ANCIENT_DEBRIS",
		Score:     20,
		StartedAt: localDateTimePtr(start),
		EndedAt:   localDateTimePtr(end),
		Summary:   "远古残骸异常密集：下界疑似床炸场景中挖掘普通方块少，但快速发现 " + strconvI64(metrics.ancientDebrisBreaks) + " 个远古残骸",
		Rows:      rows,
	}}
}

func rawRiskScore(metrics playerMetrics, strongEvidenceCount int, weakQualified bool, bedEvidence bool) int {
	score := 0.0
	score += math.Min(16, float64(metrics.rareOreBreaks)*16.0/50.0)
	score += math.Min(16, float64(metrics.peakRareOreCount)*16.0/35.0)
	score += math.Min(16, metrics.rareRatio*16.0/0.03)
	score += math.Min(12, float64(metrics.peakRareVeinCount)*12.0/35.0)
	score += math.Min(12, float64(len(metrics.rareVeins))*12.0/80.0)
	if strongEvidenceCount >= 3 {
		score += math.Min(40, float64(strongEvidenceCount)*12.0)
	} else {
		score += float64(strongEvidenceCount * 6)
	}
	if weakQualified {
		score += 5
	}
	if bedEvidence {
		score += 20
	}
	return int(math.Round(score))
}

func applyEvidenceCap(score, strongEvidenceCount int, weakQualified, bedEvidence bool) int {
	capScore := 44
	if weakQualified {
		capScore = 49
	}
	if bedEvidence {
		capScore = max(capScore, 68)
	}
	if strongEvidenceCount >= 3 {
		capScore = 100
	}
	if score > capScore {
		return capScore
	}
	return score
}

func riskReasons(session, analysis playerMetrics, strongEvidenceCount int, weakQualified, bedEvidence bool) []string {
	var reasons []string
	if session.rareOreBreaks >= 50 {
		reasons = append(reasons, "稀有矿数量异常")
	}
	if session.peakRareOreCount >= 35 {
		reasons = append(reasons, "10 分钟稀有矿峰值异常")
	}
	if session.rareRatio >= 0.03 {
		reasons = append(reasons, "地下稀有矿占比异常")
	}
	if session.peakRareVeinCount >= 20 {
		reasons = append(reasons, "矿脉发现速度异常")
	}
	if strongEvidenceCount >= 3 {
		reasons = append(reasons, "精准追矿证据 "+strconvI64(int64(strongEvidenceCount))+" 次")
	} else if strongEvidenceCount > 0 {
		reasons = append(reasons, "追矿路线证据不足，已保守降权")
	}
	if weakQualified {
		reasons = append(reasons, "频繁短直线命中稀有矿")
	}
	if bedEvidence {
		reasons = append(reasons, "下界远古残骸疑似床炸追矿")
	}
	if analysis.rareOreBreaks != session.rareOreBreaks {
		reasons = append(reasons, "统计周期稀有矿 "+strconvI64(analysis.rareOreBreaks)+" 个")
	}
	return reasons
}

type xrayState struct {
	mu           sync.Mutex
	jobID        string
	serverID     string
	serverName   string
	criteria     xrayCriteria
	startedAt    time.Time
	finishedAt   *time.Time
	status       string
	scannedFiles int
	scannedRows  int64
	failedFiles  int
	currentFile  string
	message      string
	players      []XrayPlayerRiskView
}

func newXrayState(source config.Source, criteria xrayCriteria) *xrayState {
	return &xrayState{jobID: newJobID(), serverID: source.ID, serverName: source.Name, criteria: criteria, startedAt: time.Now().UTC(), status: "PENDING"}
}

func (s *xrayState) markRunning() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status = "RUNNING"
	s.message = "正在扫描 CSV"
}

func (s *xrayState) markFileStarted(file RemoteLogFile) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.scannedFiles++
	s.currentFile = file.FileName
}

func (s *xrayState) addScannedRows(rows int64) {
	s.mu.Lock()
	s.scannedRows += rows
	s.mu.Unlock()
}

func (s *xrayState) markFileFailed(file RemoteLogFile, message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failedFiles++
	s.currentFile = file.FileName
	s.message = "部分文件无法解析：" + message
}

func (s *xrayState) markFinished(players []XrayPlayerRiskView) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	s.finishedAt = &now
	s.status = "FINISHED"
	if s.failedFiles > 0 {
		s.status = "FINISHED_WITH_ERRORS"
	}
	s.currentFile = ""
	s.players = players
	findingCount := 0
	maxRisk := 0
	for _, player := range players {
		findingCount += player.AnalysisTrackingEvidenceCount
		if player.RiskScore > maxRisk {
			maxRisk = player.RiskScore
		}
	}
	if s.scannedFiles == 0 {
		s.message = "没有扫描到时间范围内的 CSV 文件"
	} else if s.scannedRows == 0 {
		s.message = "时间范围内的 CSV 没有可分析事件"
	} else if len(players) == 0 {
		s.message = "没有发现明显矿透风险"
	} else {
		s.message = "发现 " + strconvI64(int64(len(players))) + " 名玩家存在疑似风险，最高 " + strconvI64(int64(maxRisk)) + " 分"
	}
}

func (s *xrayState) markFailed(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	s.finishedAt = &now
	s.status = "FAILED"
	s.currentFile = ""
	s.message = message
}

func (s *xrayState) view() XrayAnalysisView {
	s.mu.Lock()
	defer s.mu.Unlock()
	findingCount := 0
	maxRisk := 0
	for _, player := range s.players {
		findingCount += player.AnalysisTrackingEvidenceCount
		if player.RiskScore > maxRisk {
			maxRisk = player.RiskScore
		}
	}
	return XrayAnalysisView{
		JobID:        s.jobID,
		ServerID:     s.serverID,
		ServerName:   s.serverName,
		Status:       s.status,
		StartedAt:    s.startedAt,
		FinishedAt:   s.finishedAt,
		FromTime:     localDateTimeValue(s.criteria.fromTime),
		ToTime:       localDateTimeValue(s.criteria.toTime),
		PlayerName:   s.criteria.playerName,
		Dimension:    s.criteria.dimension,
		ScannedFiles: s.scannedFiles,
		ScannedRows:  s.scannedRows,
		FailedFiles:  s.failedFiles,
		PlayerCount:  len(s.players),
		FindingCount: findingCount,
		MaxRiskScore: maxRisk,
		CurrentFile:  s.currentFile,
		Message:      s.message,
		Players:      append([]XrayPlayerRiskView(nil), s.players...),
	}
}

func idleXrayView(source config.Source) XrayAnalysisView {
	return XrayAnalysisView{ServerID: source.ID, ServerName: source.Name, Status: "IDLE", StartedAt: time.Now().UTC(), Message: "还没有分析记录", Players: []XrayPlayerRiskView{}}
}

func firstKnownXrayBlock(detail1, detail2 string) string {
	first := catalog.NormalizeBlockID(detail1)
	if _, ok := catalog.OreTypeFromBlock(first); ok {
		return first
	}
	if _, ok := tunnelBlocks[first]; ok {
		return first
	}
	second := catalog.NormalizeBlockID(detail2)
	if _, ok := catalog.OreTypeFromBlock(second); ok {
		return second
	}
	if _, ok := tunnelBlocks[second]; ok {
		return second
	}
	if first != "" {
		return first
	}
	return second
}

func isXrayDestroyAction(action string) bool {
	if parseAction(action) == "DESTROY_BLOCK" {
		return true
	}
	normalized := strings.ToLower(strings.TrimSpace(action))
	return strings.Contains(normalized, "破坏") || strings.Contains(normalized, "destroy") || strings.Contains(normalized, "break")
}

func suspiciousRareDepth(oreType string, point point3) bool {
	if catalog.IsAncientDebris(oreType) {
		return point.y <= 32
	}
	return point.y <= 16
}

func parsePoint3(xRaw, yRaw, zRaw string) (point3, bool) {
	x, ok := parseFloat(xRaw)
	if !ok {
		return point3{}, false
	}
	y, ok := parseFloat(yRaw)
	if !ok {
		return point3{}, false
	}
	z, ok := parseFloat(zRaw)
	if !ok {
		return point3{}, false
	}
	return point3{x: x, y: y, z: z}, true
}

func recentTunnelEvents(events []xrayEvent, target xrayEvent, seconds int) []xrayEvent {
	result := make([]xrayEvent, 0)
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		if target.happenedAt.Sub(event.happenedAt) > time.Duration(seconds)*time.Second {
			break
		}
		if event.tunnelBlock && event.hasPoint && event.row.Dimension2 == target.row.Dimension2 {
			result = append(result, event)
		}
	}
	for left, right := 0, len(result)-1; left < right; left, right = left+1, right-1 {
		result[left], result[right] = result[right], result[left]
	}
	return result
}

func sameTargetUsed(used []xrayEvent, target xrayEvent) bool {
	for _, event := range used {
		if event.row.Dimension2 == target.row.Dimension2 && math.Abs(target.happenedAt.Sub(event.happenedAt).Seconds()) <= 60 && distance(event.point, target.point) < 3 {
			return true
		}
	}
	return false
}

func straightness(events []xrayEvent) float64 {
	if len(events) < 2 {
		return 1
	}
	path := 0.0
	for i := 1; i < len(events); i++ {
		path += distance(events[i-1].point, events[i].point)
	}
	if path == 0 {
		return 1
	}
	return distance(events[0].point, events[len(events)-1].point) / path
}

func direction(a, b point3) (point3, bool) {
	dx, dz := b.x-a.x, b.z-a.z
	length := math.Sqrt(dx*dx + dz*dz)
	if length == 0 {
		return point3{}, false
	}
	return point3{x: dx / length, z: dz / length}, true
}

func dot(a, b point3) float64 {
	return a.x*b.x + a.z*b.z
}

func distance(a, b point3) float64 {
	dx, dy, dz := a.x-b.x, a.y-b.y, a.z-b.z
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

func tunnelBlocksBetween(events []xrayEvent, start, end time.Time) int {
	count := 0
	for _, event := range events {
		if event.happenedAt.After(start) && event.happenedAt.Before(end) && event.tunnelBlock {
			count++
		}
	}
	return count
}

func weakEvidenceQualified(evidence []XrayEvidenceView) bool {
	if len(evidence) < 3 {
		return false
	}
	var times []time.Time
	for _, ev := range evidence {
		if ev.StartedAt != nil {
			times = append(times, ev.StartedAt.Time)
		}
	}
	sort.Slice(times, func(i, j int) bool { return times[i].Before(times[j]) })
	for i := 2; i < len(times); i++ {
		if times[i].Sub(times[i-2]) <= 3*time.Minute {
			return true
		}
	}
	return false
}

func rowsFromEvents(events []xrayEvent) []LogQueryRow {
	rows := make([]LogQueryRow, 0, len(events))
	for _, event := range events {
		rows = append(rows, event.row)
	}
	return rows
}

func oreCounts(counts map[string]int64, rareOnly bool) []XrayOreCount {
	result := make([]XrayOreCount, 0)
	for _, typ := range catalog.OreTypes {
		count := counts[typ.Type]
		if count == 0 {
			continue
		}
		if rareOnly && !catalog.IsRareOre(typ.Type) {
			continue
		}
		result = append(result, XrayOreCount{OreType: typ.Type, DisplayName: typ.Label, Count: count})
	}
	return result
}

func riskLevel(score int) string {
	switch {
	case score >= 85:
		return "极高"
	case score >= 70:
		return "高"
	case score >= 45:
		return "中"
	case score >= 20:
		return "低"
	default:
		return "观察"
	}
}

func parseLocalDateTime(raw string, location *time.Location) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	layouts := []string{"2006-01-02T15:04:05", "2006-01-02 15:04:05", time.RFC3339}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, raw, location); err == nil {
			return &t, nil
		}
	}
	return nil, auth.NewHTTPError(400, "时间格式应为 YYYY-MM-DDTHH:mm:ss")
}

func localDateTimePtr(value *time.Time) *apitype.LocalDateTime {
	if value == nil {
		return nil
	}
	result := apitype.NewLocalDateTime(*value)
	return &result
}

func localDateTimeValue(value time.Time) *apitype.LocalDateTime {
	result := apitype.NewLocalDateTime(value)
	return &result
}

func xrayDateTimeArg(value *apitype.LocalDateTime) any {
	if value == nil {
		return nil
	}
	return value.Time
}

func formatDistance(value float64) string {
	return strconvI64(int64(math.Round(value)))
}
