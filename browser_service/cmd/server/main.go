package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/sarathmenon/browser-service/internal/constants"
	"github.com/sarathmenon/browser-service/internal/server"
)

// extractPortFromURL extracts the port from a URL string
// Returns error if URL is invalid or has no port
func extractPortFromURL(urlStr string) (string, error) {
	if urlStr == "" {
		return "", fmt.Errorf("URL is empty")
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse URL: %w", err)
	}

	port := parsedURL.Port()
	if port == "" {
		return "", fmt.Errorf("URL has no port: %s", urlStr)
	}

	return port, nil
}

func main() {
	// Load .env file from project root (ignore error if file doesn't exist)
	// Try current directory first, then parent directory
	if err := godotenv.Load(); err != nil {
		_ = godotenv.Load("../.env")
	}

	log.SetFlags(0)

	// Get BROWSER_SERVICE_URL from environment variable (required, no default)
	browserServiceURL := os.Getenv("BROWSER_SERVICE_URL")

	// Extract port from URL
	portFromEnv, err := extractPortFromURL(browserServiceURL)
	if err != nil {
		log.Fatalf("Invalid BROWSER_SERVICE_URL: %v", err)
	}

	port := flag.String("port", portFromEnv, "WebSocket server port")
	headless := flag.Bool("headless", false, "Run browser in headless mode")
	stealth := flag.Bool("stealth", false, "Enable stealth mode (disable automation detection)")
	windowWidth := flag.Int("window-width", 1280, "Browser window width")
	windowHeight := flag.Int("window-height", 720, "Browser window height")
	storageStatePath := flag.String("storage-state-path", "", "Path to save/load storage state (empty to disable)")

	// Extension flags
	enableExtensions := flag.Bool("enable-extensions", false, "Enable browser extensions (uBlock Origin, cookie handlers, ClearURLs)")
	extensionCacheDir := flag.String("extension-cache-dir", "", "Extension cache directory (default: ~/.cache/mix-browser/extensions)")
	cookieWhitelist := flag.String("cookie-whitelist", "", "Comma-separated list of domains allowed to set cookies (e.g., example.com,test.com)")
	uBlockEnabled := flag.Bool("ublock-enabled", true, "Enable uBlock Origin extension")
	cookieConsentEnabled := flag.Bool("cookie-consent-enabled", true, "Enable 'I don't care about cookies' extension")
	clearURLsEnabled := flag.Bool("clearurls-enabled", true, "Enable ClearURLs extension")
	allowModals := flag.Bool("allow-modals", false, "Disable modal blocking (modals blocked by default)")

	flag.Parse()

	// Parse cookie whitelist
	var cookieWhitelistDomains []string
	if *cookieWhitelist != "" {
		cookieWhitelistDomains = strings.Split(*cookieWhitelist, ",")
		// Trim whitespace from each domain
		for i, domain := range cookieWhitelistDomains {
			cookieWhitelistDomains[i] = strings.TrimSpace(domain)
		}
	}

	cfg := server.Config{
		Port:                   *port,
		Headless:               *headless,
		Stealth:                *stealth,
		WindowWidth:            *windowWidth,
		WindowHeight:           *windowHeight,
		StorageStatePath:       *storageStatePath,
		EnableExtensions:       *enableExtensions,
		ExtensionCacheDir:      *extensionCacheDir,
		CookieWhitelistDomains: cookieWhitelistDomains,
		UBlockEnabled:          *uBlockEnabled,
		CookieConsentEnabled:   *cookieConsentEnabled,
		ClearURLsEnabled:       *clearURLsEnabled,
		BlockModals:            !*allowModals, // Modal blocking enabled by default
	}

	// Create root context
	ctx := context.Background()

	srv, err := server.New(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down gracefully...")

		// Create shutdown context with timeout
		shutdownCtx, cancel := context.WithTimeout(context.Background(), constants.DefaultShutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("Error during shutdown: %v", err)
		}
		os.Exit(0)
	}()

	log.Printf("Starting browser service on port %s (headless=%v)", *port, *headless)
	if err := srv.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
