package utils

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

// ExecuteAppleScript executes an AppleScript command and returns the output
func ExecuteAppleScript(ctx context.Context, script string) (string, error) {
	cmd := exec.CommandContext(ctx, "osascript", "-e", script)
	
	// Capture both stdout and stderr
	output, err := cmd.Output()
	if err != nil {
		// Try to get stderr for more detailed error info
		var stderr string
		if exitError, ok := err.(*exec.ExitError); ok {
			stderr = string(exitError.Stderr)
		}
		
		log.Printf("[AppleScript] Execution failed - Exit error: %v, Stderr: %s", err, stderr)
		
		if stderr != "" {
			return "", fmt.Errorf("applescript execution failed: %w - stderr: %s", err, stderr)
		}
		return "", fmt.Errorf("applescript execution failed: %w", err)
	}
	
	result := strings.TrimSpace(string(output))
	return result, nil
}