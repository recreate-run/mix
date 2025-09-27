package database

import (
	"context"
	"database/sql"
)

// Provider defines the interface for database providers
type Provider interface {
	// Connection management
	Connect(ctx context.Context) error
	Close() error
	GetDB() *sql.DB
	Ping(ctx context.Context) error

	// Transaction support
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)

	// Provider identification
	Type() ProviderType

	// Migration support
	RunMigrations(ctx context.Context) error
}

// ProviderType identifies the type of database provider
type ProviderType string

const (
	ProviderSQLite ProviderType = "sqlite"
	ProviderTurso  ProviderType = "turso"
)

// Config holds database configuration
type Config struct {
	Type   ProviderType `json:"type"`
	SQLite SQLiteConfig `json:"sqlite"`
	Turso  TursoConfig  `json:"turso"`
}

// SQLiteConfig holds SQLite-specific configuration
type SQLiteConfig struct {
	DataDir  string `json:"dataDir"`
	Filename string `json:"filename"`
}

// TursoConfig holds Turso-specific configuration
type TursoConfig struct {
	URL       string `json:"url"`       // libsql://[databaseId]-[organizationName].turso.io
	AuthToken string `json:"authToken"` // Turso auth token
}