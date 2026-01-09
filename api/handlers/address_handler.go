package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"tmpemail_api/config"
	"tmpemail_api/database"
	"tmpemail_api/models"
)

// AddressHandler handles email address generation
type AddressHandler struct {
	db     *database.DB
	config *config.Config
	logger *slog.Logger
}

// NewAddressHandler creates a new address handler
func NewAddressHandler(db *database.DB, cfg *config.Config, logger *slog.Logger) *AddressHandler {
	return &AddressHandler{
		db:     db,
		config: cfg,
		logger: logger,
	}
}

// GenerateResponse represents the response for email address generation
type GenerateResponse struct {
	Address   string `json:"address"`
	ExpiresAt string `json:"expires_at"`
}

// Generate handles POST /api/generate - generates a new temporary email address
func (h *AddressHandler) Generate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Generate new email address
	emailAddr, err := models.NewEmailAddress(h.config.EmailDomain, h.config.DefaultExpiration)
	if err != nil {
		h.logger.Error("Failed to generate email address", "error", err)
		http.Error(w, "Failed to generate email address", http.StatusInternalServerError)
		return
	}

	// Insert into database
	if err := h.db.InsertAddress(emailAddr); err != nil {
		h.logger.Error("Failed to insert address into database", "error", err, "address", emailAddr.Address)
		http.Error(w, "Failed to save email address", http.StatusInternalServerError)
		return
	}

	h.logger.Info("Generated new email address", "address", emailAddr.Address, "expires_at", emailAddr.ExpiresAt)

	// Return response
	response := GenerateResponse{
		Address:   emailAddr.Address,
		ExpiresAt: emailAddr.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
