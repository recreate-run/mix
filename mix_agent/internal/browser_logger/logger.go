package browser_logger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mix/internal/session"
)

const BrowserLogFilename = "browser.log"

// BrowserLogMessage represents a browser action log event from the Electron app
type BrowserLogMessage struct {
	Type      string `json:"type"`      // Always "browser_log"
	SessionID string `json:"sessionId"` // Session identifier
	Timestamp int64  `json:"timestamp"` // Unix timestamp in milliseconds
	TabID     string `json:"tabId"`     // Tab identifier
	LogType   string `json:"logType"`   // "navigation" | "network_request"
	Data      any    `json:"data"`      // Type-specific data
}

// AppendLog writes a browser log message to the session's browser.log file
// Logs are written in structured key=value format (one log per line)
// Format: HH:MM:SS tabId=X |  logType=Y key1=val1 key2=val2
// This is a fire-and-forget operation - errors are logged but don't fail the operation
func AppendLog(logMessage BrowserLogMessage, config session.Config) error {
	if logMessage.SessionID == "" {
		return fmt.Errorf("sessionID is required")
	}

	// Get session storage directory
	sessionDir := session.GetSessionStoragePath(logMessage.SessionID, config)
	logFilePath := filepath.Join(sessionDir, BrowserLogFilename)

	// Open file in append mode (create if doesn't exist)
	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o640)
	if err != nil {
		return fmt.Errorf("failed to open browser log file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	// Format log line
	logLine := formatBrowserLog(logMessage)

	// Write to file
	if _, err := file.WriteString(logLine); err != nil {
		return fmt.Errorf("failed to write to browser log file: %w", err)
	}

	return nil
}

// formatBrowserLog formats a browser log message in structured key=value format
// Format: HH:MM:SS tabId=X |  logType=Y key1=val1 key2=val2
func formatBrowserLog(msg BrowserLogMessage) string {
	var b strings.Builder

	// Convert timestamp from milliseconds to HH:MM:SS
	t := time.UnixMilli(msg.Timestamp)
	b.WriteString(t.Format("15:04:05"))
	b.WriteString(" ")

	// Add tabId
	b.WriteString("tabId=")
	b.WriteString(msg.TabID)
	b.WriteString(" |  ")

	// Add logType
	b.WriteString("logType=")
	b.WriteString(msg.LogType)

	// Flatten and add data fields
	if msg.Data != nil {
		dataMap, ok := msg.Data.(map[string]interface{})
		if !ok {
			// Try to convert via JSON if not already a map
			jsonBytes, err := json.Marshal(msg.Data)
			if err == nil {
				_ = json.Unmarshal(jsonBytes, &dataMap)
			}
		}

		// Add each data field as key=value
		for key, value := range dataMap {
			b.WriteString(" ")
			b.WriteString(key)
			b.WriteString("=")
			b.WriteString(formatValue(value))
		}
	}

	b.WriteString("\n")
	return b.String()
}

// formatValue formats a value for logging (quotes strings with spaces, handles other types)
func formatValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		// Quote if contains spaces
		if strings.Contains(val, " ") {
			return fmt.Sprintf("%q", val)
		}
		return val
	case nil:
		return "<nil>"
	default:
		return fmt.Sprintf("%v", val)
	}
}
