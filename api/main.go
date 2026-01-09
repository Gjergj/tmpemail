package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"tmpemail_api/cleanup"
	"tmpemail_api/config"
	"tmpemail_api/database"
	"tmpemail_api/handlers"
	"tmpemail_api/middleware"
	"tmpemail_api/websocket"
)

func main() {
	// Setup logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("Starting TmpEmail API Server")

	// Load configuration
	cfg := config.Load()
	logger.Info("Configuration loaded",
		"port", cfg.Port,
		"domain", cfg.EmailDomain,
		"cleanup_interval", cfg.CleanupInterval.String(),
	)

	// Ensure storage directory exists
	if err := os.MkdirAll(cfg.StoragePath, 0755); err != nil {
		logger.Error("Failed to create storage directory", "error", err, "path", cfg.StoragePath)
		os.Exit(1)
	}

	// Ensure database directory exists
	dbDir := filepath.Dir(cfg.DBPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		logger.Error("Failed to create database directory", "error", err, "path", dbDir)
		os.Exit(1)
	}

	// Initialize database
	db, err := database.InitDB(cfg.DBPath)
	if err != nil {
		logger.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("Database initialized", "path", cfg.DBPath)

	// Create WebSocket hub
	hub := websocket.NewHub(logger)
	go hub.Run()
	logger.Info("WebSocket hub started")

	// Create rate limiters for different endpoints
	generateRateLimiter := middleware.NewRateLimiterWithName(cfg.RateLimitGenerate, "generate")
	apiRateLimiter := middleware.NewRateLimiterWithName(cfg.RateLimitAPI, "api")
	wsRateLimiter := middleware.NewRateLimiterWithName(cfg.RateLimitWS, "websocket")

	// Start rate limiter cleanup goroutine
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			generateRateLimiter.Cleanup()
			apiRateLimiter.Cleanup()
			wsRateLimiter.Cleanup()
		}
	}()

	// Create handlers
	healthHandler := handlers.NewHealthHandler(db)
	addressHandler := handlers.NewAddressHandler(db, cfg, logger)
	emailHandler := handlers.NewEmailHandler(db, cfg, logger)
	internalHandler := handlers.NewInternalHandler(db, cfg, logger, hub)
	wsHandler := websocket.NewHandlerWithRateLimiter(hub, db, logger, wsRateLimiter)

	// Setup chi router
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimiddleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(middleware.CORS(cfg.AllowedOrigins))
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.StripSlashes)

	// Root endpoint
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","message":"TmpEmail API Server","version":"v1"}`))
	})

	// Health check endpoints (no rate limiting)
	r.Get("/health", healthHandler.Health)
	r.Get("/readiness", healthHandler.Readiness)

	// WebSocket endpoint (rate limiting handled in handler)
	r.Get("/ws", wsHandler.ServeWS)

	// ==========================================
	// API v1 routes
	// ==========================================
	r.Route("/api/v1", func(r chi.Router) {
		// Generate endpoint with stricter rate limiting
		r.With(generateRateLimiter.Middleware).Get("/generate", addressHandler.Generate)

		// Email endpoints with standard rate limiting
		r.With(apiRateLimiter.Middleware).Get("/emails/{address}", emailHandler.GetEmails)
		r.With(apiRateLimiter.Middleware).Get("/email/{address}/{emailID}", emailHandler.GetEmailContent)
		r.With(apiRateLimiter.Middleware).Get("/email/{address}/{emailID}/attachments", emailHandler.GetAttachments)
		r.With(apiRateLimiter.Middleware).Get("/email/{address}/{emailID}/attachments/{attachmentID}", emailHandler.DownloadAttachment)
	})

	// ==========================================
	// Internal routes (for Email Service)
	// ==========================================
	r.Route("/internal/v1", func(r chi.Router) {
		r.Get("/email/{address}", internalHandler.ValidateAddress)
		r.Post("/email/{address}/store", internalHandler.StoreEmail)
	})

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Create context for cleanup goroutine
	cleanupCtx, cleanupCancel := context.WithCancel(context.Background())
	defer cleanupCancel()

	// Start cleanup goroutine
	go cleanup.Start(cleanupCtx, db, cfg, logger)

	// Start server in a goroutine
	go func() {
		logger.Info("Server starting", "port", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Server shutting down...")

	// Stop cleanup goroutine
	cleanupCancel()

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
	}

	logger.Info("Server stopped")
}
