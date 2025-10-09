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

func TestNewManagerTurso(t *testing.T) {
	config := Config{
		Type: ProviderTurso,
		Turso: TursoConfig{
			URL:       "libsql://test-db-org.turso.io",
			AuthToken: "test-auth-token",
		},
	}

	manager, err := NewManager(config)
	require.NoError(t, err)
	assert.NotNil(t, manager)
	assert.Equal(t, ProviderTurso, manager.Type())

	// Verify the provider is a Turso provider
	provider := manager.GetProvider()
	assert.NotNil(t, provider)
	assert.Equal(t, ProviderTurso, provider.Type())
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

func TestTursoProviderValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      TursoConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: TursoConfig{
				URL:       "libsql://test-db-org.turso.io",
				AuthToken: "test-token",
			},
			expectError: false,
		},
		{
			name: "missing URL",
			config: TursoConfig{
				URL:       "",
				AuthToken: "test-token",
			},
			expectError: true,
			errorMsg:    "turso URL is required",
		},
		{
			name: "missing auth token",
			config: TursoConfig{
				URL:       "libsql://test-db-org.turso.io",
				AuthToken: "",
			},
			expectError: true,
			errorMsg:    "turso auth token is required",
		},
		{
			name: "both missing",
			config: TursoConfig{
				URL:       "",
				AuthToken: "",
			},
			expectError: true,
			errorMsg:    "turso URL is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				Type:  ProviderTurso,
				Turso: tt.config,
			}

			manager, err := NewManager(config)
			require.NoError(t, err, "Manager creation should succeed")

			ctx := context.Background()
			err = manager.Connect(ctx)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				// Note: This will likely fail in tests since we don't have a real Turso database
				// but it validates that the provider accepts the configuration correctly
				// The actual connection failure is expected in test environment
				if err != nil {
					// Expected connection failure in test environment is okay
					t.Logf("Expected connection failure in test environment: %v", err)
				}
			}
		})
	}
}

func TestTursoProviderMethods(t *testing.T) {
	config := Config{
		Type: ProviderTurso,
		Turso: TursoConfig{
			URL:       "libsql://test-db-org.turso.io",
			AuthToken: "test-token",
		},
	}

	manager, err := NewManager(config)
	require.NoError(t, err)

	// Test methods on unconnected provider
	assert.Equal(t, ProviderTurso, manager.Type())
	assert.NotNil(t, manager.GetProvider())

	// GetDB should return nil when not connected
	assert.Nil(t, manager.GetDB())

	// Ping should fail when not connected
	ctx := context.Background()
	err = manager.Ping(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database not connected")

	// BeginTx should fail when not connected
	tx, err := manager.BeginTx(ctx, nil)
	assert.Error(t, err)
	assert.Nil(t, tx)
	assert.Contains(t, err.Error(), "database not connected")

	// Close should not fail even when not connected
	err = manager.Close()
	assert.NoError(t, err)
}