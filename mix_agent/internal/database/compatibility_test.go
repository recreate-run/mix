package database

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mix/internal/db"
)

// TestAbstractionVsOriginalCompatibility ensures the abstraction layer
// produces identical results to the original db.Connect function
func TestAbstractionVsOriginalCompatibility(t *testing.T) {
	ctx := context.Background()

	// Create two separate test directories
	originalDir := filepath.Join(os.TempDir(), "mix_original_test")
	abstractionDir := filepath.Join(os.TempDir(), "mix_abstraction_test")
	defer func() {
		_ = os.RemoveAll(originalDir)
		_ = os.RemoveAll(abstractionDir)
	}()

	// Test 1: Original system
	originalConn, err := db.Connect(ctx, originalDir)
	require.NoError(t, err, "Original db.Connect should work")
	defer func() {
		if err := originalConn.Close(); err != nil {
			t.Logf("failed to close original connection: %v", err)
		}
	}()

	// Test 2: Abstraction layer
	config := Config{
		Type: ProviderSQLite,
		SQLite: SQLiteConfig{
			DataDir:  abstractionDir,
			Filename: "mix.db",
		},
	}

	manager, err := NewManager(config)
	require.NoError(t, err)

	err = manager.Connect(ctx)
	require.NoError(t, err, "Abstraction Connect should work")
	defer func() {
		if err := manager.Close(); err != nil {
			t.Logf("failed to close manager: %v", err)
		}
	}()

	abstractionConn := manager.GetDB()

	// Compare: Both should have working connections
	assert.NoError(t, originalConn.PingContext(ctx))
	assert.NoError(t, abstractionConn.PingContext(ctx))

	// Compare: Both should have same database schema
	// Check that both have the sessions table with same structure
	originalSchema := getTableSchema(t, originalConn, "sessions")
	abstractionSchema := getTableSchema(t, abstractionConn, "sessions")
	assert.Equal(t, originalSchema, abstractionSchema, "Sessions table schema should be identical")

	// Check that both have the messages table with same structure
	originalSchema = getTableSchema(t, originalConn, "messages")
	abstractionSchema = getTableSchema(t, abstractionConn, "messages")
	assert.Equal(t, originalSchema, abstractionSchema, "Messages table schema should be identical")

	// Compare: Both should have same migration version
	originalVersion := getCurrentMigrationVersion(t, originalConn)
	abstractionVersion := getCurrentMigrationVersion(t, abstractionConn)
	assert.Equal(t, originalVersion, abstractionVersion, "Migration versions should be identical")

	// Compare: Both should support transactions
	originalTx, err := originalConn.BeginTx(ctx, nil)
	require.NoError(t, err)
	if err := originalTx.Rollback(); err != nil {
		t.Fatalf("failed to rollback original transaction: %v", err)
	}

	abstractionTx, err := abstractionConn.BeginTx(ctx, nil)
	require.NoError(t, err)
	if err := abstractionTx.Rollback(); err != nil {
		t.Fatalf("failed to rollback abstraction transaction: %v", err)
	}

	// Test SQLC compatibility: Both should work with db.New()
	originalQuerier := db.New(originalConn)
	abstractionQuerier := db.New(abstractionConn)

	assert.NotNil(t, originalQuerier)
	assert.NotNil(t, abstractionQuerier)
}

// Helper function to get table schema
func getTableSchema(t *testing.T, conn *sql.DB, tableName string) string {
	t.Helper()
	var schema string
	query := "SELECT sql FROM sqlite_master WHERE type='table' AND name=?"
	ctx := context.Background()
	err := conn.QueryRowContext(ctx, query, tableName).Scan(&schema)
	require.NoError(t, err, "Should be able to get table schema for %s", tableName)
	return schema
}

// Helper function to get current migration version
func getCurrentMigrationVersion(t *testing.T, conn *sql.DB) string {
	t.Helper()
	var version string
	query := "SELECT version_id FROM goose_db_version ORDER BY id DESC LIMIT 1"
	ctx := context.Background()
	err := conn.QueryRowContext(ctx, query).Scan(&version)
	require.NoError(t, err, "Should be able to get migration version")
	return version
}

// TestSQLCIntegration ensures our abstraction works with existing SQLC queries
func TestSQLCIntegration(t *testing.T) {
	testDir := filepath.Join(os.TempDir(), "mix_sqlc_test")
	defer func() {
		_ = os.RemoveAll(testDir)
	}()

	config := Config{
		Type: ProviderSQLite,
		SQLite: SQLiteConfig{
			DataDir:  testDir,
			Filename: "mix.db",
		},
	}

	manager, err := NewManager(config)
	require.NoError(t, err)

	ctx := context.Background()
	err = manager.Connect(ctx)
	require.NoError(t, err)
	defer func() {
		if err := manager.Close(); err != nil {
			t.Logf("failed to close manager: %v", err)
		}
	}()

	// Test that SQLC queries work with our abstraction
	querier := db.New(manager.GetDB())
	assert.NotNil(t, querier)

	// Test a simple query (should not fail)
	sessions, err := querier.ListSessionsMetadata(ctx)
	require.NoError(t, err, "SQLC queries should work with abstraction")
	assert.NotNil(t, sessions, "Should get valid result from SQLC query")
}

// TestDefaultBehaviorConsistency ensures defaults match the original system
func TestDefaultBehaviorConsistency(t *testing.T) {
	testDir := filepath.Join(os.TempDir(), "mix_defaults_test")
	defer func() {
		_ = os.RemoveAll(testDir)
	}()

	// Test that our default filename matches the original
	config := Config{
		Type: ProviderSQLite,
		SQLite: SQLiteConfig{
			DataDir: testDir,
			// No filename specified - should default to "mix.db"
		},
	}

	manager, err := NewManager(config)
	require.NoError(t, err)

	ctx := context.Background()
	err = manager.Connect(ctx)
	require.NoError(t, err)
	defer func() {
		if err := manager.Close(); err != nil {
			t.Logf("failed to close manager: %v", err)
		}
	}()

	// Verify the default filename was used (same as original system)
	expectedPath := filepath.Join(testDir, "mix.db")
	_, err = os.Stat(expectedPath)
	assert.NoError(t, err, "Should use mix.db as default filename like original system")
}
