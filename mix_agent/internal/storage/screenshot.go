package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"mix/internal/config"
)

// ScreenshotStorage handles saving screenshots to local or cloud storage
type ScreenshotStorage interface {
	Save(ctx context.Context, sessionID string, imageData []byte) (string, error)
}

// LocalStorage saves screenshots to the session's local directory
type LocalStorage struct {
	sessionStorageDir string
	baseURL           string // Base URL for generating HTTP URLs
}

// NewLocalStorage creates a new local storage handler
func NewLocalStorage(sessionStorageDir, baseURL string) *LocalStorage {
	return &LocalStorage{
		sessionStorageDir: sessionStorageDir,
		baseURL:           baseURL,
	}
}

// Save saves the screenshot to {sessionDir}/screenshots/{timestamp}.png
// Returns an HTTP URL to access the screenshot
func (l *LocalStorage) Save(ctx context.Context, sessionID string, imageData []byte) (string, error) {
	// Create screenshots directory within session directory
	screenshotDir := filepath.Join(l.sessionStorageDir, sessionID, "screenshots")
	if err := os.MkdirAll(screenshotDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create screenshot directory: %w", err)
	}

	// Generate filename with timestamp
	timestamp := time.Now().Format("20060102_150405_000")
	filename := fmt.Sprintf("%s.png", timestamp)
	filePath := filepath.Join(screenshotDir, filename)

	// Write image data to file
	if err := os.WriteFile(filePath, imageData, 0o644); err != nil {
		return "", fmt.Errorf("failed to write screenshot file: %w", err)
	}

	// Return HTTP URL
	screenshotURL := fmt.Sprintf("%s/api/sessions/%s/screenshots/%s", l.baseURL, sessionID, filename)
	return screenshotURL, nil
}

// S3Storage saves screenshots to AWS S3
type S3Storage struct {
	bucket string
	region string
	// TODO: Add S3 client when implementing
}

// NewS3Storage creates a new S3 storage handler
func NewS3Storage(bucket, region string) *S3Storage {
	return &S3Storage{
		bucket: bucket,
		region: region,
	}
}

// Save uploads the screenshot to S3 and returns the public URL
func (s *S3Storage) Save(ctx context.Context, sessionID string, imageData []byte) (string, error) {
	// TODO: Implement S3 upload
	// 1. Create S3 client with credentials
	// 2. Generate S3 key: screenshots/{sessionID}/{timestamp}.png
	// 3. Upload with PutObject
	// 4. Return public URL: https://{bucket}.s3.{region}.amazonaws.com/{key}
	return "", fmt.Errorf("S3 storage not yet implemented")
}

// NewStorage creates a storage handler based on available configuration
// Returns S3Storage if credentials exist, otherwise LocalStorage
func NewStorage(sessionStorageDir, baseURL string) ScreenshotStorage {
	// TODO: Check if S3 credentials are configured when S3 implementation is ready
	// cfg := config.Get()
	// if cfg has S3 config, return NewS3Storage(bucket, region)

	_ = config.Get() // Keep import for future use

	// For now, always use local storage until S3 is implemented
	return NewLocalStorage(sessionStorageDir, baseURL)
}
