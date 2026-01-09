package storage

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

// Storage handles email file storage
type Storage struct {
	basePath string
}

// NewStorage creates a new storage instance
func NewStorage(basePath string) *Storage {
	return &Storage{
		basePath: basePath,
	}
}

// SaveEmail saves an email to the filesystem and returns the file path
func (s *Storage) SaveEmail(toAddress string, rawEmail []byte) (string, error) {
	// Ensure storage directory exists
	if err := os.MkdirAll(s.basePath, 0755); err != nil {
		return "", fmt.Errorf("failed to create storage directory: %w", err)
	}

	// Generate filename hash
	filename, err := generateFilename(toAddress)
	if err != nil {
		return "", fmt.Errorf("failed to generate filename: %w", err)
	}

	filePath := filepath.Join(s.basePath, filename)

	// Write to temporary file first (atomic write)
	tempPath := filePath + ".tmp"
	if err := os.WriteFile(tempPath, rawEmail, 0644); err != nil {
		return "", fmt.Errorf("failed to write temporary file: %w", err)
	}

	// Rename to final path (atomic operation)
	if err := os.Rename(tempPath, filePath); err != nil {
		os.Remove(tempPath) // Clean up temp file on error
		return "", fmt.Errorf("failed to rename file: %w", err)
	}

	return filePath, nil
}

// SaveAttachment saves an attachment to the filesystem and returns the file path
func (s *Storage) SaveAttachment(emailFilename, attachmentName string, data []byte) (string, error) {
	// Ensure storage directory exists
	if err := os.MkdirAll(s.basePath, 0755); err != nil {
		return "", fmt.Errorf("failed to create storage directory: %w", err)
	}

	// Generate attachment filename: emailFilename_attachmentName
	// Remove .eml extension from email filename
	baseEmailName := emailFilename
	if len(baseEmailName) > 4 && baseEmailName[len(baseEmailName)-4:] == ".eml" {
		baseEmailName = baseEmailName[:len(baseEmailName)-4]
	}

	attachmentFilename := fmt.Sprintf("%s_%s", baseEmailName, sanitizeFilename(attachmentName))
	filePath := filepath.Join(s.basePath, attachmentFilename)

	// Write to temporary file first
	tempPath := filePath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write attachment: %w", err)
	}

	// Rename to final path
	if err := os.Rename(tempPath, filePath); err != nil {
		os.Remove(tempPath)
		return "", fmt.Errorf("failed to rename attachment: %w", err)
	}

	return filePath, nil
}

// generateFilename generates a secure filename using SHA256(timestamp + address + random)
func generateFilename(address string) (string, error) {
	// Generate random number between 1000 and 999999 (4-6 digits)
	minNum := int64(1000)
	maxNum := int64(999999)
	numRange := maxNum - minNum + 1
	randomNum, err := rand.Int(rand.Reader, big.NewInt(numRange))
	if err != nil {
		return "", err
	}
	randomValue := minNum + randomNum.Int64()

	// Create hash input: timestamp + address + random
	timestamp := time.Now().UTC().Format("20060102150405.000000")
	hashInput := fmt.Sprintf("%s%s%d", timestamp, address, randomValue)

	// Calculate SHA256 hash
	hash := sha256.Sum256([]byte(hashInput))
	hashStr := fmt.Sprintf("%x", hash)

	// Return filename with .eml extension
	return hashStr + ".eml", nil
}

// sanitizeFilename removes potentially dangerous characters from attachment filenames
func sanitizeFilename(filename string) string {
	// Simple sanitization - remove path separators and dangerous characters
	safe := ""
	for _, ch := range filename {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') || ch == '.' || ch == '-' || ch == '_' {
			safe += string(ch)
		} else {
			safe += "_"
		}
	}
	return safe
}

// ReadEmail reads an email from the filesystem
func (s *Storage) ReadEmail(filePath string) ([]byte, error) {
	return os.ReadFile(filePath)
}
