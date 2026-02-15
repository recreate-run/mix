package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/sarathmenon/browser-service/internal/constants"
	"github.com/sarathmenon/browser-service/internal/server"
)

func main() {
	port := flag.String("port", constants.DefaultPort, "WebSocket server port")
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
