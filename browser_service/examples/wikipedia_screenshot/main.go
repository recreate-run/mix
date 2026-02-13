package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/sarathmenon/browser-service/pkg/client"
	"github.com/sarathmenon/browser-service/pkg/protocol"
)

func main() {
	// Connect to the browser service
	// Make sure the server is running: task dev
	wsURL := "ws://localhost:8081/ws"

	log.Println("Connecting to browser service at", wsURL)
	c, err := client.New(wsURL)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer c.Close()

	// Navigate to Wikipedia page for cats
	log.Println("Navigating to Wikipedia Cats page...")
	navCtx, navCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer navCancel()

	navResult, err := c.Navigate(navCtx, "https://en.wikipedia.org/wiki/Cat")
	if err != nil {
		log.Fatalf("Failed to navigate: %v", err)
	}
	log.Printf("Navigation successful (Frame ID: %s)\n", navResult.FrameID)

	// Wait a bit for page to fully load
	log.Println("Waiting for page to load...")
	time.Sleep(3 * time.Second)

	// Take a screenshot
	log.Println("Taking screenshot...")
	screenshotCtx, screenshotCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer screenshotCancel()

	screenshotParams := protocol.ScreenshotParams{
		Format:   "png",
		FullPage: false, // Viewport only
	}

	screenshotResult, err := c.Screenshot(screenshotCtx, screenshotParams)
	if err != nil {
		log.Fatalf("Failed to take screenshot: %v", err)
	}

	// Decode base64 image data
	imageData, err := base64.StdEncoding.DecodeString(screenshotResult.Data)
	if err != nil {
		log.Fatalf("Failed to decode screenshot data: %v", err)
	}

	// Save to file
	outputFile := "wikipedia_cats.png"
	log.Printf("Saving screenshot to %s (%d bytes)...\n", outputFile, len(imageData))

	err = os.WriteFile(outputFile, imageData, 0644)
	if err != nil {
		log.Fatalf("Failed to write file: %v", err)
	}

	log.Printf("✓ Success! Screenshot saved to %s\n", outputFile)
	fmt.Printf("\nTo view the screenshot:\n")
	fmt.Printf("  open %s\n", outputFile)
}
