package httpapi

import (
	"context"
	"net/http"
	"strings"

	"player-stats-backend-go/internal/auth"
)

type principalContextKey struct{}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions || !s.requiresAdminAuth(r) {
			next.ServeHTTP(w, r)
			return
		}
		if s.isPluginStatsRequest(r) && s.authService != nil && s.authService.MatchesAstrBotKey(r.Context(), r.Header.Get("X-Player-Stats-Key")) {
			next.ServeHTTP(w, r)
			return
		}
		if s.authService == nil {
			writeError(w, http.StatusUnauthorized, "请先登录管理员账号")
			return
		}
		principal, ok := s.authService.Authenticate(r.Context(), r.Header.Get("Authorization"))
		if !ok {
			writeError(w, http.StatusUnauthorized, "请先登录管理员账号")
			return
		}
		ctx := context.WithValue(r.Context(), principalContextKey{}, principal)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) isPluginStatsRequest(r *http.Request) bool {
	return r.Method == http.MethodGet && (r.URL.Path == "/api/stats/player" ||
		r.URL.Path == "/api/stats/player-presence" ||
		r.URL.Path == "/api/stats/players")
}

func (s *Server) requiresAdminAuth(r *http.Request) bool {
	path := r.URL.Path
	if strings.HasPrefix(path, "/api/import/") ||
		strings.HasPrefix(path, "/api/stats/") ||
		strings.HasPrefix(path, "/api/config/") {
		return true
	}
	if path == "/api/share/xray/send-to-group" ||
		path == "/api/auth/me" ||
		path == "/api/auth/password" ||
		path == "/api/auth/logout" {
		return true
	}
	return false
}

func PrincipalFromContext(ctx context.Context) (auth.Principal, bool) {
	principal, ok := ctx.Value(principalContextKey{}).(auth.Principal)
	return principal, ok
}
