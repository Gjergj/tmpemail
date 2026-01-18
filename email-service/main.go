package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"blitiri.com.ar/go/spf"
	"github.com/emersion/go-msgauth/dkim"
	"github.com/emersion/go-msgauth/dmarc"
	"github.com/emersion/go-smtp"
	"github.com/jhillyerd/enmime"

	"tmpemail_email_service/client"
	"tmpemail_email_service/config"
	"tmpemail_email_service/storage"
)

// Backend implements SMTP backend
type Backend struct {
	storage   *storage.Storage
	apiClient *client.APIClient
	config    *config.Config
	logger    *slog.Logger
}

func NewBackend(storage *storage.Storage, apiClient *client.APIClient, cfg *config.Config, logger *slog.Logger) *Backend {
	return &Backend{
		storage:   storage,
		apiClient: apiClient,
		config:    cfg,
		logger:    logger,
	}
}

// NewSession creates a new SMTP session
func (b *Backend) NewSession(c *smtp.Conn) (smtp.Session, error) {
	// Extract client IP from connection
	clientIP := net.IP{}
	if addr := c.Conn().RemoteAddr(); addr != nil {
		if tcpAddr, ok := addr.(*net.TCPAddr); ok {
			clientIP = tcpAddr.IP
		}
	}

	return &Session{
		backend:  b,
		logger:   b.logger,
		clientIP: clientIP,
	}, nil
}

// recipientInfo holds validation data for a recipient
type recipientInfo struct {
	address      string
	storageUsed  int64
	storageQuota int64
}

// Session represents an SMTP session
type Session struct {
	backend    *Backend
	from       string
	recipients []recipientInfo
	logger     *slog.Logger
	clientIP   net.IP
}

// Mail is called when the MAIL FROM command is received
func (s *Session) Mail(from string, opts *smtp.MailOptions) error {
	s.from = from
	s.logger.Info("MAIL FROM received",
		"from", from,
		"client_ip", s.clientIP.String(),
	)
	return nil
}

// Rcpt is called when RCPT TO command is received
func (s *Session) Rcpt(to string, opts *smtp.RcptOptions) error {
	s.logger.Info("RCPT TO received",
		"to", to,
		"from", s.from,
		"client_ip", s.clientIP.String(),
	)

	// Extract email address from angle brackets if present
	address := extractEmailAddress(to)

	// Validate address with API Service
	validation, err := s.backend.apiClient.ValidateAddress(address)
	if err != nil {
		s.logger.Error("SMTP REJECT: Failed to validate address with API",
			"error", err,
			"address", address,
			"from", s.from,
			"client_ip", s.clientIP.String(),
			"smtp_code", 451,
		)
		return &smtp.SMTPError{
			Code:    451,
			Message: "Temporary failure validating address",
		}
	}

	if !validation.Valid {
		s.logger.Warn("SMTP REJECT: Invalid email address (not found)",
			"address", address,
			"from", s.from,
			"client_ip", s.clientIP.String(),
			"smtp_code", 550,
		)
		return &smtp.SMTPError{
			Code:    550,
			Message: "Recipient address rejected: User unknown",
		}
	}

	if validation.Expired {
		s.logger.Warn("SMTP REJECT: Expired email address",
			"address", address,
			"from", s.from,
			"client_ip", s.clientIP.String(),
			"smtp_code", 550,
		)
		return &smtp.SMTPError{
			Code:    550,
			Message: "Recipient address rejected: Address expired",
		}
	}

	s.logger.Info("Recipient accepted",
		"address", address,
		"storage_used", validation.StorageUsed,
		"storage_quota", validation.StorageQuota,
	)

	// Store recipient with quota info
	s.recipients = append(s.recipients, recipientInfo{
		address:      address,
		storageUsed:  validation.StorageUsed,
		storageQuota: validation.StorageQuota,
	})
	return nil
}

// Data is called when the DATA command is received
func (s *Session) Data(r io.Reader) error {
	if len(s.recipients) == 0 {
		s.logger.Warn("SMTP REJECT: No valid recipients",
			"from", s.from,
			"client_ip", s.clientIP.String(),
			"smtp_code", 554,
		)
		return &smtp.SMTPError{
			Code:    554,
			Message: "No valid recipients",
		}
	}

	s.logger.Info("DATA command received, reading email content",
		"from", s.from,
		"recipients", len(s.recipients),
		"client_ip", s.clientIP.String(),
	)

	// Read email data with size limit
	limitReader := io.LimitReader(r, int64(s.backend.config.MaxEmailSize))
	rawEmail, err := io.ReadAll(limitReader)
	if err != nil {
		s.logger.Error("SMTP REJECT: Failed to read email data",
			"error", err,
			"from", s.from,
			"recipients", len(s.recipients),
			"client_ip", s.clientIP.String(),
			"smtp_code", 451,
		)
		return &smtp.SMTPError{
			Code:    451,
			Message: "Failed to read email data",
		}
	}

	// Check if email exceeds size limit
	if len(rawEmail) >= s.backend.config.MaxEmailSize {
		recipientAddrs := make([]string, len(s.recipients))
		for i, r := range s.recipients {
			recipientAddrs[i] = r.address
		}
		s.logger.Warn("SMTP REJECT: Email exceeds size limit",
			"size", len(rawEmail),
			"max_size", s.backend.config.MaxEmailSize,
			"from", s.from,
			"to", recipientAddrs,
			"client_ip", s.clientIP.String(),
			"smtp_code", 552,
		)
		return &smtp.SMTPError{
			Code:    552,
			Message: "Email exceeds maximum size (20MB)",
		}
	}

	emailSize := int64(len(rawEmail))
	recipientAddrs := make([]string, len(s.recipients))
	for i, r := range s.recipients {
		recipientAddrs[i] = r.address
	}
	s.logger.Info("Email data received successfully",
		"from", s.from,
		"to", recipientAddrs,
		"recipients_count", len(s.recipients),
		"size_bytes", emailSize,
		"client_ip", s.clientIP.String(),
	)

	// Perform email authentication validation (SPF/DKIM/DMARC)
	cfg := s.backend.config
	if cfg.ValidateSPF || cfg.ValidateDKIM || cfg.ValidateDMARC {
		authResult := s.validateEmailAuth(rawEmail)

		// Check if we should reject the email based on policy
		if s.shouldRejectEmail(authResult) {
			s.logger.Warn("SMTP REJECT: Email authentication failed",
				"from", s.from,
				"to", recipientAddrs,
				"client_ip", s.clientIP.String(),
				"spf_result", authResult.SPFResult,
				"dkim_result", authResult.DKIMResult,
				"dmarc_result", authResult.DMARCResult,
				"policy", cfg.AuthPolicy,
				"smtp_code", 550,
			)
			return &smtp.SMTPError{
				Code:    550,
				Message: "Email rejected: authentication failed (SPF/DKIM/DMARC)",
			}
		}
	}

	// Process email for each recipient (check quota first)
	successCount := 0
	for _, rcpt := range s.recipients {
		// Check storage quota (0 = unlimited)
		if rcpt.storageQuota > 0 && rcpt.storageUsed+emailSize > rcpt.storageQuota {
			s.logger.Warn("SMTP WARN: Storage quota exceeded for recipient, skipping",
				"address", rcpt.address,
				"storage_used", rcpt.storageUsed,
				"storage_quota", rcpt.storageQuota,
				"email_size", emailSize,
				"would_use", rcpt.storageUsed+emailSize,
				"from", s.from,
				"client_ip", s.clientIP.String(),
			)
			// Skip this recipient but continue with others
			continue
		}

		if err := s.processEmail(rcpt.address, rawEmail); err != nil {
			s.logger.Error("Failed to process email for recipient",
				"error", err,
				"to", rcpt.address,
				"from", s.from,
				"client_ip", s.clientIP.String(),
			)
			// Continue processing other recipients even if one fails
		} else {
			successCount++
		}
	}

	s.logger.Info("Email processing completed",
		"from", s.from,
		"total_recipients", len(s.recipients),
		"successful", successCount,
		"failed", len(s.recipients)-successCount,
		"client_ip", s.clientIP.String(),
	)

	return nil
}

// processEmail handles storing and notifying the API about a new email
func (s *Session) processEmail(toAddress string, rawEmail []byte) error {
	s.logger.Info("Processing email for recipient",
		"to", toAddress,
		"from", s.from,
		"size_bytes", len(rawEmail),
	)

	// Save email to filesystem
	filePath, err := s.backend.storage.SaveEmail(toAddress, rawEmail)
	if err != nil {
		s.logger.Error("Failed to save email to filesystem",
			"error", err,
			"to", toAddress,
			"from", s.from,
			"size_bytes", len(rawEmail),
		)
		return fmt.Errorf("failed to save email: %w", err)
	}

	s.logger.Info("Email saved to filesystem",
		"path", filePath,
		"to", toAddress,
		"from", s.from,
	)

	// Parse email using enmime - much more robust MIME parsing
	env, err := enmime.ReadEnvelope(bytes.NewReader(rawEmail))
	if err != nil {
		s.logger.Warn("Failed to parse email with enmime",
			"error", err,
			"to", toAddress,
			"from", s.from,
		)
		// Create empty envelope for fallback
		env = &enmime.Envelope{}
	}

	// Log any parsing errors (enmime captures them instead of failing)
	if len(env.Errors) > 0 {
		s.logger.Warn("MIME parsing encountered issues",
			"error_count", len(env.Errors),
			"to", toAddress,
			"from", s.from,
		)
		for i, parseErr := range env.Errors {
			s.logger.Debug("MIME parsing issue detail",
				"issue_number", i+1,
				"error", parseErr.String(),
				"to", toAddress,
			)
		}
	}

	// Extract email components - enmime handles charset decoding automatically
	subject := env.GetHeader("Subject")
	fromHeader := env.GetHeader("From")
	if fromHeader == "" {
		fromHeader = s.from
	}

	// Get body text and HTML - enmime extracts these automatically
	bodyText := env.Text
	bodyHTML := env.HTML

	// Save attachments - enmime already parsed them
	attachmentPaths := []string{}
	attachmentNames := []string{}
	attachmentSizes := []int64{}

	emailFilename := filepath.Base(filePath)

	// Process regular attachments
	s.logger.Info("Processing attachments",
		"attachment_count", len(env.Attachments),
		"inline_count", len(env.Inlines),
		"to", toAddress,
	)

	for _, att := range env.Attachments {
		filename := att.FileName
		if filename == "" {
			filename = "unnamed"
		}
		attPath, err := s.backend.storage.SaveAttachment(emailFilename, filename, att.Content)
		if err != nil {
			s.logger.Error("Failed to save attachment",
				"error", err,
				"filename", filename,
				"size_bytes", len(att.Content),
				"content_type", att.ContentType,
				"to", toAddress,
				"from", s.from,
			)
			continue
		}
		attachmentPaths = append(attachmentPaths, attPath)
		attachmentNames = append(attachmentNames, filename)
		attachmentSizes = append(attachmentSizes, int64(len(att.Content)))

		s.logger.Info("Attachment saved successfully",
			"path", attPath,
			"filename", filename,
			"size_bytes", len(att.Content),
			"content_type", att.ContentType,
			"to", toAddress,
		)
	}

	// Process inline attachments (images embedded in HTML, etc.)
	for _, att := range env.Inlines {
		filename := att.FileName
		if filename == "" {
			filename = "inline_" + att.ContentID
		}
		attPath, err := s.backend.storage.SaveAttachment(emailFilename, filename, att.Content)
		if err != nil {
			s.logger.Error("Failed to save inline attachment",
				"error", err,
				"filename", filename,
				"size_bytes", len(att.Content),
				"content_id", att.ContentID,
				"to", toAddress,
				"from", s.from,
			)
			continue
		}
		attachmentPaths = append(attachmentPaths, attPath)
		attachmentNames = append(attachmentNames, filename)
		attachmentSizes = append(attachmentSizes, int64(len(att.Content)))

		s.logger.Info("Inline attachment saved successfully",
			"path", attPath,
			"filename", filename,
			"size_bytes", len(att.Content),
			"content_id", att.ContentID,
			"to", toAddress,
		)
	}

	// Store email via API
	storeReq := &client.StoreEmailRequest{
		To:              toAddress,
		From:            fromHeader,
		Subject:         subject,
		BodyText:        bodyText,
		BodyHTML:        bodyHTML,
		RawEmail:        string(rawEmail),
		FilePath:        filePath,
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		AttachmentPaths: attachmentPaths,
		AttachmentNames: attachmentNames,
		AttachmentSizes: attachmentSizes,
	}

	s.logger.Info("Storing email metadata via API",
		"to", toAddress,
		"from", fromHeader,
		"subject", subject,
		"attachment_count", len(attachmentPaths),
	)

	resp, err := s.backend.apiClient.StoreEmail(toAddress, storeReq)
	if err != nil {
		// Just log the error, don't break the operation - email is already saved to filesystem
		s.logger.Error("Failed to store email metadata via API (email saved to filesystem)",
			"error", err,
			"to", toAddress,
			"from", fromHeader,
			"subject", subject,
			"file_path", filePath,
			"client_ip", s.clientIP.String(),
		)
		return nil
	}

	s.logger.Info("Email stored successfully in database",
		"to", toAddress,
		"from", fromHeader,
		"subject", subject,
		"email_id", resp.EmailID,
		"file_path", filePath,
		"attachment_count", len(attachmentPaths),
	)
	return nil
}

// Reset is called when RSET command is received
func (s *Session) Reset() {
	s.logger.Info("RSET command received, resetting session",
		"client_ip", s.clientIP.String(),
		"previous_from", s.from,
		"previous_recipients", len(s.recipients),
	)
	s.from = ""
	s.recipients = nil
}

// Logout is called when the session is closed
func (s *Session) Logout() error {
	s.logger.Info("Session closed",
		"client_ip", s.clientIP.String(),
	)
	return nil
}

// extractEmailAddress extracts email from format like "<user@domain.com>" or "User <user@domain.com>"
func extractEmailAddress(address string) string {
	// Remove angle brackets if present
	address = strings.TrimSpace(address)
	if strings.Contains(address, "<") && strings.Contains(address, ">") {
		start := strings.Index(address, "<")
		end := strings.Index(address, ">")
		if start < end {
			address = address[start+1 : end]
		}
	}
	return strings.TrimSpace(address)
}

// extractDomain extracts the domain from an email address
func extractDomain(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) == 2 {
		return parts[1]
	}
	return ""
}

// AuthResult holds the result of email authentication checks
type AuthResult struct {
	SPFResult   string // pass, fail, softfail, neutral, none, temperror, permerror
	DKIMResult  string // pass, fail, none
	DMARCResult string // pass, fail, none
	SPFError    error
	DKIMError   error
	DMARCError  error
}

// validateEmailAuth performs SPF, DKIM, and DMARC validation
func (s *Session) validateEmailAuth(rawEmail []byte) *AuthResult {
	result := &AuthResult{
		SPFResult:   "none",
		DKIMResult:  "none",
		DMARCResult: "none",
	}

	cfg := s.backend.config
	senderDomain := extractDomain(s.from)

	// SPF Validation
	if cfg.ValidateSPF && senderDomain != "" && s.clientIP != nil {
		spfResult, err := spf.CheckHostWithSender(s.clientIP, "localhost", s.from)
		if err != nil {
			result.SPFError = err
			result.SPFResult = "temperror"
			s.logger.Warn("SPF check error", "error", err, "sender", s.from, "ip", s.clientIP.String())
		} else {
			result.SPFResult = spfResultToString(spfResult)
			s.logger.Info("SPF check completed", "result", result.SPFResult, "sender", s.from, "ip", s.clientIP.String())
		}
	}

	// DKIM Validation
	if cfg.ValidateDKIM {
		verifications, err := dkim.Verify(bytes.NewReader(rawEmail))
		if err != nil {
			result.DKIMError = err
			result.DKIMResult = "temperror"
			s.logger.Warn("DKIM verification error", "error", err)
		} else if len(verifications) == 0 {
			result.DKIMResult = "none"
			s.logger.Info("DKIM check completed", "result", "none (no signatures)")
		} else {
			// Check if any signature passed
			allPassed := true
			for _, v := range verifications {
				if v.Err != nil {
					allPassed = false
					s.logger.Warn("DKIM signature failed", "domain", v.Domain, "error", v.Err)
				} else {
					s.logger.Info("DKIM signature passed", "domain", v.Domain)
				}
			}
			if allPassed {
				result.DKIMResult = "pass"
			} else {
				result.DKIMResult = "fail"
			}
		}
	}

	// DMARC Validation
	if cfg.ValidateDMARC && senderDomain != "" {
		dmarcRecord, err := dmarc.Lookup(senderDomain)
		if err != nil {
			if err == dmarc.ErrNoPolicy {
				result.DMARCResult = "none"
				s.logger.Info("DMARC check completed", "result", "none (no policy)", "domain", senderDomain)
			} else {
				result.DMARCError = err
				result.DMARCResult = "temperror"
				s.logger.Warn("DMARC lookup error", "error", err, "domain", senderDomain)
			}
		} else {
			// Evaluate DMARC based on SPF and DKIM results
			spfAligned := result.SPFResult == "pass"
			dkimAligned := result.DKIMResult == "pass"

			if spfAligned || dkimAligned {
				result.DMARCResult = "pass"
			} else {
				result.DMARCResult = "fail"
			}

			s.logger.Info("DMARC check completed",
				"result", result.DMARCResult,
				"policy", dmarcRecord.Policy,
				"domain", senderDomain,
				"spf_aligned", spfAligned,
				"dkim_aligned", dkimAligned,
			)
		}
	}

	return result
}

// spfResultToString converts SPF result to string
func spfResultToString(result spf.Result) string {
	switch result {
	case spf.Pass:
		return "pass"
	case spf.Fail:
		return "fail"
	case spf.SoftFail:
		return "softfail"
	case spf.Neutral:
		return "neutral"
	case spf.None:
		return "none"
	case spf.TempError:
		return "temperror"
	case spf.PermError:
		return "permerror"
	default:
		return "unknown"
	}
}

// shouldRejectEmail determines if email should be rejected based on auth results and policy
func (s *Session) shouldRejectEmail(authResult *AuthResult) bool {
	cfg := s.backend.config

	// Only reject if policy is "reject"
	if cfg.AuthPolicy != "reject" {
		s.logger.Info("Email authentication checked (policy: log only)",
			"policy", cfg.AuthPolicy,
			"spf_result", authResult.SPFResult,
			"dkim_result", authResult.DKIMResult,
			"dmarc_result", authResult.DMARCResult,
			"from", s.from,
		)
		return false
	}

	// Check each enabled validation
	if cfg.ValidateSPF && (authResult.SPFResult == "fail" || authResult.SPFResult == "permerror") {
		s.logger.Warn("Rejecting email due to SPF failure",
			"result", authResult.SPFResult,
			"from", s.from,
			"client_ip", s.clientIP.String(),
			"spf_error", authResult.SPFError,
		)
		return true
	}

	if cfg.ValidateDKIM && authResult.DKIMResult == "fail" {
		s.logger.Warn("Rejecting email due to DKIM failure",
			"result", authResult.DKIMResult,
			"from", s.from,
			"client_ip", s.clientIP.String(),
			"dkim_error", authResult.DKIMError,
		)
		return true
	}

	if cfg.ValidateDMARC && authResult.DMARCResult == "fail" {
		s.logger.Warn("Rejecting email due to DMARC failure",
			"result", authResult.DMARCResult,
			"from", s.from,
			"client_ip", s.clientIP.String(),
			"dmarc_error", authResult.DMARCError,
		)
		return true
	}

	s.logger.Info("Email authentication passed",
		"spf_result", authResult.SPFResult,
		"dkim_result", authResult.DKIMResult,
		"dmarc_result", authResult.DMARCResult,
		"from", s.from,
	)

	return false
}

// HealthServer provides HTTP health check endpoints
type HealthServer struct {
	apiClient *client.APIClient
	logger    *slog.Logger
	ready     *atomic.Bool
}

// NewHealthServer creates a new health server
func NewHealthServer(apiClient *client.APIClient, logger *slog.Logger) *HealthServer {
	ready := &atomic.Bool{}
	ready.Store(false)
	return &HealthServer{
		apiClient: apiClient,
		logger:    logger,
		ready:     ready,
	}
}

// SetReady marks the server as ready
func (h *HealthServer) SetReady(ready bool) {
	h.ready.Store(ready)
}

// healthResponse represents the health check response
type healthResponse struct {
	Status    string `json:"status"`
	Service   string `json:"service"`
	Timestamp string `json:"timestamp"`
}

// readinessResponse represents the readiness check response
type readinessResponse struct {
	Status    string            `json:"status"`
	Service   string            `json:"service"`
	Timestamp string            `json:"timestamp"`
	Checks    map[string]string `json:"checks"`
}

// HealthHandler returns a simple liveness check
func (h *HealthServer) HealthHandler(w http.ResponseWriter, r *http.Request) {
	resp := healthResponse{
		Status:    "ok",
		Service:   "tmpemail-email-service",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// ReadinessHandler checks if the service is ready to receive traffic
func (h *HealthServer) ReadinessHandler(w http.ResponseWriter, r *http.Request) {
	checks := make(map[string]string)
	allHealthy := true

	// Check if SMTP server is ready
	if h.ready.Load() {
		checks["smtp_server"] = "ok"
	} else {
		checks["smtp_server"] = "not_ready"
		allHealthy = false
	}

	// Check API connectivity
	_, err := h.apiClient.ValidateAddress("health-check-test@tmpemail.xyz")
	if err != nil {
		// This might fail with "user unknown" which is expected,
		// we just want to check connectivity
		if strings.Contains(err.Error(), "failed to send request") ||
			strings.Contains(err.Error(), "connection refused") {
			checks["api_connectivity"] = "failed: " + err.Error()
			allHealthy = false
		} else {
			// API is reachable, just returned an error for invalid address
			checks["api_connectivity"] = "ok"
		}
	} else {
		checks["api_connectivity"] = "ok"
	}

	status := "ok"
	statusCode := http.StatusOK
	if !allHealthy {
		status = "degraded"
		statusCode = http.StatusServiceUnavailable
	}

	resp := readinessResponse{
		Status:    status,
		Service:   "tmpemail-email-service",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Checks:    checks,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(resp)
}

func main() {
	// Setup logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("Starting TmpEmail Email Service (SMTP Server)")

	// Load configuration
	cfg := config.Load()
	logger.Info("Configuration loaded",
		"smtp_port", cfg.SMTPPort,
		"health_port", cfg.HealthPort,
		"storage_path", cfg.StoragePath,
		"api_url", cfg.APIServiceURL,
		"tls_enabled", cfg.TLSEnabled,
		"validate_spf", cfg.ValidateSPF,
		"validate_dkim", cfg.ValidateDKIM,
		"validate_dmarc", cfg.ValidateDMARC,
		"auth_policy", cfg.AuthPolicy,
	)

	// Ensure storage directory exists
	if err := os.MkdirAll(cfg.StoragePath, 0755); err != nil {
		logger.Error("Failed to create storage directory", "error", err)
		os.Exit(1)
	}

	// Initialize components
	stor := storage.NewStorage(cfg.StoragePath)
	apiClient := client.NewAPIClient(cfg.APIServiceURL)

	// Create health server
	healthServer := NewHealthServer(apiClient, logger)

	// Setup HTTP health check server
	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/health", healthServer.HealthHandler)
	httpMux.HandleFunc("/readiness", healthServer.ReadinessHandler)

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.HealthPort),
		Handler:      httpMux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	// Start HTTP health server in goroutine
	go func() {
		logger.Info("Health check HTTP server starting", "port", cfg.HealthPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Health check HTTP server failed", "error", err)
		}
	}()

	// Create SMTP backend
	backend := NewBackend(stor, apiClient, cfg, logger)

	// Create SMTP server
	smtpServer := smtp.NewServer(backend)
	smtpServer.Addr = fmt.Sprintf("%s:%s", cfg.SMTPHost, cfg.SMTPPort)
	smtpServer.Domain = "tmpemail.xyz"
	smtpServer.MaxMessageBytes = int64(cfg.MaxEmailSize)
	smtpServer.MaxRecipients = 50
	smtpServer.AllowInsecureAuth = true

	// Configure TLS/STARTTLS if enabled
	if cfg.TLSEnabled {
		cert, err := tls.LoadX509KeyPair(cfg.TLSCertPath, cfg.TLSKeyPath)
		if err != nil {
			logger.Error("Failed to load TLS certificate", "error", err, "cert", cfg.TLSCertPath, "key", cfg.TLSKeyPath)
			os.Exit(1)
		}

		smtpServer.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}

		logger.Info("STARTTLS enabled for SMTP server", "cert", cfg.TLSCertPath, "key", cfg.TLSKeyPath)
	}

	logger.Info("SMTP server configured", "addr", smtpServer.Addr, "tls_enabled", cfg.TLSEnabled)

	// Start SMTP server in goroutine
	go func() {
		logger.Info("SMTP server starting", "port", cfg.SMTPPort)
		// Mark as ready once the server starts listening
		healthServer.SetReady(true)
		if err := smtpServer.ListenAndServe(); err != nil {
			logger.Error("SMTP server failed", "error", err)
			healthServer.SetReady(false)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down servers...")

	// Shutdown HTTP server gracefully
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("Error shutting down HTTP server", "error", err)
	}

	// Close SMTP server
	if err := smtpServer.Close(); err != nil {
		logger.Error("Error closing SMTP server", "error", err)
	}

	logger.Info("Servers stopped")
}
