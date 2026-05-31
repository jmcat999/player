package httpapi

import (
	"encoding/json"
	"net/http"
	"time"
)

type apiError struct {
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, apiError{
		Timestamp: time.Now().UTC(),
		Message:   message,
	})
}
