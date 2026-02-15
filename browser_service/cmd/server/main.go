package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/sarathmenon/browser-service/internal/constants"
	"github.com/sarathmenon/browser-service/internal/server"
)

func main() {
	port := flag.String("port", constants.DefaultPort, "WebSocket server port")
	headless := flag.Bool("headless", false, "Run browser in headless mode")
	stealth := flag.Bool("stealth", false, "Enable stealth mode (disable automation detection)")
	windowWidth := flag.Int("window-width", 1920, "Browser window width")
	windowHeight := flag.Int("window-height", 1080, "Browser window height")
	flag.Parse()

	cfg := server.Config{
		Port:         *port,
		Headless:     *headless,
		Stealth:      *stealth,
		WindowWidth:  *windowWidth,
		WindowHeight: *windowHeight,
	}

	// Create cancellable root context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())

	srv, err := server.New(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Set up signal handling for graceful shutdown during hot reload
	// This ensures the WebSocket server releases the port before the new process starts
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal, initiating graceful shutdown")
		cancel() // Cancel the context to trigger server shutdown
	}()

	log.Printf("Starting browser service on port %s (headless=%v)", *port, *headless)
	if err := srv.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
