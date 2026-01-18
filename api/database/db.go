package database

import (
	"embed"
	"fmt"
	"log"
	"time"

	"tmpemail_api/models"

	"github.com/jmoiron/sqlx"
	// _ "github.com/mattn/go-sqlite3"
	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaFS embed.FS

// DB wraps the SQLx database connection
type DB struct {
	*sqlx.DB
}

// InitDB initializes the SQLite database with the schema
func InitDB(dbPath string) (*DB, error) {
	// Open SQLite database
	db, err := sqlx.Open("sqlite", fmt.Sprintf("%s?_foreign_keys=on&_journal_mode=WAL", dbPath))
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Read schema from embedded file
	schemaSQL, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return nil, fmt.Errorf("failed to read schema.sql: %w", err)
	}

	// Execute schema
	if _, err := db.Exec(string(schemaSQL)); err != nil {
		return nil, fmt.Errorf("failed to execute schema: %w", err)
	}

	log.Println("Database initialized successfully")
	return &DB{db}, nil
}

// InsertAddress inserts a new email address into the database
func (db *DB) InsertAddress(addr *models.EmailAddress) error {
	query := `INSERT INTO email_addresses (id, address, created_at, expires_at)
	          VALUES (:id, :address, :created_at, :expires_at)`
	_, err := db.NamedExec(query, addr)
	if err != nil {
		return fmt.Errorf("failed to insert address: %w", err)
	}
	return nil
}

// GetAddress retrieves an email address by its address string
func (db *DB) GetAddress(address string) (*models.EmailAddress, error) {
	var addr models.EmailAddress
	query := `SELECT id, address, created_at, expires_at FROM email_addresses WHERE address = ?`
	err := db.Get(&addr, query, address)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get address: %w", err)
	}
	return &addr, nil
}

// IsValidAddress checks if an address exists and is not expired
func (db *DB) IsValidAddress(address string) (bool, bool, error) {
	addr, err := db.GetAddress(address)
	if err != nil {
		return false, false, err
	}
	if addr == nil {
		return false, false, nil // address doesn't exist
	}
	expired := addr.IsExpired()
	return true, expired, nil // valid, expired status, no error
}

// InsertEmail inserts a new email into the database
func (db *DB) InsertEmail(email *models.Email) error {
	query := `INSERT INTO emails (id, to_address, from_address, subject, body_preview, body_text, body_html, file_path, received_at)
	          VALUES (:id, :to_address, :from_address, :subject, :body_preview, :body_text, :body_html, :file_path, :received_at)`
	_, err := db.NamedExec(query, email)
	if err != nil {
		return fmt.Errorf("failed to insert email: %w", err)
	}
	return nil
}

// GetEmailsByAddress retrieves all emails for a given address, ordered by received_at DESC
func (db *DB) GetEmailsByAddress(address string) ([]*models.Email, error) {
	query := `SELECT id, to_address, from_address, subject, body_preview, body_text, body_html, file_path, received_at
	          FROM emails WHERE to_address = ? ORDER BY received_at DESC`
	var emails []*models.Email
	err := db.Select(&emails, query, address)
	if err != nil {
		return nil, fmt.Errorf("failed to query emails: %w", err)
	}
	return emails, nil
}

// GetEmailByID retrieves a single email by its ID and address
func (db *DB) GetEmailByID(address, emailID string) (*models.Email, error) {
	var email models.Email
	query := `SELECT id, to_address, from_address, subject, body_preview, body_text, body_html, file_path, received_at
	          FROM emails WHERE id = ? AND to_address = ?`
	err := db.Get(&email, query, emailID, address)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get email: %w", err)
	}
	return &email, nil
}

// InsertAttachment inserts a new attachment into the database
func (db *DB) InsertAttachment(att *models.Attachment) error {
	query := `INSERT INTO attachments (id, email_id, filename, filepath, size)
	          VALUES (:id, :email_id, :filename, :filepath, :size)`
	_, err := db.NamedExec(query, att)
	if err != nil {
		return fmt.Errorf("failed to insert attachment: %w", err)
	}
	return nil
}

// GetAttachmentsByEmailID retrieves all attachments for a given email
func (db *DB) GetAttachmentsByEmailID(emailID string) ([]*models.Attachment, error) {
	query := `SELECT id, email_id, filename, filepath, size FROM attachments WHERE email_id = ?`
	var attachments []*models.Attachment
	err := db.Select(&attachments, query, emailID)
	if err != nil {
		return nil, fmt.Errorf("failed to query attachments: %w", err)
	}
	return attachments, nil
}

// GetAttachmentByID retrieves a single attachment by ID and email ID
func (db *DB) GetAttachmentByID(emailID, attachmentID string) (*models.Attachment, error) {
	var att models.Attachment
	query := `SELECT id, email_id, filename, filepath, size FROM attachments WHERE id = ? AND email_id = ?`
	err := db.Get(&att, query, attachmentID, emailID)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get attachment: %w", err)
	}
	return &att, nil
}

// GetExpiredAddresses retrieves all expired email addresses
func (db *DB) GetExpiredAddresses() ([]*models.EmailAddress, error) {
	query := `SELECT id, address, created_at, expires_at FROM email_addresses WHERE expires_at < ?`
	var addresses []*models.EmailAddress
	err := db.Select(&addresses, query, time.Now().UTC())
	if err != nil {
		return nil, fmt.Errorf("failed to query expired addresses: %w", err)
	}
	return addresses, nil
}

// DeleteAddress deletes an email address and all its associated emails (cascade)
func (db *DB) DeleteAddress(address string) error {
	query := `DELETE FROM email_addresses WHERE address = ?`
	_, err := db.Exec(query, address)
	if err != nil {
		return fmt.Errorf("failed to delete address: %w", err)
	}
	return nil
}

// GetEmailFilePathsByAddress retrieves all email file paths for a given address
func (db *DB) GetEmailFilePathsByAddress(address string) ([]string, error) {
	query := `SELECT file_path FROM emails WHERE to_address = ?`
	var paths []string
	err := db.Select(&paths, query, address)
	if err != nil {
		return nil, fmt.Errorf("failed to query email file paths: %w", err)
	}
	return paths, nil
}

// GetAttachmentFilePathsByAddress retrieves all attachment file paths for emails belonging to an address
func (db *DB) GetAttachmentFilePathsByAddress(address string) ([]string, error) {
	query := `SELECT a.filepath FROM attachments a
	          INNER JOIN emails e ON a.email_id = e.id
	          WHERE e.to_address = ?`
	var paths []string
	err := db.Select(&paths, query, address)
	if err != nil {
		return nil, fmt.Errorf("failed to query attachment file paths: %w", err)
	}
	return paths, nil
}

// GetStorageUsedByAddress calculates total storage used by an email address in bytes
// This includes email body sizes (text + html) and attachment sizes
func (db *DB) GetStorageUsedByAddress(address string) (int64, error) {
	// Sum of email body sizes
	var emailSize int64
	emailQuery := `SELECT COALESCE(SUM(LENGTH(body_text) + LENGTH(body_html)), 0) FROM emails WHERE to_address = ?`
	err := db.Get(&emailSize, emailQuery, address)
	if err != nil {
		return 0, fmt.Errorf("failed to query email sizes: %w", err)
	}

	// Sum of attachment sizes
	var attachmentSize int64
	attachmentQuery := `SELECT COALESCE(SUM(a.size), 0) FROM attachments a
	                    INNER JOIN emails e ON a.email_id = e.id
	                    WHERE e.to_address = ?`
	err = db.Get(&attachmentSize, attachmentQuery, address)
	if err != nil {
		return 0, fmt.Errorf("failed to query attachment sizes: %w", err)
	}

	return emailSize + attachmentSize, nil
}

// EmailFilter represents filter criteria for email queries
type EmailFilter struct {
	FromAddress     string
	SubjectContains string
	Since           *time.Time
}

// GetEmailsByFilter retrieves emails for a given address with optional filters, ordered by received_at DESC
func (db *DB) GetEmailsByFilter(address string, filter EmailFilter) ([]*models.Email, error) {
	query := `SELECT id, to_address, from_address, subject, body_preview, body_text, body_html, file_path, received_at
	          FROM emails WHERE to_address = ?`

	args := []interface{}{address}

	// Add from_address filter if provided
	if filter.FromAddress != "" {
		query += " AND from_address = ?"
		args = append(args, filter.FromAddress)
	}

	// Add subject filter if provided (case-insensitive LIKE)
	if filter.SubjectContains != "" {
		query += " AND subject LIKE ?"
		args = append(args, "%"+filter.SubjectContains+"%")
	}

	// Add since filter if provided
	if filter.Since != nil {
		query += " AND received_at >= ?"
		args = append(args, filter.Since)
	}

	query += " ORDER BY received_at DESC"

	var emails []*models.Email
	err := db.Select(&emails, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query emails with filters: %w", err)
	}
	return emails, nil
}
