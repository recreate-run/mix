package database

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	config := Config{
		Type: ProviderSQLite,
		SQLite: SQLiteConfig{
			DataDir:  ".test",
			Filename: "test.db",
		},
	}

	manager, err := NewManager(config)
	require.NoError(t, err)
	assert.NotNil(t, manager)
	assert.Equal(t, ProviderSQLite, manager.Type())
}

func TestNewManagerUnsupportedType(t *testing.T) {
	config := Config{
		Type: ProviderType("unsupported"),
	}

	manager, err := NewManager(config)
	assert.Error(t, err)
	assert.Nil(t, manager)
	assert.Contains(t, err.Error(), "unsupported database provider type")
}

func TestSQLiteManagerLifecycle(t *testing.T) {
	// Create a temporary directory for testing
	testDir := filepath.Join(os.TempDir(), "mix_db_test")
	defer os.RemoveAll(testDir)

	config := Config{
		Type: ProviderSQLite,
		SQLite: SQLiteConfig{
			DataDir:  testDir,
			Filename: "test.db",
		},
	}

	manager, err := NewManager(config)
	require.NoError(t, err)

	ctx := context.Background()

	// Test connection
	err = manager.Connect(ctx)
	require.NoError(t, err)

	// Test that we get a valid database instance
	db := manager.GetDB()
	assert.NotNil(t, db)

	// Test ping
	err = manager.Ping(ctx)
	assert.NoError(t, err)

	// Test that the database file was created
	dbPath := filepath.Join(testDir, "test.db")
	_, err = os.Stat(dbPath)
	assert.NoError(t, err, "Database file should be created")

	// Test transaction
	tx, err := manager.BeginTx(ctx, nil)
	require.NoError(t, err)
	err = tx.Rollback()
	assert.NoError(t, err)

	// Test provider access
	provider := manager.GetProvider()
	assert.NotNil(t, provider)
	assert.Equal(t, ProviderSQLite, provider.Type())

	// Test close
	err = manager.Close()
	assert.NoError(t, err)
}

func TestSQLiteProviderDefaults(t *testing.T) {
	testDir := filepath.Join(os.TempDir(), "mix_db_test_defaults")
	defer os.RemoveAll(testDir)

	config := Config{
		Type: ProviderSQLite,
		SQLite: SQLiteConfig{
			DataDir: testDir,
			// Filename not specified - should default to "mix.db"
		},
	}

	manager, err := NewManager(config)
	require.NoError(t, err)

	ctx := context.Background()
	err = manager.Connect(ctx)
	require.NoError(t, err)
	defer manager.Close()

	// Test that default filename was used
	dbPath := filepath.Join(testDir, "mix.db")
	_, err = os.Stat(dbPath)
	assert.NoError(t, err, "Default database filename should be mix.db")
}

func TestSQLiteProviderMigrations(t *testing.T) {
	testDir := filepath.Join(os.TempDir(), "mix_db_test_migrations")
	defer os.RemoveAll(testDir)

	config := Config{
		Type: ProviderSQLite,
		SQLite: SQLiteConfig{
			DataDir:  testDir,
			Filename: "test.db",
		},
	}

	manager, err := NewManager(config)
	require.NoError(t, err)

	ctx := context.Background()
	err = manager.Connect(ctx)
	require.NoError(t, err)
	defer manager.Close()

	// Test that migrations created expected tables
	db := manager.GetDB()

	// Check that sessions table exists (from migrations)
	var tableName string
	err = db.QueryRowContext(ctx, "SELECT name FROM sqlite_master WHERE type='table' AND name='sessions'").Scan(&tableName)
	assert.NoError(t, err)
	assert.Equal(t, "sessions", tableName)

	// Check that messages table exists (from migrations)
	err = db.QueryRowContext(ctx, "SELECT name FROM sqlite_master WHERE type='table' AND name='messages'").Scan(&tableName)
	assert.NoError(t, err)
	assert.Equal(t, "messages", tableName)
}

func TestSQLiteProviderConnectErrors(t *testing.T) {
	// Test with empty data directory
	config := Config{
		Type: ProviderSQLite,
		SQLite: SQLiteConfig{
			DataDir:  "",
			Filename: "test.db",
		},
	}

	manager, err := NewManager(config)
	require.NoError(t, err)

	ctx := context.Background()
	err = manager.Connect(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "data.dir is not set")
}

func TestManagerMethods(t *testing.T) {
	testDir := filepath.Join(os.TempDir(), "mix_db_test_methods")
	defer os.RemoveAll(testDir)

	config := Config{
		Type: ProviderSQLite,
		SQLite: SQLiteConfig{
			DataDir:  testDir,
			Filename: "test.db",
		},
	}

	manager, err := NewManager(config)
	require.NoError(t, err)

	ctx := context.Background()
	err = manager.Connect(ctx)
	require.NoError(t, err)
	defer manager.Close()

	// Test all manager methods work
	assert.Equal(t, ProviderSQLite, manager.Type())
	assert.NotNil(t, manager.GetDB())
	assert.NotNil(t, manager.GetProvider())
	assert.NoError(t, manager.Ping(ctx))
}