package httpapi

import (
	"log/slog"
	"net/http"
	"time"

	"player-stats-backend-go/internal/auth"
	"player-stats-backend-go/internal/config"
	"player-stats-backend-go/internal/importer"
	"player-stats-backend-go/internal/settings"
	"player-stats-backend-go/internal/share"
	"player-stats-backend-go/internal/stats"
)

type Server struct {
	cfg             config.Config
	logger          *slog.Logger
	authService     *auth.Service
	settingsService *settings.Service
	shareService    *share.Service
	importService   *importer.Service
	importJobs      *importer.JobService
	logQueryService *importer.LogQueryService
	xrayService     *importer.XrayAnalysisService
	statsService    *stats.Service
	mux             *http.ServeMux
}

func NewServer(cfg config.Config, logger *slog.Logger, authService *auth.Service, settingsService *settings.Service, shareService *share.Service, importService *importer.Service, importJobs *importer.JobService, logQueryService *importer.LogQueryService, xrayService *importer.XrayAnalysisService, statsService *stats.Service) *Server {
	server := &Server{
		cfg:             cfg,
		logger:          logger,
		authService:     authService,
		settingsService: settingsService,
		shareService:    shareService,
		importService:   importService,
		importJobs:      importJobs,
		logQueryService: logQueryService,
		xrayService:     xrayService,
		statsService:    statsService,
		mux:             http.NewServeMux(),
	}
	server.routes()
	return server
}

func (s *Server) Handler() http.Handler {
	return s.corsMiddleware(s.authMiddleware(s.mux))
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/health", s.health)
	s.mux.HandleFunc("GET /api/stats/server-time", s.serverTime)

	s.mux.HandleFunc("POST /api/auth/login", s.login)
	s.mux.HandleFunc("GET /api/auth/me", s.me)
	s.mux.HandleFunc("POST /api/auth/password", s.changePassword)
	s.mux.HandleFunc("POST /api/auth/logout", s.logout)

	s.mux.HandleFunc("GET /api/stats/overview", s.statsOverview)
	s.mux.HandleFunc("GET /api/stats/servers", s.statsServers)
	s.mux.HandleFunc("GET /api/stats/server-options", s.statsServerOptions)
	s.mux.HandleFunc("GET /api/stats/players", s.statsPlayers)
	s.mux.HandleFunc("GET /api/stats/player", s.statsPlayer)
	s.mux.HandleFunc("GET /api/stats/player-presence", s.statsPlayerPresence)
	s.mux.HandleFunc("GET /api/stats/public-coordinate-logs", s.publicCoordinateLogs)
	s.mux.HandleFunc("GET /api/stats/daily", s.statsDaily)
	s.mux.HandleFunc("GET /api/stats/imports", s.statsImports)

	s.mux.HandleFunc("GET /api/import/files", s.importFiles)
	s.mux.HandleFunc("GET /api/import/remote-files", s.remoteImportFiles)
	s.mux.HandleFunc("POST /api/import/jobs", s.startImportJob)
	s.mux.HandleFunc("GET /api/import/jobs/{jobId}", s.importJob)
	s.mux.HandleFunc("DELETE /api/import/files", s.deleteImportFile)
	s.mux.HandleFunc("POST /api/import/files/delete-records", s.deleteImportRecords)
	s.mux.HandleFunc("POST /api/import/local-files/delete", s.deleteLocalFiles)
	s.mux.HandleFunc("POST /api/import/sync-jobs", s.startSyncJob)
	s.mux.HandleFunc("GET /api/import/sync-jobs/{jobId}", s.syncJob)
	s.mux.HandleFunc("GET /api/import/auto-task-logs", s.autoTaskLogs)
	s.mux.HandleFunc("DELETE /api/import/auto-task-logs", s.clearAutoTaskLogs)
	s.mux.HandleFunc("POST /api/import/log-query-jobs", s.startLogQuery)
	s.mux.HandleFunc("GET /api/import/log-query-jobs/latest", s.latestLogQuery)
	s.mux.HandleFunc("DELETE /api/import/log-query-jobs/latest", s.clearLogQuery)
	s.mux.HandleFunc("POST /api/import/xray-analysis-jobs", s.startXrayAnalysis)
	s.mux.HandleFunc("GET /api/import/xray-analysis-jobs/latest", s.latestXrayAnalysis)
	s.mux.HandleFunc("DELETE /api/import/xray-analysis-jobs/latest", s.clearXrayAnalysis)

	s.mux.HandleFunc("GET /api/config/sync", s.configSync)
	s.mux.HandleFunc("PUT /api/config/sync", s.saveConfigSync)
	s.mux.HandleFunc("GET /api/config/astrbot-key", s.astrBotKey)
	s.mux.HandleFunc("POST /api/config/astrbot-key/reset", s.resetAstrBotKey)
	s.mux.HandleFunc("POST /api/config/smb-test", s.smbTest)
	s.mux.HandleFunc("GET /api/config/source-files", s.sourceFiles)

	s.mux.HandleFunc("POST /api/share/tokens", s.createPlayerShareToken)
	s.mux.HandleFunc("POST /api/share/ranking-tokens", s.createRankingShareToken)
	s.mux.HandleFunc("GET /api/share/ranking/{token}", s.rankingShareDetails)
	s.mux.HandleFunc("POST /api/share/xray/send-to-group", s.sendXrayToGroup)
	s.mux.HandleFunc("GET /api/share/xray/{token}", s.xrayShareDetails)
	s.mux.HandleFunc("GET /api/share/xray-group-messages/pending", s.pendingXrayGroupMessages)
	s.mux.HandleFunc("GET /api/share/xray-group-messages/ws", s.xrayGroupMessagesWS)
	s.mux.HandleFunc("POST /api/share/xray-group-messages/{messageId}/delivery", s.markXrayGroupDelivery)
	s.mux.HandleFunc("GET /api/share/{token}", s.playerShareDetails)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "UP",
		"server": "go",
		"time":   time.Now().In(s.cfg.Location).Format(time.RFC3339),
	})
}

func (s *Server) serverTime(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"serverTime": time.Now().In(s.cfg.Location).Format(time.RFC3339),
	})
}
