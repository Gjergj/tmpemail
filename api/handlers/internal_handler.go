package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"tmpemail_api/config"
	"tmpemail_api/database"
	"tmpemail_api/models"
	"tmpemail_api/websocket"
)

// InternalHandler handles internal API endpoints for Email Service communication
type InternalHandler struct {
	db     *database.DB
	config *config.Config
	logger *slog.Logger
	hub    *websocket.Hub
}

// NewInternalHandler creates a new internal handler
func NewInternalHandler(db *database.DB, cfg *config.Config, logger *slog.Logger, hub *websocket.Hub) *InternalHandler {
	return &InternalHandler{
		db:     db,
		config: cfg,
		logger: logger,
		hub:    hub,
	}
}

// ValidationResponse represents the response for address validation
type ValidationResponse struct {
	Valid        bool  `json:"valid"`
	Expired      bool  `json:"expired"`
	StorageUsed  int64 `json:"storage_used"`  // Current storage used in bytes
	StorageQuota int64 `json:"storage_quota"` // Max storage allowed in bytes (0 = unlimited)
}

// ValidateAddress handles GET /internal/email/{address} - validates if an address exists and is not expired
func (ih *InternalHandler) ValidateAddress(w http.ResponseWriter, r *http.Request) {
	address := chi.URLParam(r, "address")
	if address == "" {
		http.Error(w, "Missing address parameter", http.StatusBadRequest)
		return
	}

	// Validate address
	valid, expired, err := ih.db.IsValidAddress(address)
	if err != nil {
		ih.logger.Error("Failed to validate address", "error", err, "address", address)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Get storage used (only if address is valid)
	var storageUsed int64
	if valid {
		storageUsed, err = ih.db.GetStorageUsedByAddress(address)
		if err != nil {
			ih.logger.Error("Failed to get storage used", "error", err, "address", address)
			// Don't fail the request, just log and continue with 0
			storageUsed = 0
		}
	}

	response := ValidationResponse{
		Valid:        valid,
		Expired:      expired,
		StorageUsed:  storageUsed,
		StorageQuota: ih.config.StorageQuotaPerAddress,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// StoreEmailRequest represents the request to store an email
type StoreEmailRequest struct {
	To              string   `json:"to"`
	From            string   `json:"from"`
	Subject         string   `json:"subject"`
	BodyText        string   `json:"body_text"`
	BodyHTML        string   `json:"body_html"`
	RawEmail        string   `json:"raw_email"`
	FilePath        string   `json:"file_path"`
	Timestamp       string   `json:"timestamp"`
	AttachmentPaths []string `json:"attachment_paths"`
	AttachmentNames []string `json:"attachment_names"`
	AttachmentSizes []int64  `json:"attachment_sizes"`
}

// StoreEmailResponse represents the response for storing an email
type StoreEmailResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	EmailID string `json:"email_id,omitempty"`
}

// StoreEmail handles POST /internal/email/{address}/store - stores email from Email Service
func (ih *InternalHandler) StoreEmail(w http.ResponseWriter, r *http.Request) {
	address := chi.URLParam(r, "address")
	if address == "" {
		response := StoreEmailResponse{Success: false, Message: "Missing address parameter"}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Validate address exists and not expired
	valid, expired, err := ih.db.IsValidAddress(address)
	if err != nil {
		ih.logger.Error("Failed to validate address", "error", err, "address", address)
		response := StoreEmailResponse{Success: false, Message: "Failed to validate address"}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	if !valid {
		ih.logger.Warn("Attempted to store email for non-existent address", "address", address)
		response := StoreEmailResponse{Success: false, Message: "Email address does not exist"}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	if expired {
		ih.logger.Warn("Attempted to store email for expired address", "address", address)
		response := StoreEmailResponse{Success: false, Message: "Email address has expired"}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusGone)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Parse request body
	var req StoreEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ih.logger.Error("Failed to parse request body", "error", err)
		response := StoreEmailResponse{Success: false, Message: "Invalid request body"}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Generate preview (first 200 characters of text body)
	preview := req.BodyText
	if len(preview) > 200 {
		preview = preview[:200] + "..."
	}

	// Create email model
	email := models.NewEmail(
		address,
		req.From,
		req.Subject,
		preview,
		req.BodyText,
		req.BodyHTML,
		req.FilePath,
	)

	// Insert email into database
	if err := ih.db.InsertEmail(email); err != nil {
		ih.logger.Error("Failed to insert email", "error", err, "address", address)
		response := StoreEmailResponse{Success: false, Message: "Failed to store email"}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Insert attachments if any
	if len(req.AttachmentPaths) > 0 {
		for i, path := range req.AttachmentPaths {
			filename := ""
			size := int64(0)

			if i < len(req.AttachmentNames) {
				filename = req.AttachmentNames[i]
			}
			if i < len(req.AttachmentSizes) {
				size = req.AttachmentSizes[i]
			}

			att := models.NewAttachment(email.ID, filename, path, size)
			if err := ih.db.InsertAttachment(att); err != nil {
				ih.logger.Error("Failed to insert attachment", "error", err, "email_id", email.ID, "filename", filename)
				// Continue even if attachment insert fails
			}
		}
	}

	ih.logger.Info("Stored new email", "address", address, "email_id", email.ID, "from", req.From, "subject", req.Subject)

	// Broadcast to WebSocket clients
	ih.hub.BroadcastToAddress(address, websocket.Message{
		Type: "new_email",
		Data: map[string]interface{}{
			"id":          email.ID,
			"from":        email.FromAddress,
			"subject":     email.Subject,
			"preview":     email.BodyPreview,
			"received_at": email.ReceivedAt.Format("2006-01-02T15:04:05Z07:00"),
		},
	})

	// Return success response
	response := StoreEmailResponse{
		Success: true,
		Message: "Email stored successfully",
		EmailID: email.ID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
