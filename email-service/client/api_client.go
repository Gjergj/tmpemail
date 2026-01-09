package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// APIClient handles communication with the API Service
type APIClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewAPIClient creates a new API client
func NewAPIClient(baseURL string) *APIClient {
	return &APIClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// ValidationResponse represents the address validation response
type ValidationResponse struct {
	Valid        bool  `json:"valid"`
	Expired      bool  `json:"expired"`
	StorageUsed  int64 `json:"storage_used"`  // Current storage used in bytes
	StorageQuota int64 `json:"storage_quota"` // Max storage allowed in bytes (0 = unlimited)
}

// ValidateAddress checks if an email address is valid and not expired
func (c *APIClient) ValidateAddress(address string) (*ValidationResponse, error) {
	url := fmt.Sprintf("%s/internal/v1/email/%s/", c.baseURL, address)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("validation request to %s failed: %s - %s", url, resp.Status, string(body))
	}

	var validation ValidationResponse
	if err := json.NewDecoder(resp.Body).Decode(&validation); err != nil {
		return nil, fmt.Errorf("failed to decode response from %s: %w", url, err)
	}

	return &validation, nil
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

// StoreEmailResponse represents the store email response
type StoreEmailResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	EmailID string `json:"email_id,omitempty"`
}

// StoreEmail sends email metadata to the API Service with retry logic
func (c *APIClient) StoreEmail(address string, req *StoreEmailRequest) (*StoreEmailResponse, error) {
	maxRetries := 3
	var lastErr error

	for attempt := range maxRetries {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			time.Sleep(backoff)
		}

		resp, err := c.doStoreEmail(address, req)
		if err == nil {
			return resp, nil
		}

		lastErr = err
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}

// doStoreEmail performs a single store email request
func (c *APIClient) doStoreEmail(address string, req *StoreEmailRequest) (*StoreEmailResponse, error) {
	url := fmt.Sprintf("%s/internal/v1/email/%s/store", c.baseURL, address)

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("store request failed: %s - %s", resp.Status, string(body))
	}

	var storeResp StoreEmailResponse
	if err := json.Unmarshal(body, &storeResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !storeResp.Success {
		return nil, fmt.Errorf("store failed: %s", storeResp.Message)
	}

	return &storeResp, nil
}
