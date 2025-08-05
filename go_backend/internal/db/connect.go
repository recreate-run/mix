package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"

	"mix/internal/config"
	"mix/internal/logging"

	"github.com/pressly/goose/v3"
)

const (
	// Database operation timeouts
	DBConnectionTimeout = 30 * time.Second
	DBPingTimeout       = 10 * time.Second
	DBPragmaTimeout     = 10 * time.Second
	DBMigrationTimeout  = 5 * time.Minute
)

func Connect(ctx context.Context) (*sql.DB, error) {
	dataDir := config.Get().Data.Directory
	if dataDir == "" {
		return nil, fmt.Errorf("data.dir is not set")
	}
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}
	dbPath := filepath.Join(dataDir, "mix.db")
	// Open the SQLite database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Verify connection with timeout
	pingCtx, cancel := context.WithTimeout(ctx, DBPingTimeout)
	defer cancel()
	if err = db.PingContext(pingCtx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Set pragmas for better performance
	pragmas := []string{
		"PRAGMA foreign_keys = ON;",
		"PRAGMA journal_mode = WAL;",
		"PRAGMA page_size = 4096;",
		"PRAGMA cache_size = -8000;",
		"PRAGMA synchronous = NORMAL;",
	}

	for _, pragma := range pragmas {
		pragmaCtx, cancel := context.WithTimeout(ctx, DBPragmaTimeout)
		if _, err = db.ExecContext(pragmaCtx, pragma); err != nil {
			logging.Error("Failed to set pragma", pragma, err)
		} else {
			logging.Debug("Set pragma", "pragma", pragma)
		}
		cancel()
	}

	goose.SetBaseFS(FS)

	if err := goose.SetDialect("sqlite3"); err != nil {
		logging.Error("Failed to set dialect", "error", err)
		return nil, fmt.Errorf("failed to set dialect: %w", err)
	}

	migrationCtx, cancel := context.WithTimeout(ctx, DBMigrationTimeout)
	defer cancel()
	if err := goose.UpContext(migrationCtx, db, "migrations"); err != nil {
		logging.Error("Failed to apply migrations", "error", err)
		return nil, fmt.Errorf("failed to apply migrations: %w", err)
	}
	return db, nil
}

// SetupTestDatabase applies migrations to a test database connection
func SetupTestDatabase(ctx context.Context, db *sql.DB) error {
	goose.SetBaseFS(FS)

	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}

	migrationCtx, cancel := context.WithTimeout(ctx, DBMigrationTimeout)
	defer cancel()
	if err := goose.UpContext(migrationCtx, db, "migrations"); err != nil {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	return nil
}
