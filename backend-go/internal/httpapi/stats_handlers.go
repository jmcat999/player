package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"player-stats-backend-go/internal/importer"
	"player-stats-backend-go/internal/stats"
)

func (s *Server) statsOverview(w http.ResponseWriter, r *http.Request) {
	from, to, ok := s.parseDateRange(w, r)
	if !ok {
		return
	}
	response, err := s.statsService.Overview(r.Context(), r.URL.Query().Get("serverId"), from, to)
	s.writeServiceResult(w, response, err)
}

func (s *Server) statsServers(w http.ResponseWriter, r *http.Request) {
	from, to, ok := s.parseDateRange(w, r)
	if !ok {
		return
	}
	response, err := s.statsService.Servers(r.Context(), from, to)
	s.writeServiceResult(w, response, err)
}

func (s *Server) statsServerOptions(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.statsService.ServerOptions())
}

func (s *Server) statsPlayers(w http.ResponseWriter, r *http.Request) {
	from, to, ok := s.parseDateRange(w, r)
	if !ok {
		return
	}
	limit := queryInt(r, "limit", 50)
	response, err := s.statsService.Players(
		r.Context(),
		r.URL.Query().Get("serverId"),
		from,
		to,
		r.URL.Query().Get("player"),
		limit,
	)
	s.writeServiceResult(w, response, err)
}

func (s *Server) statsPlayer(w http.ResponseWriter, r *http.Request) {
	from, to, ok := s.parseDateRange(w, r)
	if !ok {
		return
	}
	response, found, err := s.statsService.Player(
		r.Context(),
		r.URL.Query().Get("serverId"),
		r.URL.Query().Get("playerName"),
		from,
		to,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "服务器内部错误")
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "玩家没有统计数据")
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) statsPlayerPresence(w http.ResponseWriter, r *http.Request) {
	response, found, err := s.statsService.PlayerPresence(
		r.Context(),
		r.URL.Query().Get("serverId"),
		r.URL.Query().Get("playerName"),
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "服务器内部错误")
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "没有这个玩家信息")
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) publicCoordinateLogs(w http.ResponseWriter, r *http.Request) {
	x, ok := queryFloat(w, r, "x")
	if !ok {
		return
	}
	y, ok := queryFloat(w, r, "y")
	if !ok {
		return
	}
	z, ok := queryFloat(w, r, "z")
	if !ok {
		return
	}
	response, err := s.logQueryService.PublicCoordinate(r.Context(), importer.PublicCoordinateLogQueryRequest{
		ServerID: r.URL.Query().Get("serverId"),
		X:        x,
		Y:        y,
		Z:        z,
		Limit:    queryInt(r, "limit", 8),
		Days:     queryInt(r, "days", 7),
	})
	s.writeHTTPErrorResult(w, http.StatusOK, response, err)
}

func (s *Server) statsDaily(w http.ResponseWriter, r *http.Request) {
	from, to, ok := s.parseDateRange(w, r)
	if !ok {
		return
	}
	response, err := s.statsService.Daily(
		r.Context(),
		r.URL.Query().Get("serverId"),
		from,
		to,
		r.URL.Query().Get("player"),
	)
	s.writeServiceResult(w, response, err)
}

func (s *Server) statsImports(w http.ResponseWriter, r *http.Request) {
	limit := queryInt(r, "limit", 50)
	response, err := s.statsService.ImportedFiles(r.Context(), r.URL.Query().Get("serverId"), limit)
	s.writeServiceResult(w, response, err)
}

func (s *Server) parseDateRange(w http.ResponseWriter, r *http.Request) (from, to *time.Time, ok bool) {
	from, err := stats.ParseDate(r.URL.Query().Get("from"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return nil, nil, false
	}
	to, err = stats.ParseDate(r.URL.Query().Get("to"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return nil, nil, false
	}
	return from, to, true
}

func (s *Server) writeServiceResult(w http.ResponseWriter, payload any, err error) {
	if err != nil {
		s.logger.Error("request failed", "error", err)
		writeError(w, http.StatusInternalServerError, "服务器内部错误")
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

func queryInt(r *http.Request, key string, fallback int) int {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func queryFloat(w http.ResponseWriter, r *http.Request, key string) (float64, bool) {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		writeError(w, http.StatusBadRequest, key+" 不能为空")
		return 0, false
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, key+" 必须是数字")
		return 0, false
	}
	return value, true
}
