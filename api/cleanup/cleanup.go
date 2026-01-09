package cleanup

import (
	"context"
	"log/slog"
	"os"
	"time"

	"tmpemail_api/config"
	"tmpemail_api/database"
)

// Start begins the cleanup goroutine that removes expired email addresses
func Start(ctx context.Context, db *database.DB, cfg *config.Config, logger *slog.Logger) {
	ticker := time.NewTicker(cfg.CleanupInterval)
	defer ticker.Stop()

	logger.Info("Cleanup job started", "interval", cfg.CleanupInterval.String())

	// Run cleanup immediately on start
	runCleanup(db, cfg, logger)

	for {
		select {
		case <-ticker.C:
			runCleanup(db, cfg, logger)
		case <-ctx.Done():
			logger.Info("Cleanup job stopping")
			return
		}
	}
}

// runCleanup performs the actual cleanup of expired addresses
func runCleanup(db *database.DB, cfg *config.Config, logger *slog.Logger) {
	logger.Info("Running cleanup job")

	// Get all expired addresses
	expiredAddresses, err := db.GetExpiredAddresses()
	if err != nil {
		logger.Error("Failed to get expired addresses", "error", err)
		return
	}

	if len(expiredAddresses) == 0 {
		logger.Info("No expired addresses to clean up")
		return
	}

	logger.Info("Found expired addresses", "count", len(expiredAddresses))

	cleanedCount := 0
	for _, addr := range expiredAddresses {
		if err := cleanupAddress(db, cfg, addr.Address, logger); err != nil {
			logger.Error("Failed to cleanup address", "error", err, "address", addr.Address)
			// Continue with next address even if this one failed
			continue
		}
		cleanedCount++
	}

	logger.Info("Cleanup job completed", "cleaned", cleanedCount, "failed", len(expiredAddresses)-cleanedCount)
}

// cleanupAddress removes a single email address and all its associated data
func cleanupAddress(db *database.DB, cfg *config.Config, address string, logger *slog.Logger) error {
	logger.Info("Cleaning up address", "address", address)

	// Get all email file paths for this address
	emailPaths, err := db.GetEmailFilePathsByAddress(address)
	if err != nil {
		return err
	}

	// Get all attachment file paths for this address
	attachmentPaths, err := db.GetAttachmentFilePathsByAddress(address)
	if err != nil {
		return err
	}

	// Delete email files from filesystem
	emailFilesDeleted := 0
	for _, path := range emailPaths {
		if err := os.Remove(path); err != nil {
			if !os.IsNotExist(err) {
				logger.Warn("Failed to delete email file", "error", err, "path", path)
			}
		} else {
			emailFilesDeleted++
		}
	}

	// Delete attachment files from filesystem
	attachmentFilesDeleted := 0
	for _, path := range attachmentPaths {
		if err := os.Remove(path); err != nil {
			if !os.IsNotExist(err) {
				logger.Warn("Failed to delete attachment file", "error", err, "path", path)
			}
		} else {
			attachmentFilesDeleted++
		}
	}

	// Delete address from database (cascade deletes emails and attachments)
	if err := db.DeleteAddress(address); err != nil {
		return err
	}

	logger.Info("Address cleaned up successfully",
		"address", address,
		"email_files_deleted", emailFilesDeleted,
		"attachment_files_deleted", attachmentFilesDeleted,
	)

	return nil
}
