package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"player-stats-backend-go/internal/auth"
	"player-stats-backend-go/internal/settings"
	sharemodel "player-stats-backend-go/internal/share"
	"player-stats-backend-go/internal/stats"
)

func (s *Server) createPlayerShareToken(w http.ResponseWriter, r *http.Request) {
	if !s.authService.MatchesAstrBotKey(r.Context(), r.Header.Get(settings.AstrBotAPIKeyHeader)) {
		writeError(w, http.StatusUnauthorized, "插件密钥无效")
		return
	}
	var body struct {
		PlayerName string `json:"playerName"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "请求体不是有效 JSON")
		return
	}
	response, err := s.shareService.CreatePlayerToken(r.Context(), body.PlayerName)
	s.writeHTTPErrorResult(w, http.StatusCreated, response, err)
}

func (s *Server) playerShareDetails(w http.ResponseWriter, r *http.Request) {
	response, err := s.shareService.PlayerDetails(r.Context(), r.PathValue("token"))
	s.writeHTTPErrorResult(w, http.StatusOK, response, err)
}

func (s *Server) createRankingShareToken(w http.ResponseWriter, r *http.Request) {
	if !s.authService.MatchesAstrBotKey(r.Context(), r.Header.Get(settings.AstrBotAPIKeyHeader)) {
		writeError(w, http.StatusUnauthorized, "插件密钥无效")
		return
	}
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "请求体不是有效 JSON")
		return
	}
	rankingType := bodyString(body, "type", "total")
	limit := bodyInt(body, "limit", 10)
	from, err := bodyDate(body, "fromDate")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	to, err := bodyDate(body, "toDate")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	response, err := s.shareService.CreateRankingToken(r.Context(), rankingType, limit, from, to)
	s.writeHTTPErrorResult(w, http.StatusCreated, response, err)
}

func (s *Server) rankingShareDetails(w http.ResponseWriter, r *http.Request) {
	response, err := s.shareService.RankingDetails(r.Context(), r.PathValue("token"))
	s.writeHTTPErrorResult(w, http.StatusOK, response, err)
}

func (s *Server) sendXrayToGroup(w http.ResponseWriter, r *http.Request) {
	var request sharemodel.XrayGroupSendRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "请求体不是有效 JSON")
		return
	}
	response, err := s.shareService.SendXrayToGroup(r.Context(), request)
	s.writeHTTPErrorResult(w, http.StatusCreated, response, err)
}

func (s *Server) xrayShareDetails(w http.ResponseWriter, r *http.Request) {
	response, err := s.shareService.XrayDetails(r.Context(), r.PathValue("token"))
	s.writeHTTPErrorResult(w, http.StatusOK, response, err)
}

func (s *Server) pendingXrayGroupMessages(w http.ResponseWriter, r *http.Request) {
	if !s.authService.MatchesAstrBotKey(r.Context(), r.Header.Get(settings.AstrBotAPIKeyHeader)) {
		writeError(w, http.StatusUnauthorized, "插件密钥无效")
		return
	}
	response, err := s.shareService.PendingXrayGroupMessages(r.Context(), queryInt(r, "limit", 5))
	s.writeHTTPErrorResult(w, http.StatusOK, response, err)
}

func (s *Server) markXrayGroupDelivery(w http.ResponseWriter, r *http.Request) {
	if !s.authService.MatchesAstrBotKey(r.Context(), r.Header.Get(settings.AstrBotAPIKeyHeader)) {
		writeError(w, http.StatusUnauthorized, "插件密钥无效")
		return
	}
	messageID, err := strconv.ParseInt(r.PathValue("messageId"), 10, 64)
	if err != nil || messageID <= 0 {
		writeError(w, http.StatusBadRequest, "messageId is invalid")
		return
	}
	var request sharemodel.XrayGroupDeliveryRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "请求体不是有效 JSON")
		return
	}
	response, err := s.shareService.MarkXrayGroupDelivery(r.Context(), messageID, request.Success, request.ErrorMessage)
	s.writeHTTPErrorResult(w, http.StatusOK, response, err)
}

func (s *Server) writeHTTPErrorResult(w http.ResponseWriter, successStatus int, payload any, err error) {
	if err != nil {
		status, message := auth.ErrorStatus(err)
		if status == http.StatusInternalServerError {
			s.logger.Error("request failed", "error", err)
		}
		writeError(w, status, message)
		return
	}
	writeJSON(w, successStatus, payload)
}

func bodyString(body map[string]any, key, fallback string) string {
	value, ok := body[key]
	if !ok || value == nil {
		return fallback
	}
	text, ok := value.(string)
	if !ok {
		return fallback
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return fallback
	}
	return text
}

func bodyInt(body map[string]any, key string, fallback int) int {
	value, ok := body[key]
	if !ok || value == nil {
		return fallback
	}
	switch typed := value.(type) {
	case float64:
		return int(typed)
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err == nil {
			return parsed
		}
	}
	return fallback
}

func bodyDate(body map[string]any, key string) (*time.Time, error) {
	value, ok := body[key]
	if !ok || value == nil {
		return nil, nil
	}
	text, ok := value.(string)
	if !ok || strings.TrimSpace(text) == "" {
		return nil, nil
	}
	return stats.ParseDate(text)
}
