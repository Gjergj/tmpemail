package websocket

import (
	"log/slog"
	"net/http"

	"github.com/gorilla/websocket"

	"tmpemail_api/database"
	"tmpemail_api/middleware"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for now - in production, check allowed origins
		return true
	},
}

// Handler handles WebSocket connection upgrades
type Handler struct {
	hub         *Hub
	db          *database.DB
	logger      *slog.Logger
	rateLimiter *middleware.RateLimiter
}

// NewHandler creates a new WebSocket handler
func NewHandler(hub *Hub, db *database.DB, logger *slog.Logger) *Handler {
	return &Handler{
		hub:         hub,
		db:          db,
		logger:      logger,
		rateLimiter: nil,
	}
}

// NewHandlerWithRateLimiter creates a new WebSocket handler with rate limiting
func NewHandlerWithRateLimiter(hub *Hub, db *database.DB, logger *slog.Logger, rateLimiter *middleware.RateLimiter) *Handler {
	return &Handler{
		hub:         hub,
		db:          db,
		logger:      logger,
		rateLimiter: rateLimiter,
	}
}

// ServeWS handles WebSocket requests from clients
func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
	// Check rate limit if configured
	// Note: chi's RealIP middleware already sets r.RemoteAddr to the real client IP
	if h.rateLimiter != nil {
		if !h.rateLimiter.Allow(r.RemoteAddr) {
			h.logger.Warn("WebSocket rate limit exceeded", "ip", r.RemoteAddr)
			http.Error(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
			return
		}
	}

	// Extract email address from query params
	address := r.URL.Query().Get("address")
	if address == "" {
		http.Error(w, "Missing address parameter", http.StatusBadRequest)
		return
	}

	// Validate that address exists and is not expired
	valid, expired, err := h.db.IsValidAddress(address)
	if err != nil {
		h.logger.Error("Failed to validate address for WebSocket", "error", err, "address", address)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if !valid {
		http.Error(w, "Email address not found", http.StatusNotFound)
		return
	}

	if expired {
		http.Error(w, "Email address has expired", http.StatusGone)
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade WebSocket connection", "error", err, "address", address)
		return
	}

	// Create new client
	client := NewClient(conn, h.hub, address, h.logger)

	// Register client with hub
	h.hub.register <- client

	// Start client's pumps
	client.Start()

	h.logger.Info("WebSocket connection established", "address", address)
}
