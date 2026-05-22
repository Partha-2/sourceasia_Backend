package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"backend-assignment/internal/ratelimiter"
)

type RateLimiterHandler struct {
	limiter *ratelimiter.RateLimiter
}

func NewRateLimiterHandler(limiter *ratelimiter.RateLimiter) *RateLimiterHandler {
	return &RateLimiterHandler{limiter: limiter}
}

type requestBody struct {
	UserID  string          `json:"user_id"`
	Payload json.RawMessage `json:"payload"`
}

type requestAccepted struct {
	Message   string `json:"message"`
	UserID    string `json:"user_id"`
	Timestamp string `json:"timestamp"`
}

type requestError struct {
	Error   string `json:"error"`
	UserID  string `json:"user_id,omitempty"`
	RetryAfter *int `json:"retry_after_seconds,omitempty"`
}

func (h *RateLimiterHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/request":
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		h.handlePostRequest(w, r)
	case "/stats":
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		h.handleGetStats(w, r)
	default:
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
	}
}

func (h *RateLimiterHandler) handlePostRequest(w http.ResponseWriter, r *http.Request) {
	var body requestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, requestError{Error: "invalid JSON body"})
		return
	}

	if body.UserID == "" {
		writeJSON(w, http.StatusBadRequest, requestError{Error: "user_id is required and must not be empty"})
		return
	}

	allowed, retryAfter := h.limiter.Allow(body.UserID)
	if !allowed {
		resp := requestError{
			Error:   "rate limit exceeded",
			UserID:  body.UserID,
			RetryAfter: &retryAfter,
		}
		writeJSON(w, http.StatusTooManyRequests, resp)
		return
	}

	resp := requestAccepted{
		Message:   "request accepted",
		UserID:    body.UserID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (h *RateLimiterHandler) handleGetStats(w http.ResponseWriter, r *http.Request) {
	stats := h.limiter.Stats()
	resp := map[string]interface{}{
		"users": stats,
	}
	writeJSON(w, http.StatusOK, resp)
}
