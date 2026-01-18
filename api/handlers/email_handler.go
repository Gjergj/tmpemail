package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/microcosm-cc/bluemonday"

	"tmpemail_api/config"
	"tmpemail_api/database"
)

// EmailHandler handles email retrieval operations
type EmailHandler struct {
	db        *database.DB
	config    *config.Config
	logger    *slog.Logger
	sanitizer *bluemonday.Policy
}

// NewEmailHandler creates a new email handler
func NewEmailHandler(db *database.DB, cfg *config.Config, logger *slog.Logger) *EmailHandler {
	// Create HTML sanitizer to prevent XSS
	sanitizer := bluemonday.UGCPolicy()

	return &EmailHandler{
		db:        db,
		config:    cfg,
		logger:    logger,
		sanitizer: sanitizer,
	}
}

// EmailListResponse represents the list of emails for an address
type EmailListResponse struct {
	Emails []EmailSummary `json:"emails"`
}

// EmailSummary represents a summary of an email
type EmailSummary struct {
	ID             string `json:"id"`
	From           string `json:"from"`
	Subject        string `json:"subject"`
	Preview        string `json:"preview"`
	ReceivedAt     string `json:"received_at"`
	HasAttachments bool   `json:"has_attachments"`
}

// EmailContentResponse represents the full content of an email
type EmailContentResponse struct {
	ID          string           `json:"id"`
	From        string           `json:"from"`
	Subject     string           `json:"subject"`
	BodyHTML    string           `json:"body_html"`
	BodyText    string           `json:"body_text"`
	ReceivedAt  string           `json:"received_at"`
	Attachments []AttachmentInfo `json:"attachments"`
}

// AttachmentInfo represents attachment metadata
type AttachmentInfo struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
}

// AttachmentsResponse represents the list of attachments for an email
type AttachmentsResponse struct {
	Files []AttachmentInfo `json:"files"`
}

// GetEmails handles GET /api/v1/emails/{address} - retrieves all emails for an address
func (h *EmailHandler) GetEmails(w http.ResponseWriter, r *http.Request) {
	address := chi.URLParam(r, "address")
	if address == "" {
		http.Error(w, "Missing address parameter", http.StatusBadRequest)
		return
	}

	// Validate address exists and is not expired
	valid, expired, err := h.db.IsValidAddress(address)
	if err != nil {
		h.logger.Error("Failed to validate address", "error", err, "address", address)
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

	// Get emails
	emails, err := h.db.GetEmailsByAddress(address)
	if err != nil {
		h.logger.Error("Failed to get emails", "error", err, "address", address)
		http.Error(w, "Failed to retrieve emails", http.StatusInternalServerError)
		return
	}

	// Convert to summaries
	summaries := make([]EmailSummary, 0, len(emails))
	for _, email := range emails {
		// Check if email has attachments
		attachments, _ := h.db.GetAttachmentsByEmailID(email.ID)
		hasAttachments := len(attachments) > 0

		summaries = append(summaries, EmailSummary{
			ID:             email.ID,
			From:           email.FromAddress,
			Subject:        email.Subject,
			Preview:        email.BodyPreview,
			ReceivedAt:     email.ReceivedAt.Format("2006-01-02T15:04:05Z07:00"),
			HasAttachments: hasAttachments,
		})
	}

	response := EmailListResponse{Emails: summaries}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetEmailsFiltered handles GET /api/v1/emails/{address}/filter - retrieves emails with filters
func (h *EmailHandler) GetEmailsFiltered(w http.ResponseWriter, r *http.Request) {
	address := chi.URLParam(r, "address")
	if address == "" {
		http.Error(w, "Missing address parameter", http.StatusBadRequest)
		return
	}

	// Validate address exists and is not expired
	valid, expired, err := h.db.IsValidAddress(address)
	if err != nil {
		h.logger.Error("Failed to validate address", "error", err, "address", address)
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

	// Parse query parameters
	filter := database.EmailFilter{}

	// from parameter
	if from := r.URL.Query().Get("from"); from != "" {
		filter.FromAddress = from
	}

	// subject parameter (contains)
	if subject := r.URL.Query().Get("subject"); subject != "" {
		filter.SubjectContains = subject
	}

	// since parameter (RFC3339 format: 2006-01-02T15:04:05Z07:00)
	if since := r.URL.Query().Get("since"); since != "" {
		sinceTime, err := time.Parse(time.RFC3339, since)
		if err != nil {
			http.Error(w, "Invalid since parameter. Use RFC3339 format (e.g., 2006-01-02T15:04:05Z)", http.StatusBadRequest)
			return
		}
		filter.Since = &sinceTime
	}

	// Get filtered emails
	emails, err := h.db.GetEmailsByFilter(address, filter)
	if err != nil {
		h.logger.Error("Failed to get filtered emails", "error", err, "address", address, "filter", filter)
		http.Error(w, "Failed to retrieve emails", http.StatusInternalServerError)
		return
	}

	// Convert to summaries
	summaries := make([]EmailSummary, 0, len(emails))
	for _, email := range emails {
		// Check if email has attachments
		attachments, _ := h.db.GetAttachmentsByEmailID(email.ID)
		hasAttachments := len(attachments) > 0

		summaries = append(summaries, EmailSummary{
			ID:             email.ID,
			From:           email.FromAddress,
			Subject:        email.Subject,
			Preview:        email.BodyPreview,
			ReceivedAt:     email.ReceivedAt.Format("2006-01-02T15:04:05Z07:00"),
			HasAttachments: hasAttachments,
		})
	}

	response := EmailListResponse{Emails: summaries}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetEmailContent handles GET /api/v1/email/{address}/{emailID} - retrieves full email content
func (h *EmailHandler) GetEmailContent(w http.ResponseWriter, r *http.Request) {
	address := chi.URLParam(r, "address")
	emailID := chi.URLParam(r, "emailID")

	if address == "" || emailID == "" {
		http.Error(w, "Missing address or email ID parameter", http.StatusBadRequest)
		return
	}

	// Validate address
	valid, expired, err := h.db.IsValidAddress(address)
	if err != nil {
		h.logger.Error("Failed to validate address", "error", err, "address", address)
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

	// Get email
	email, err := h.db.GetEmailByID(address, emailID)
	if err != nil {
		h.logger.Error("Failed to get email", "error", err, "address", address, "email_id", emailID)
		http.Error(w, "Failed to retrieve email", http.StatusInternalServerError)
		return
	}

	if email == nil {
		http.Error(w, "Email not found", http.StatusNotFound)
		return
	}

	// Get attachments
	attachments, err := h.db.GetAttachmentsByEmailID(emailID)
	if err != nil {
		h.logger.Warn("Failed to get attachments", "error", err, "email_id", emailID)
		// Continue without attachments on error
	}

	// Convert attachments to response format
	attachmentInfos := make([]AttachmentInfo, 0, len(attachments))
	for _, att := range attachments {
		attachmentInfos = append(attachmentInfos, AttachmentInfo{
			ID:       att.ID,
			Filename: att.Filename,
		})
	}

	// Sanitize HTML content
	sanitizedHTML := h.sanitizer.Sanitize(email.BodyHTML)

	response := EmailContentResponse{
		ID:          email.ID,
		From:        email.FromAddress,
		Subject:     email.Subject,
		BodyHTML:    sanitizedHTML,
		BodyText:    email.BodyText,
		ReceivedAt:  email.ReceivedAt.Format("2006-01-02T15:04:05Z07:00"),
		Attachments: attachmentInfos,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetAttachments handles GET /api/v1/email/{address}/{emailID}/attachments - retrieves attachments list
func (h *EmailHandler) GetAttachments(w http.ResponseWriter, r *http.Request) {
	address := chi.URLParam(r, "address")
	emailID := chi.URLParam(r, "emailID")

	if address == "" || emailID == "" {
		http.Error(w, "Missing address or email ID parameter", http.StatusBadRequest)
		return
	}

	// Validate address
	valid, expired, err := h.db.IsValidAddress(address)
	if err != nil {
		h.logger.Error("Failed to validate address", "error", err, "address", address)
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

	// Verify email exists for this address
	email, err := h.db.GetEmailByID(address, emailID)
	if err != nil {
		h.logger.Error("Failed to get email", "error", err, "address", address, "email_id", emailID)
		http.Error(w, "Failed to retrieve email", http.StatusInternalServerError)
		return
	}

	if email == nil {
		http.Error(w, "Email not found", http.StatusNotFound)
		return
	}

	// Get attachments
	attachments, err := h.db.GetAttachmentsByEmailID(emailID)
	if err != nil {
		h.logger.Error("Failed to get attachments", "error", err, "email_id", emailID)
		http.Error(w, "Failed to retrieve attachments", http.StatusInternalServerError)
		return
	}

	// Convert to response format
	files := make([]AttachmentInfo, 0, len(attachments))
	for _, att := range attachments {
		files = append(files, AttachmentInfo{
			ID:       att.ID,
			Filename: att.Filename,
		})
	}

	response := AttachmentsResponse{Files: files}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// DownloadAttachment handles GET /api/v1/email/{address}/{emailID}/attachments/{attachmentID} - downloads attachment file
func (h *EmailHandler) DownloadAttachment(w http.ResponseWriter, r *http.Request) {
	address := chi.URLParam(r, "address")
	emailID := chi.URLParam(r, "emailID")
	attachmentID := chi.URLParam(r, "attachmentID")

	if address == "" || emailID == "" || attachmentID == "" {
		http.Error(w, "Missing required parameters", http.StatusBadRequest)
		return
	}

	// Validate address
	valid, expired, err := h.db.IsValidAddress(address)
	if err != nil {
		h.logger.Error("Failed to validate address", "error", err, "address", address)
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

	// Verify email exists for this address
	email, err := h.db.GetEmailByID(address, emailID)
	if err != nil {
		h.logger.Error("Failed to get email", "error", err, "address", address, "email_id", emailID)
		http.Error(w, "Failed to retrieve email", http.StatusInternalServerError)
		return
	}

	if email == nil {
		http.Error(w, "Email not found", http.StatusNotFound)
		return
	}

	// Get the specific attachment
	attachment, err := h.db.GetAttachmentByID(emailID, attachmentID)
	if err != nil {
		h.logger.Error("Failed to get attachment", "error", err, "email_id", emailID, "attachment_id", attachmentID)
		http.Error(w, "Failed to retrieve attachment", http.StatusInternalServerError)
		return
	}

	if attachment == nil {
		http.Error(w, "Attachment not found", http.StatusNotFound)
		return
	}

	// Security: Ensure the file path is within the storage directory
	cleanPath := filepath.Clean(attachment.Filepath)
	if !filepath.IsAbs(cleanPath) {
		cleanPath = filepath.Join(h.config.StoragePath, cleanPath)
	}

	// Open the file
	file, err := os.Open(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			h.logger.Warn("Attachment file not found", "path", cleanPath, "attachment_id", attachmentID)
			http.Error(w, "Attachment file not found", http.StatusNotFound)
			return
		}
		h.logger.Error("Failed to open attachment file", "error", err, "path", cleanPath)
		http.Error(w, "Failed to read attachment", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Get file info for size
	stat, err := file.Stat()
	if err != nil {
		h.logger.Error("Failed to stat attachment file", "error", err, "path", cleanPath)
		http.Error(w, "Failed to read attachment", http.StatusInternalServerError)
		return
	}

	// Determine content type from filename extension
	contentType := mime.TypeByExtension(filepath.Ext(attachment.Filename))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Set headers for file download
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, attachment.Filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", stat.Size()))
	w.Header().Set("Cache-Control", "private, max-age=3600")

	// Stream the file to the response
	if _, err := io.Copy(w, file); err != nil {
		h.logger.Error("Failed to stream attachment", "error", err, "attachment_id", attachmentID)
		// Can't send error response here as headers are already sent
		return
	}

	h.logger.Info("Served attachment", "attachment_id", attachmentID, "filename", attachment.Filename, "size", stat.Size())
}
