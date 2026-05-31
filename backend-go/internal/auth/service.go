package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"player-stats-backend-go/internal/config"
)

const BearerPrefix = "Bearer "

type Service struct {
	db  *sql.DB
	cfg config.Config
}

type Principal struct {
	UserID    int64     `json:"userId"`
	Username  string    `json:"username"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token     string    `json:"token"`
	Username  string    `json:"username"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

type user struct {
	ID           int64
	Username     string
	PasswordHash string
	Enabled      bool
}

func NewService(db *sql.DB, cfg config.Config) *Service {
	return &Service{db: db, cfg: cfg}
}

func (s *Service) EnsureAdmin(ctx context.Context) error {
	username := strings.TrimSpace(s.cfg.AdminUsername)
	if username == "" {
		username = "admin"
	}
	var exists int
	err := s.db.QueryRowContext(ctx, `select count(*) from admin_users where lower(username) = lower(?)`, username).Scan(&exists)
	if err != nil {
		return err
	}
	if exists > 0 {
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(s.cfg.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	_, err = s.db.ExecContext(ctx, `
		insert into admin_users (username, password_hash, enabled, created_at, updated_at)
		values (?, ?, true, ?, ?)
	`, username, string(hash), now, now)
	return err
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (LoginResponse, error) {
	username := strings.TrimSpace(req.Username)
	if username == "" {
		return LoginResponse{}, errUnauthorized()
	}
	u, err := s.findUserByUsername(ctx, username)
	if err != nil {
		return LoginResponse{}, errUnauthorized()
	}
	if !u.Enabled || bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)) != nil {
		return LoginResponse{}, errUnauthorized()
	}
	issuedAt := time.Now().UTC()
	expiresAt := issuedAt.Add(time.Duration(max(1, s.cfg.AdminTokenTTLDays)) * 24 * time.Hour)
	token, err := s.createJWT(u, issuedAt, expiresAt)
	if err != nil {
		return LoginResponse{}, err
	}
	return LoginResponse{Token: token, Username: u.Username, ExpiresAt: expiresAt}, nil
}

func (s *Service) Authenticate(ctx context.Context, authorizationHeader string) (Principal, bool) {
	token := strings.TrimSpace(strings.TrimPrefix(authorizationHeader, BearerPrefix))
	if token == "" || token == authorizationHeader {
		return Principal{}, false
	}
	claims, ok := s.verifyJWT(token)
	if !ok || claims.ExpiresAt.Before(time.Now().UTC()) {
		return Principal{}, false
	}
	u, err := s.findUserByID(ctx, claims.UserID)
	if err != nil || !u.Enabled || u.Username != claims.Username {
		return Principal{}, false
	}
	return Principal{UserID: u.ID, Username: u.Username, ExpiresAt: claims.ExpiresAt}, true
}

func (s *Service) ChangePassword(ctx context.Context, principal Principal, req ChangePasswordRequest) error {
	if principal.UserID <= 0 {
		return errUnauthorized()
	}
	if len(req.NewPassword) < 8 {
		return NewHTTPError(400, "新密码至少 8 位")
	}
	if req.CurrentPassword == req.NewPassword {
		return NewHTTPError(400, "新密码不能和当前密码相同")
	}
	u, err := s.findUserByID(ctx, principal.UserID)
	if err != nil || !u.Enabled {
		return errUnauthorized()
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.CurrentPassword)) != nil {
		return NewHTTPError(400, "当前密码不正确")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
		update admin_users set password_hash = ?, updated_at = ? where id = ?
	`, string(hash), time.Now().UTC(), principal.UserID)
	return err
}

func (s *Service) MatchesAstrBotKey(ctx context.Context, apiKey string) bool {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return false
	}
	var stored sql.NullString
	if err := s.db.QueryRowContext(ctx, `select astrbot_api_key from sync_config where id = 1`).Scan(&stored); err != nil {
		return false
	}
	return stored.Valid && strings.TrimSpace(stored.String) != "" && subtle.ConstantTimeCompare([]byte(strings.TrimSpace(stored.String)), []byte(apiKey)) == 1
}

func (s *Service) findUserByUsername(ctx context.Context, username string) (user, error) {
	var u user
	err := s.db.QueryRowContext(ctx, `
		select id, username, password_hash, enabled
		from admin_users
		where lower(username) = lower(?)
	`, username).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Enabled)
	return u, err
}

func (s *Service) findUserByID(ctx context.Context, id int64) (user, error) {
	var u user
	err := s.db.QueryRowContext(ctx, `
		select id, username, password_hash, enabled
		from admin_users
		where id = ?
	`, id).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Enabled)
	return u, err
}

type jwtClaims struct {
	Issuer    string    `json:"iss"`
	Username  string    `json:"sub"`
	UserID    int64     `json:"uid"`
	IssuedAt  int64     `json:"iat"`
	ExpiresAt time.Time `json:"-"`
	Expires   int64     `json:"exp"`
}

func (s *Service) createJWT(u user, issuedAt, expiresAt time.Time) (string, error) {
	header, err := encodeJSON(map[string]any{"alg": "HS256", "typ": "JWT"})
	if err != nil {
		return "", err
	}
	payload, err := encodeJSON(jwtClaims{
		Issuer:   s.cfg.AdminJWTIssuer,
		Username: u.Username,
		UserID:   u.ID,
		IssuedAt: issuedAt.Unix(),
		Expires:  expiresAt.Unix(),
	})
	if err != nil {
		return "", err
	}
	signed := header + "." + payload
	return signed + "." + s.sign(signed), nil
}

func (s *Service) verifyJWT(token string) (jwtClaims, bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return jwtClaims{}, false
	}
	signed := parts[0] + "." + parts[1]
	expected, err := base64.RawURLEncoding.DecodeString(s.sign(signed))
	if err != nil {
		return jwtClaims{}, false
	}
	actual, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return jwtClaims{}, false
	}
	if subtle.ConstantTimeCompare(expected, actual) != 1 {
		return jwtClaims{}, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return jwtClaims{}, false
	}
	var claims jwtClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return jwtClaims{}, false
	}
	if claims.Issuer != s.cfg.AdminJWTIssuer || claims.UserID <= 0 || claims.Username == "" || claims.Expires <= 0 {
		return jwtClaims{}, false
	}
	claims.ExpiresAt = time.Unix(claims.Expires, 0).UTC()
	return claims, true
}

func encodeJSON(value any) (string, error) {
	bytes, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func (s *Service) sign(signedContent string) string {
	mac := hmac.New(sha256.New, []byte(s.cfg.AdminJWTSecret))
	mac.Write([]byte(signedContent))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func RandomToken(size int) (string, error) {
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

type HTTPError struct {
	Status  int
	Message string
}

func (e HTTPError) Error() string {
	return e.Message
}

func NewHTTPError(status int, message string) HTTPError {
	return HTTPError{Status: status, Message: message}
}

func ErrorStatus(err error) (int, string) {
	var httpErr HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.Status, httpErr.Message
	}
	return 500, "服务器内部错误"
}

func errUnauthorized() HTTPError {
	return NewHTTPError(401, "账号或密码错误")
}
