// Package logging provides simplified logging for embedded binary use.
package logging

import (
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"runtime/debug"
	"time"
)

func init() {
	// Create a custom handler that removes timestamps
	opts := &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Remove the time attribute
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	}
	handler := slog.NewTextHandler(os.Stdout, opts)
	slog.SetDefault(slog.New(handler))
}

func getCaller() string {
	var caller string
	if _, file, line, ok := runtime.Caller(2); ok {
		// caller = fmt.Sprintf("%s:%d", filepath.Base(file), line)
		caller = fmt.Sprintf("%s:%d", file, line)
	} else {
		caller = "unknown"
	}
	return caller
}
func Info(msg string, args ...any) {
	slog.Info(msg, args...)
}

func Debug(msg string, args ...any) {
	slog.Debug(msg, args...)
}

func Warn(msg string, args ...any) {
	slog.Warn(msg, args...)
}

func Error(msg string, args ...any) {
	slog.Error(msg, args...)
}

// Simplified logging - removed persist functions for embedded binary

// RecoverPanic handles panics gracefully for embedded binary.
func RecoverPanic(name string, cleanup func()) {
	if r := recover(); r != nil {
		// Log the panic
		Error(fmt.Sprintf("Panic in %s: %v", name, r))

		// Create a simple panic log file
		timestamp := time.Now().Format("20060102-150405")
		filename := fmt.Sprintf("mix-panic-%s-%s.log", name, timestamp)

		file, err := os.Create(filename)
		if err != nil {
			Error(fmt.Sprintf("Failed to create panic log: %v", err))
		} else {
			defer file.Close()

			// Write panic information and stack trace
			fmt.Fprintf(file, "Panic in %s: %v\n\n", name, r)
			fmt.Fprintf(file, "Time: %s\n\n", time.Now().Format(time.RFC3339))
			fmt.Fprintf(file, "Stack Trace:\n%s\n", debug.Stack())

			Info(fmt.Sprintf("Panic details written to %s", filename))
		}

		// Execute cleanup function if provided
		if cleanup != nil {
			cleanup()
		}
	}
}

// Removed complex session message logging for embedded binary
