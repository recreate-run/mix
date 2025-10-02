package storage

import (
	"context"
	"io"
	"time"
)

// Provider defines the storage interface for all storage backends
type Provider interface {
	// Core operations
	Upload(ctx context.Context, key string, data io.Reader, contentType string) (*FileInfo, error)
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	List(ctx context.Context, prefix string) ([]*FileInfo, error)
	Exists(ctx context.Context, key string) (bool, error)

	// URL generation
	GetPublicURL(key string) string
	GetPresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error)

	// Lifecycle
	Close() error
}

// FileInfo represents metadata about a stored file
type FileInfo struct {
	Key         string
	Size        int64
	ContentType string
	PublicURL   string
	ETag        string
	Metadata    map[string]string
}

// Config holds storage configuration
type Config struct {
	Type string `json:"type"` // "s3" or "local"

	// S3-compatible configuration (works with ALL S3-compatible providers)
	Endpoint       string `json:"endpoint"`
	Bucket         string `json:"bucket"`
	AccessKey      string `json:"access_key"`
	SecretKey      string `json:"secret_key"`
	Region         string `json:"region"`
	UseSSL         bool   `json:"use_ssl"`
	ForcePathStyle bool   `json:"force_path_style"`
	PublicURLBase  string `json:"public_url_base"`
}

// Provider type constants
const (
	ProviderTypeLocal    = "local"
	ProviderTypeSupabase = "supabase"
)
