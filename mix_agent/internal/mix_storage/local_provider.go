package storage

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LocalProvider implements Provider interface for local filesystem storage
type LocalProvider struct {
	basePath      string
	publicURLBase string
}

// NewLocalProvider creates a new local filesystem storage provider
func NewLocalProvider(cfg Config) (*LocalProvider, error) {
	// Ensure base path exists
	if err := os.MkdirAll(cfg.Endpoint, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	// Get absolute path
	absPath, err := filepath.Abs(cfg.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	return &LocalProvider{
		basePath:      absPath,
		publicURLBase: cfg.PublicURLBase,
	}, nil
}

// Upload uploads a file to local storage
func (p *LocalProvider) Upload(ctx context.Context, key string, data io.Reader, contentType string) (*FileInfo, error) {
	// Sanitize key to prevent path traversal
	key = sanitizeKey(key)
	fullPath := filepath.Join(p.basePath, key)

	// Create directories if needed
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create directories: %w", err)
	}

	// Create file
	file, err := os.Create(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Copy data and calculate hash
	hash := md5.New()
	tee := io.TeeReader(data, hash)
	size, err := io.Copy(file, tee)
	if err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	etag := fmt.Sprintf("%x", hash.Sum(nil))

	return &FileInfo{
		Key:         key,
		Size:        size,
		ContentType: contentType,
		PublicURL:   p.GetPublicURL(key),
		ETag:        etag,
	}, nil
}

// Download downloads a file from local storage
func (p *LocalProvider) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	key = sanitizeKey(key)
	fullPath := filepath.Join(p.basePath, key)

	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", key)
		}
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return file, nil
}

// Delete deletes a file from local storage
func (p *LocalProvider) Delete(ctx context.Context, key string) error {
	key = sanitizeKey(key)
	fullPath := filepath.Join(p.basePath, key)

	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			return nil // Already deleted, consider success
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// List lists files with a given prefix
func (p *LocalProvider) List(ctx context.Context, prefix string) ([]*FileInfo, error) {
	prefix = sanitizeKey(prefix)
	searchPath := filepath.Join(p.basePath, prefix)

	var files []*FileInfo

	err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Get relative path from base
		relPath, err := filepath.Rel(p.basePath, path)
		if err != nil {
			return err
		}

		files = append(files, &FileInfo{
			Key:       relPath,
			Size:      info.Size(),
			PublicURL: p.GetPublicURL(relPath),
		})

		return nil
	})

	if err != nil {
		if os.IsNotExist(err) {
			return []*FileInfo{}, nil // Empty list for non-existent prefix
		}
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	return files, nil
}

// Exists checks if a file exists
func (p *LocalProvider) Exists(ctx context.Context, key string) (bool, error) {
	key = sanitizeKey(key)
	fullPath := filepath.Join(p.basePath, key)

	_, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check file existence: %w", err)
	}

	return true, nil
}

// GetPublicURL returns the public URL for a file
func (p *LocalProvider) GetPublicURL(key string) string {
	if p.publicURLBase != "" {
		return fmt.Sprintf("%s/%s", strings.TrimSuffix(p.publicURLBase, "/"), key)
	}
	// Default to file:// URL for local development
	return fmt.Sprintf("file://%s", filepath.Join(p.basePath, key))
}

// GetPresignedURL returns a presigned URL (for local provider, just returns public URL)
func (p *LocalProvider) GetPresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	// Local provider doesn't support presigned URLs, just return public URL
	return p.GetPublicURL(key), nil
}

// Close closes the provider (no-op for local provider)
func (p *LocalProvider) Close() error {
	return nil
}

// sanitizeKey removes dangerous path components to prevent path traversal
func sanitizeKey(key string) string {
	// Remove leading slashes
	key = strings.TrimPrefix(key, "/")

	// Remove any ".." components
	parts := strings.Split(key, string(filepath.Separator))
	var cleaned []string
	for _, part := range parts {
		if part != ".." && part != "." && part != "" {
			cleaned = append(cleaned, part)
		}
	}

	return filepath.Join(cleaned...)
}
