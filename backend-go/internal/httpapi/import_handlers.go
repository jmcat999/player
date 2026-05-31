package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"player-stats-backend-go/internal/auth"
	"player-stats-backend-go/internal/importer"
)

func (s *Server) importFiles(w http.ResponseWriter, r *http.Request) {
	response, err := s.importService.ListLocalImportFiles(r.Context(), r.URL.Query().Get("serverId"))
	s.writeHTTPErrorResult(w, http.StatusOK, response, err)
}

func (s *Server) remoteImportFiles(w http.ResponseWriter, r *http.Request) {
	response, err := s.importService.ListRemoteSMBFiles(r.Context(), r.URL.Query().Get("serverId"))
	s.writeHTTPErrorResult(w, http.StatusOK, response, err)
}

func (s *Server) startImportJob(w http.ResponseWriter, r *http.Request) {
	serverID, ok := requireSingleServer(w, r.URL.Query().Get("serverId"))
	if !ok {
		return
	}
	response, err := s.importJobs.StartImport(r.Context(), serverID, queryBool(r, "skipToday", true))
	s.writeHTTPErrorResult(w, http.StatusAccepted, response, err)
}

func (s *Server) importJob(w http.ResponseWriter, r *http.Request) {
	response, err := s.importJobs.GetJob(r.PathValue("jobId"))
	s.writeHTTPErrorResult(w, http.StatusOK, response, err)
}

func (s *Server) deleteImportFile(w http.ResponseWriter, r *http.Request) {
	response, err := s.importService.DeleteImportRecord(r.Context(), r.URL.Query().Get("serverId"), r.URL.Query().Get("remotePath"))
	s.writeHTTPErrorResult(w, http.StatusOK, response, err)
}

func (s *Server) deleteImportRecords(w http.ResponseWriter, r *http.Request) {
	var request importer.DeleteImportRecordsRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "请求体不是有效 JSON")
		return
	}
	writeJSON(w, http.StatusOK, s.importService.DeleteImportRecords(r.Context(), request))
}

func (s *Server) deleteLocalFiles(w http.ResponseWriter, r *http.Request) {
	var request importer.DeleteImportRecordsRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "请求体不是有效 JSON")
		return
	}
	writeJSON(w, http.StatusOK, s.importService.DeleteLocalFiles(r.Context(), request))
}

func (s *Server) startSyncJob(w http.ResponseWriter, r *http.Request) {
	serverID, ok := requireSingleServer(w, r.URL.Query().Get("serverId"))
	if !ok {
		return
	}
	response, err := s.importJobs.StartSync(r.Context(), serverID, queryBool(r, "skipToday", true))
	s.writeHTTPErrorResult(w, http.StatusAccepted, response, err)
}

func (s *Server) syncJob(w http.ResponseWriter, r *http.Request) {
	response, err := s.importJobs.GetSyncJob(r.PathValue("jobId"))
	s.writeHTTPErrorResult(w, http.StatusOK, response, err)
}

func (s *Server) autoTaskLogs(w http.ResponseWriter, r *http.Request) {
	response, err := s.importService.LatestAutoTaskLogs(r.Context(), queryInt(r, "limit", 80))
	s.writeHTTPErrorResult(w, http.StatusOK, response, err)
}

func (s *Server) clearAutoTaskLogs(w http.ResponseWriter, r *http.Request) {
	err := s.importService.ClearAutoTaskLogs(r.Context())
	if err != nil {
		s.writeHTTPErrorResult(w, http.StatusOK, nil, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) startLogQuery(w http.ResponseWriter, r *http.Request) {
	var request importer.LogQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "请求体不是有效 JSON")
		return
	}
	response, err := s.logQueryService.Start(r.Context(), request)
	s.writeHTTPErrorResult(w, http.StatusAccepted, response, err)
}

func (s *Server) latestLogQuery(w http.ResponseWriter, r *http.Request) {
	response, err := s.logQueryService.Latest(
		r.Context(),
		r.URL.Query().Get("serverId"),
		r.URL.Query().Get("queryType"),
		queryInt(r, "page", 1),
		queryInt(r, "pageSize", 100),
	)
	s.writeHTTPErrorResult(w, http.StatusOK, response, err)
}

func (s *Server) clearLogQuery(w http.ResponseWriter, r *http.Request) {
	response, err := s.logQueryService.Clear(r.Context(), r.URL.Query().Get("serverId"), r.URL.Query().Get("queryType"))
	s.writeHTTPErrorResult(w, http.StatusOK, response, err)
}

func (s *Server) startXrayAnalysis(w http.ResponseWriter, r *http.Request) {
	var request importer.XrayAnalysisRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "请求体不是有效 JSON")
		return
	}
	response, err := s.xrayService.Start(r.Context(), request)
	s.writeHTTPErrorResult(w, http.StatusAccepted, response, err)
}

func (s *Server) latestXrayAnalysis(w http.ResponseWriter, r *http.Request) {
	response, err := s.xrayService.Latest(r.Context(), r.URL.Query().Get("serverId"))
	s.writeHTTPErrorResult(w, http.StatusOK, response, err)
}

func (s *Server) clearXrayAnalysis(w http.ResponseWriter, r *http.Request) {
	response, err := s.xrayService.Clear(r.Context(), r.URL.Query().Get("serverId"))
	s.writeHTTPErrorResult(w, http.StatusOK, response, err)
}

func requireSingleServer(w http.ResponseWriter, serverID string) (string, bool) {
	serverID = strings.TrimSpace(serverID)
	if serverID == "" || strings.EqualFold(serverID, "all") || strings.EqualFold(serverID, "total") {
		status, message := auth.ErrorStatus(auth.NewHTTPError(400, "请指定单个服务器后再执行复制或解析"))
		writeError(w, status, message)
		return "", false
	}
	return serverID, true
}

func queryBool(r *http.Request, key string, fallback bool) bool {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return value
}
