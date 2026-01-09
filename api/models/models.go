package models

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
)

// EmailAddress represents a temporary email address
type EmailAddress struct {
	ID        string    `db:"id" json:"id"`
	Address   string    `db:"address" json:"address"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	ExpiresAt time.Time `db:"expires_at" json:"expires_at"`
}

// Email represents a received email
type Email struct {
	ID          string    `db:"id" json:"id"`
	ToAddress   string    `db:"to_address" json:"to_address"`
	FromAddress string    `db:"from_address" json:"from_address"`
	Subject     string    `db:"subject" json:"subject"`
	BodyPreview string    `db:"body_preview" json:"body_preview"`
	BodyText    string    `db:"body_text" json:"body_text"`
	BodyHTML    string    `db:"body_html" json:"body_html"`
	FilePath    string    `db:"file_path" json:"file_path"`
	ReceivedAt  time.Time `db:"received_at" json:"received_at"`
}

// Attachment represents an email attachment
type Attachment struct {
	ID       string `db:"id" json:"id"`
	EmailID  string `db:"email_id" json:"email_id"`
	Filename string `db:"filename" json:"filename"`
	Filepath string `db:"filepath" json:"filepath"`
	Size     int64  `db:"size" json:"size"`
}

// Adjectives for readable email addresses
var adjectives = []string{
	"happy", "silly", "brave", "clever", "gentle", "kind", "wise", "calm", "jolly", "bright",
	"swift", "quiet", "loud", "smooth", "rough", "soft", "hard", "warm", "cool", "hot",
	"cold", "sweet", "sour", "salty", "spicy", "fresh", "stale", "new", "old", "young",
	"ancient", "modern", "simple", "complex", "easy", "hard", "light", "dark", "quick", "slow",
	"fast", "lazy", "active", "passive", "strong", "weak", "big", "small", "tall", "short",
}

// Nouns for readable email addresses
var nouns = []string{
	"cat", "dog", "bird", "fish", "turtle", "rabbit", "mouse", "lion", "tiger", "bear",
	"wolf", "fox", "deer", "moose", "eagle", "hawk", "owl", "duck", "goose", "swan",
	"frog", "toad", "snake", "lizard", "dragon", "unicorn", "phoenix", "pegasus", "griffin", "sphinx",
	"panda", "koala", "monkey", "gorilla", "zebra", "giraffe", "elephant", "rhino", "hippo", "camel",
	"dolphin", "whale", "shark", "octopus", "squid", "crab", "lobster", "shrimp", "starfish", "jellyfish",
}

// GenerateEmailAddress generates a random email address in the format: adjective-noun-number@domain
// where number is 4-6 digits
func GenerateEmailAddress(domain string) (string, error) {
	// Generate random adjective
	adjIdx, err := rand.Int(rand.Reader, big.NewInt(int64(len(adjectives))))
	if err != nil {
		return "", fmt.Errorf("failed to generate random adjective: %w", err)
	}
	adjective := adjectives[adjIdx.Int64()]

	// Generate random noun
	nounIdx, err := rand.Int(rand.Reader, big.NewInt(int64(len(nouns))))
	if err != nil {
		return "", fmt.Errorf("failed to generate random noun: %w", err)
	}
	noun := nouns[nounIdx.Int64()]

	// Generate random number between 1000 and 999999 (4-6 digits)
	minNum := int64(1000)
	maxNum := int64(999999)
	numRange := maxNum - minNum + 1
	randomNum, err := rand.Int(rand.Reader, big.NewInt(numRange))
	if err != nil {
		return "", fmt.Errorf("failed to generate random number: %w", err)
	}
	number := minNum + randomNum.Int64()

	// Construct the email address
	address := fmt.Sprintf("%s-%s-%d@%s", adjective, noun, number, domain)
	return strings.ToLower(address), nil
}

// NewEmailAddress creates a new EmailAddress with the given domain and expiration duration
func NewEmailAddress(domain string, expiresIn time.Duration) (*EmailAddress, error) {
	address, err := GenerateEmailAddress(domain)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	id := ulid.MustNew(ulid.Timestamp(now), rand.Reader)

	return &EmailAddress{
		ID:        id.String(),
		Address:   address,
		CreatedAt: now,
		ExpiresAt: now.Add(expiresIn),
	}, nil
}

// IsExpired checks if the email address has expired
func (e *EmailAddress) IsExpired() bool {
	return time.Now().UTC().After(e.ExpiresAt)
}

// NewEmail creates a new Email instance
func NewEmail(toAddress, fromAddress, subject, bodyPreview, bodyText, bodyHTML, filePath string) *Email {
	now := time.Now().UTC()
	id := ulid.MustNew(ulid.Timestamp(now), rand.Reader)

	return &Email{
		ID:          id.String(),
		ToAddress:   toAddress,
		FromAddress: fromAddress,
		Subject:     subject,
		BodyPreview: bodyPreview,
		BodyText:    bodyText,
		BodyHTML:    bodyHTML,
		FilePath:    filePath,
		ReceivedAt:  now,
	}
}

// NewAttachment creates a new Attachment instance
func NewAttachment(emailID, filename, filepath string, size int64) *Attachment {
	id := ulid.MustNew(ulid.Timestamp(time.Now().UTC()), rand.Reader)

	return &Attachment{
		ID:       id.String(),
		EmailID:  emailID,
		Filename: filename,
		Filepath: filepath,
		Size:     size,
	}
}
