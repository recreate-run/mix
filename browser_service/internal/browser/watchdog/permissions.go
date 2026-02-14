package watchdog

import (
	"context"
	"fmt"
	"log"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// PermissionsWatchdog handles automatic permission grants
type PermissionsWatchdog struct {
	browser *rod.Browser
}

// NewPermissionsWatchdog creates a new permissions watchdog
func NewPermissionsWatchdog(browser *rod.Browser) *PermissionsWatchdog {
	return &PermissionsWatchdog{
		browser: browser,
	}
}

// Start grants default permissions to the browser
func (w *PermissionsWatchdog) Start(ctx context.Context) error {
	// Grant clipboard and notifications permissions
	permissions := []proto.BrowserPermissionType{
		proto.BrowserPermissionTypeClipboardReadWrite,
		proto.BrowserPermissionTypeClipboardSanitizedWrite,
		proto.BrowserPermissionTypeNotifications,
	}

	err := proto.BrowserGrantPermissions{
		Permissions: permissions,
	}.Call(w.browser)

	if err != nil {
		// Log error but don't fail - permissions are non-critical
		log.Printf("Failed to grant permissions: %v", err)
		return nil
	}

	log.Printf("Successfully granted permissions: %v", permissions)
	return nil
}

// GrantPermissions grants specific permissions
func (w *PermissionsWatchdog) GrantPermissions(ctx context.Context, permissions []proto.BrowserPermissionType) error {
	err := proto.BrowserGrantPermissions{
		Permissions: permissions,
	}.Call(w.browser)

	if err != nil {
		return fmt.Errorf("grant permissions: %w", err)
	}

	return nil
}
