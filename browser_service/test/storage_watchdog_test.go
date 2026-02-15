//go:build e2e

package test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sarathmenon/browser-service/pkg/client"
	"github.com/sarathmenon/browser-service/pkg/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStorageWatchdogAutoSavesCookies(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	// Create temp storage state file
	tmpDir := t.TempDir()
	storageStatePath := filepath.Join(tmpDir, "storage_state.json")

	ctx := context.Background()

	// Start test server with storage state path
	_, wsURL, cleanup := startTestServerWithStorageState(t, ctx, storageStatePath)
	defer cleanup()

	// Connect client
	c, err := client.New(wsURL)
	require.NoError(t, err)
	defer c.Close()

	// Set cookies via Browser.setCookies
	_, err = c.SetCookies(ctx, []protocol.Cookie{
		{
			Name:   "session",
			Value:  "abc123",
			Domain: "127.0.0.1",
			Path:   "/",
		},
		{
			Name:   "user_id",
			Value:  "12345",
			Domain: "127.0.0.1",
			Path:   "/",
		},
	})
	require.NoError(t, err)

	// Wait for auto-save (runs every 30s, but we need to wait for at least one cycle)
	// To speed up testing, we'll wait 35s to ensure the watchdog has run
	t.Log("Waiting 35 seconds for auto-save to trigger...")
	time.Sleep(35 * time.Second)

	// Check if storage state file was created and contains cookies
	require.FileExists(t, storageStatePath, "Storage state file should be created")

	// Read and parse storage state file
	data, err := os.ReadFile(storageStatePath)
	require.NoError(t, err)

	var state protocol.StorageState
	err = json.Unmarshal(data, &state)
	require.NoError(t, err)

	// Verify cookies were saved
	assert.GreaterOrEqual(t, len(state.Cookies), 2, "Should have at least 2 cookies")

	// Find our specific cookies
	foundSession := false
	foundUserID := false
	for _, cookie := range state.Cookies {
		if cookie.Name == "session" && cookie.Value == "abc123" {
			foundSession = true
		}
		if cookie.Name == "user_id" && cookie.Value == "12345" {
			foundUserID = true
		}
	}

	assert.True(t, foundSession, "Session cookie should be saved")
	assert.True(t, foundUserID, "User ID cookie should be saved")

	t.Log("Auto-save test passed: cookies saved to", storageStatePath)
}

func TestStorageWatchdogAutoLoadsOnRestart(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	// Create temp storage state file
	tmpDir := t.TempDir()
	storageStatePath := filepath.Join(tmpDir, "storage_state.json")

	// Create a pre-existing storage state file
	initialState := protocol.StorageState{
		Cookies: []protocol.Cookie{
			{
				Name:   "preloaded_session",
				Value:  "xyz789",
				Domain: "127.0.0.1",
				Path:   "/",
			},
		},
		Origins: []protocol.OriginState{},
	}

	data, err := json.MarshalIndent(initialState, "", "  ")
	require.NoError(t, err)

	err = os.WriteFile(storageStatePath, data, 0644)
	require.NoError(t, err)

	t.Log("Created pre-existing storage state file with preloaded_session cookie")

	ctx := context.Background()

	// Start test server with storage state path
	// The watchdog should auto-load the state on connect
	_, wsURL, cleanup := startTestServerWithStorageState(t, ctx, storageStatePath)
	defer cleanup()

	// Connect client
	c, err := client.New(wsURL)
	require.NoError(t, err)
	defer c.Close()

	// Give the watchdog a moment to load the state
	time.Sleep(2 * time.Second)

	// Verify the cookie was loaded by checking Browser.getCookies
	result, err := c.GetCookies(ctx)
	require.NoError(t, err)

	// Find the preloaded cookie
	found := false
	for _, cookie := range result.Cookies {
		if cookie.Name == "preloaded_session" && cookie.Value == "xyz789" {
			found = true
			break
		}
	}

	assert.True(t, found, "Preloaded cookie should be loaded on connect")

	t.Log("Auto-load test passed: preloaded cookie found in browser")
}

func TestStorageWatchdogMergesStates(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	// Create temp storage state file
	tmpDir := t.TempDir()
	storageStatePath := filepath.Join(tmpDir, "storage_state.json")

	// Create initial storage state file with one cookie
	initialState := protocol.StorageState{
		Cookies: []protocol.Cookie{
			{
				Name:   "old_cookie",
				Value:  "old_value",
				Domain: "127.0.0.1",
				Path:   "/",
			},
		},
		Origins: []protocol.OriginState{},
	}

	data, err := json.MarshalIndent(initialState, "", "  ")
	require.NoError(t, err)

	err = os.WriteFile(storageStatePath, data, 0644)
	require.NoError(t, err)

	t.Log("Created initial storage state with old_cookie")

	ctx := context.Background()

	// Start test server with storage state path
	_, wsURL, cleanup := startTestServerWithStorageState(t, ctx, storageStatePath)
	defer cleanup()

	// Connect client
	c, err := client.New(wsURL)
	require.NoError(t, err)
	defer c.Close()

	// Wait for auto-load
	time.Sleep(2 * time.Second)

	// Set new cookies
	_, err = c.SetCookies(ctx, []protocol.Cookie{
		{
			Name:   "new_cookie",
			Value:  "new_value",
			Domain: "127.0.0.1",
			Path:   "/",
		},
		{
			Name:   "old_cookie", // Same name - should overwrite
			Value:  "updated_value",
			Domain: "127.0.0.1",
			Path:   "/",
		},
	})
	require.NoError(t, err)

	// Wait for auto-save to trigger
	t.Log("Waiting 35 seconds for auto-save to trigger...")
	time.Sleep(35 * time.Second)

	// Read the saved storage state
	data, err = os.ReadFile(storageStatePath)
	require.NoError(t, err)

	var mergedState protocol.StorageState
	err = json.Unmarshal(data, &mergedState)
	require.NoError(t, err)

	// Verify merged state
	assert.GreaterOrEqual(t, len(mergedState.Cookies), 2, "Should have at least 2 cookies")

	// Check that old_cookie was updated (new value wins)
	foundOldCookie := false
	foundNewCookie := false
	for _, cookie := range mergedState.Cookies {
		if cookie.Name == "old_cookie" {
			foundOldCookie = true
			assert.Equal(t, "updated_value", cookie.Value, "Old cookie should be updated with new value")
		}
		if cookie.Name == "new_cookie" {
			foundNewCookie = true
			assert.Equal(t, "new_value", cookie.Value)
		}
	}

	assert.True(t, foundOldCookie, "Old cookie should exist in merged state")
	assert.True(t, foundNewCookie, "New cookie should exist in merged state")

	t.Log("Merge test passed: old cookie updated, new cookie added")
}
