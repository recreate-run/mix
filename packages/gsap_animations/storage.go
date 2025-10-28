package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// Storage directory name
const storageDir = "storage"

// InitializeStorage creates the storage directory if it doesn't exist
func InitializeStorage() error {
	dir := getStoragePath()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}
	log.Printf("Storage directory initialized at: %s", dir)
	return nil
}

// getStoragePath returns the absolute path to the storage directory
func getStoragePath() string {
	return filepath.Join(getAnimationsDir(), storageDir)
}

// GenerateUniqueFilename creates a unique filename for exports
func GenerateUniqueFilename() string {
	// Generate 8 random bytes
	randomBytes := make([]byte, 4)
	rand.Read(randomBytes)

	// Create a timestamp-based prefix
	timestamp := time.Now().Format("20060102-150405")

	// Combine timestamp and random hex
	return fmt.Sprintf("%s-%s.mp4", timestamp, hex.EncodeToString(randomBytes))
}

// GetStorageFilePath returns the absolute path to a file in storage
func GetStorageFilePath(filename string) string {
	return filepath.Join(getStoragePath(), filename)
}

// GetStorageURL returns the URL to access a file in storage
func GetStorageURL(r *http.Request, filename string) string {
	// Use path-relative URL to avoid http/https scheme issues
	// This ensures the browser uses the same protocol as the parent page
	return fmt.Sprintf("/storage/%s", filename)
}

// ServeStorageFiles serves files from the storage directory
func ServeStorageFiles(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	setCORSHeaders(w)
	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get file path
	filePath := strings.TrimPrefix(r.URL.Path, "/storage/")
	if filePath == "" || strings.Contains(filePath, "..") {
		http.Error(w, "Invalid file path", http.StatusBadRequest)
		return
	}

	// Read and serve file
	fullPath := filepath.Join(getStoragePath(), filePath)
	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "File not found", http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf("Failed to open file: %v", err), http.StatusInternalServerError)
		}
		return
	}
	defer file.Close()

	// Get file info for Content-Length and Last-Modified headers
	fileInfo, err := file.Stat()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get file info: %v", err), http.StatusInternalServerError)
		return
	}

	// Set content type based on extension
	contentType := "application/octet-stream"
	switch {
	case strings.HasSuffix(filePath, ".mp4"):
		contentType = "video/mp4"
	case strings.HasSuffix(filePath, ".json"):
		contentType = "application/json"
	}

	// Set content headers
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))
	w.Header().Set("Last-Modified", fileInfo.ModTime().UTC().Format(http.TimeFormat))
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", filepath.Base(filePath)))

	// Use http.ServeContent for better range support (needed for videos)
	http.ServeContent(w, r, filePath, fileInfo.ModTime(), file)
}

// UploadToS3 uploads a file to an S3-compatible storage service
func UploadToS3(ctx context.Context, s3URL string, localFilePath string) (string, error) {
	// Parse the S3 URL
	parsedURL, err := url.Parse(s3URL)
	if err != nil {
		return "", fmt.Errorf("invalid S3 URL: %w", err)
	}

	// Extract bucket and key from URL
	bucket := parsedURL.Host
	key := strings.TrimPrefix(parsedURL.Path, "/")
	if key == "" {
		// Use filename as key if not specified
		key = filepath.Base(localFilePath)
	}

	// Extract credentials from URL query parameters if present
	accessKey := parsedURL.Query().Get("access_key")
	secretKey := parsedURL.Query().Get("secret_key")

	// Extract region from query parameters or use default
	region := parsedURL.Query().Get("region")
	if region == "" {
		region = "us-east-1" // Default region
	}

	// Configure AWS session
	config := &aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(accessKey, secretKey, ""),
	}

	// Set custom endpoint if needed (for S3-compatible services)
	if parsedURL.Query().Get("endpoint") != "" {
		config.Endpoint = aws.String(parsedURL.Query().Get("endpoint"))
		config.S3ForcePathStyle = aws.Bool(true)
	}

	sess, err := session.NewSession(config)
	if err != nil {
		return "", fmt.Errorf("failed to create AWS session: %w", err)
	}

	// Create S3 service client
	svc := s3.New(sess)

	// Open file for upload
	file, err := os.Open(localFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to open local file: %w", err)
	}
	defer file.Close()

	// Get file size
	fileInfo, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to get file info: %w", err)
	}

	// Read file content
	buffer := make([]byte, fileInfo.Size())
	_, err = file.Read(buffer)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Upload to S3
	_, err = svc.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(buffer),
		ContentType: aws.String("video/mp4"),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload to S3: %w", err)
	}

	// Return S3 URL of the uploaded file
	return fmt.Sprintf("%s://%s/%s", parsedURL.Scheme, parsedURL.Host, key), nil
}

// setCORSHeaders sets CORS headers for endpoints
func setCORSHeaders(w http.ResponseWriter) {
	// Allow requests from any origin
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Allow common HTTP methods
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

	// Allow common headers
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, Accept, Origin")

	// Expose headers that clients might need to access
	w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Range, Content-Disposition")

	// Set cache control headers for better browser compatibility
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	// Set Cross-Origin-Resource-Policy to allow embedding in any site
	w.Header().Set("Cross-Origin-Resource-Policy", "cross-origin")
}

// handleCORSPreflight handles OPTIONS requests for CORS
func handleCORSPreflight(w http.ResponseWriter, r *http.Request) bool {
	if r.Method == "OPTIONS" {
		setCORSHeaders(w)
		w.WriteHeader(http.StatusOK)
		return true
	}
	return false
}
