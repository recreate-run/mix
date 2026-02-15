package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"

	"mix/internal/constants"
	"mix/internal/db"
	"mix/internal/logging"
)

// PostgresProvider implements Provider interface for PostgreSQL databases
type PostgresProvider struct {
	config PostgresConfig
	db     *sql.DB
}

// NewPostgresProvider creates a new PostgreSQL provider
func NewPostgresProvider(config PostgresConfig) *PostgresProvider {
	return &PostgresProvider{
		config: config,
	}
}

// Connect establishes a connection to the PostgreSQL database
func (p *PostgresProvider) Connect(ctx context.Context) error {
	if p.config.Host == "" {
		return fmt.Errorf("postgres host is required")
	}
	if p.config.Database == "" {
		return fmt.Errorf("postgres database is required")
	}
	if p.config.User == "" {
		return fmt.Errorf("postgres user is required")
	}

	// Set defaults
	port := p.config.Port
	if port == 0 {
		port = 5432
	}
	sslMode := p.config.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}

	// Construct connection string
	connectionString := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		p.config.Host,
		port,
		p.config.User,
		p.config.Password,
		p.config.Database,
		sslMode,
	)

	// Connect to PostgreSQL
	logging.Info("Connecting to PostgreSQL database", "host", p.config.Host, "database", p.config.Database)
	sqlDB, err := sql.Open("postgres", connectionString)
	if err != nil {
		return fmt.Errorf("failed to open PostgreSQL connection: %w", err)
	}

	// Store the connection
	p.db = sqlDB

	// Configure connection pool
	p.db.SetMaxOpenConns(25)
	p.db.SetMaxIdleConns(5)
	p.db.SetConnMaxLifetime(5 * time.Minute)

	// Verify connection with timeout
	pingCtx, cancel := context.WithTimeout(ctx, constants.DatabasePingTimeout)
	defer cancel()
	if err = p.db.PingContext(pingCtx); err != nil {
		_ = p.db.Close() // Ignore close error in cleanup path
		p.db = nil
		return fmt.Errorf("failed to ping PostgreSQL database: %w", err)
	}

	logging.Debug("Successfully connected to PostgreSQL database", "host", p.config.Host)
	return nil
}

// Close closes the database connection
func (p *PostgresProvider) Close() error {
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}

// GetDB returns the underlying sql.DB instance
func (p *PostgresProvider) GetDB() *sql.DB {
	return p.db
}

// Ping checks if the database connection is alive
func (p *PostgresProvider) Ping(ctx context.Context) error {
	if p.db == nil {
		return fmt.Errorf("database not connected")
	}
	return p.db.PingContext(ctx)
}

// BeginTx starts a transaction
func (p *PostgresProvider) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	if p.db == nil {
		return nil, fmt.Errorf("database not connected")
	}
	return p.db.BeginTx(ctx, opts)
}

// Type returns the provider type
func (p *PostgresProvider) Type() ProviderType {
	return ProviderPostgres
}

// RunMigrations executes database migrations for PostgreSQL
func (p *PostgresProvider) RunMigrations(ctx context.Context) error {
	if p.db == nil {
		return fmt.Errorf("database not connected")
	}

	// Set embedded filesystem for migrations
	goose.SetBaseFS(db.FS)

	// Set PostgreSQL dialect
	if err := goose.SetDialect("postgres"); err != nil {
		logging.Error("Failed to set dialect", "error", err)
		return fmt.Errorf("failed to set dialect: %w", err)
	}

	// Run migrations with timeout
	migrationCtx, cancel := context.WithTimeout(ctx, constants.DatabaseMigrationTimeout)
	defer cancel()

	if err := goose.UpContext(migrationCtx, p.db, "migrations"); err != nil {
		logging.Error("Failed to apply migrations to PostgreSQL", "error", err)
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	logging.Debug("Successfully applied migrations to PostgreSQL database")
	return nil
}
