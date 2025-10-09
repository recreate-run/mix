package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// SupabaseProvider implements Provider interface for Supabase Storage using native REST API
type SupabaseProvider struct {
	projectURL string
	apiKey     string
	bucket     string
	httpClient *http.Client
}

// NewSupabaseProvider creates a new Supabase storage provider
func NewSupabaseProvider(cfg Config) (*SupabaseProvider, error) {
	// Validate required fields
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("supabase project URL is required")
	}
	if cfg.AccessKey == "" {
		return nil, fmt.Errorf("supabase API key is required")
	}
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("bucket name is required")
	}

	// Clean endpoint - remove trailing slash and any paths
	projectURL := strings.TrimSuffix(cfg.Endpoint, "/")
	projectURL = strings.TrimPrefix(projectURL, "https://")
	projectURL = strings.TrimPrefix(projectURL, "http://")
	if idx := strings.Index(projectURL, "/"); idx > 0 {
		projectURL = projectURL[:idx]
	}
	projectURL = "https://" + projectURL

	return &SupabaseProvider{
		projectURL: projectURL,
		apiKey:     cfg.AccessKey, // Supabase uses service_role key as access key
		bucket:     cfg.Bucket,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// Upload uploads a file to Supabase Storage
func (p *SupabaseProvider) Upload(ctx context.Context, key string, data io.Reader, contentType string) (*FileInfo, error) {
	// Read data into buffer (Supabase needs Content-Length)
	buf := new(bytes.Buffer)
	size, err := buf.ReadFrom(data)
	if err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}

	// Supabase Storage API: POST /storage/v1/object/{bucket}/{path}
	url := fmt.Sprintf("%s/storage/v1/object/%s/%s", p.projectURL, p.bucket, key)

	req, err := http.NewRequestWithContext(ctx, "POST", url, buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("x-upsert", "true") // Allow overwriting existing files
	req.ContentLength = size

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upload request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	return &FileInfo{
		Key:         key,
		Size:        size,
		ContentType: contentType,
		PublicURL:   p.GetPublicURL(key),
	}, nil
}

// Download downloads a file from Supabase Storage
func (p *SupabaseProvider) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	// Supabase Storage API: GET /storage/v1/object/{bucket}/{path}
	url := fmt.Sprintf("%s/storage/v1/object/%s/%s", p.projectURL, p.bucket, key)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download request failed: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("file not found: %s", key)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return nil, fmt.Errorf("download failed with status %d: %s", resp.StatusCode, string(body))
	}

	return resp.Body, nil
}

// Delete deletes a file from Supabase Storage
func (p *SupabaseProvider) Delete(ctx context.Context, key string) error {
	// Supabase Storage API: DELETE /storage/v1/object/{bucket}/{path}
	url := fmt.Sprintf("%s/storage/v1/object/%s/%s", p.projectURL, p.bucket, key)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("delete request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// List lists files with a given prefix
func (p *SupabaseProvider) List(ctx context.Context, prefix string) ([]*FileInfo, error) {
	// Supabase Storage API: POST /storage/v1/object/list/{bucket}
	url := fmt.Sprintf("%s/storage/v1/object/list/%s", p.projectURL, p.bucket)

	// Request body for listing with prefix
	reqBody := map[string]interface{}{
		"prefix": prefix,
		"limit":  1000,
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var objects []struct {
		Name     string `json:"name"`
		Metadata struct {
			Size        int64  `json:"size"`
			Mimetype    string `json:"mimetype"`
			ContentType string `json:"contentType"`
		} `json:"metadata"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&objects); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to FileInfo
	files := make([]*FileInfo, 0, len(objects))
	for _, obj := range objects {
		contentType := obj.Metadata.ContentType
		if contentType == "" {
			contentType = obj.Metadata.Mimetype
		}
		files = append(files, &FileInfo{
			Key:         obj.Name,
			Size:        obj.Metadata.Size,
			ContentType: contentType,
			PublicURL:   p.GetPublicURL(obj.Name),
		})
	}

	return files, nil
}

// Exists checks if a file exists
func (p *SupabaseProvider) Exists(ctx context.Context, key string) (bool, error) {
	// Try to get file info using HEAD request
	url := fmt.Sprintf("%s/storage/v1/object/%s/%s", p.projectURL, p.bucket, key)

	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("exists check failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// Supabase returns 404 or 400 for non-existent files
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusBadRequest {
		return false, nil
	}

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}

	return false, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
}

// GetPublicURL returns the public URL for a file
func (p *SupabaseProvider) GetPublicURL(key string) string {
	return fmt.Sprintf("%s/storage/v1/object/public/%s/%s", p.projectURL, p.bucket, key)
}

// GetPresignedURL returns a presigned URL for temporary access
func (p *SupabaseProvider) GetPresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	// Supabase Storage API: POST /storage/v1/object/sign/{bucket}/{path}
	url := fmt.Sprintf("%s/storage/v1/object/sign/%s/%s", p.projectURL, p.bucket, key)

	reqBody := map[string]interface{}{
		"expiresIn": int(expiry.Seconds()),
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("sign request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("sign failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		SignedURL string `json:"signedURL"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return p.projectURL + result.SignedURL, nil
}

// Close closes the provider
func (p *SupabaseProvider) Close() error {
	// HTTP client doesn't need explicit closing
	return nil
}
