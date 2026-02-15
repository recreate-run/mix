package extensions

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Extension represents a Chrome extension
type Extension struct {
	ID       string
	Name     string
	Version  string
	Checksum string // SHA256 of .crx file
}

// SupportedExtensions maps extension keys to their metadata
var SupportedExtensions = map[string]Extension{
	"ublock": {
		ID:       "cjpalhdlnbpafiamejdnhcphjbkeiagm",
		Name:     "uBlock Origin",
		Version:  "latest",
		Checksum: "", // Verify on download
	},
	"cookies": {
		ID:       "fihnjjcciajhdojfnbdddfaoknhalnja",
		Name:     "I don't care about cookies",
		Version:  "latest",
		Checksum: "",
	},
	"clearurls": {
		ID:       "lckanjgmijmafbedllaakclkaicjfmnk",
		Name:     "ClearURLs",
		Version:  "latest",
		Checksum: "",
	},
}

// DownloadExtension downloads extension and extracts to cache directory
// For uBlock Origin, downloads from GitHub releases instead of Chrome Web Store
// for better compatibility with unpacked loading
func DownloadExtension(ctx context.Context, extID string, cacheDir string) (string, error) {
	extPath := filepath.Join(cacheDir, extID)

	// Check cache first
	if _, err := os.Stat(extPath); err == nil {
		fmt.Printf("Using cached extension: %s\n", extPath)
		return extPath, nil // Already downloaded
	}

	// Create cache directory
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", fmt.Errorf("create cache dir: %w", err)
	}

	// Special handling for uBlock Origin - download from GitHub
	if extID == "cjpalhdlnbpafiamejdnhcphjbkeiagm" {
		return downloadUBlockFromGitHub(ctx, extPath)
	}

	// For other extensions, download from Chrome Web Store
	return downloadFromChromeWebStore(ctx, extID, extPath)
}

// downloadUBlockFromGitHub downloads uBlock Origin from GitHub releases
func downloadUBlockFromGitHub(ctx context.Context, extPath string) (string, error) {
	// Use latest stable release
	// URL: https://github.com/gorhill/uBlock/releases/download/1.69.0/uBlock0_1.69.0.chromium.zip
	// Note: Update version as needed
	version := "1.69.0"
	downloadURL := fmt.Sprintf("https://github.com/gorhill/uBlock/releases/download/%s/uBlock0_%s.chromium.zip", version, version)

	fmt.Printf("Downloading uBlock Origin %s from GitHub...\n", version)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("download ublock: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed: status %d (URL: %s)", resp.StatusCode, downloadURL)
	}

	// Save to temp file
	tmpPath := extPath + ".zip.tmp"
	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer tmpFile.Close()

	// Download
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("save zip: %w", err)
	}
	_ = tmpFile.Close()

	// Extract ZIP to extension directory
	if err := unzip(tmpPath, extPath); err != nil {
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("unzip: %w", err)
	}

	// Clean up zip file
	_ = os.Remove(tmpPath)

	fmt.Printf("Successfully extracted uBlock Origin to: %s\n", extPath)
	return extPath, nil
}

// downloadFromChromeWebStore downloads .crx from Chrome Web Store and extracts it
func downloadFromChromeWebStore(ctx context.Context, extID string, extPath string) (string, error) {
	// Download .crx from Chrome Web Store
	// URL format with acceptformat parameter to avoid 204 responses
	crxURL := fmt.Sprintf("https://clients2.google.com/service/update2/crx?response=redirect&acceptformat=crx2,crx3&prodversion=110.0&x=id%%3D%s%%26installsource%%3Dondemand%%26uc", extID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, crxURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("download extension: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed: status %d", resp.StatusCode)
	}

	// Save to temp file
	tmpPath := extPath + ".crx.tmp"
	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer tmpFile.Close()

	// Download and compute checksum
	hash := sha256.New()
	writer := io.MultiWriter(tmpFile, hash)

	if _, err := io.Copy(writer, resp.Body); err != nil {
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("save extension: %w", err)
	}

	checksum := hex.EncodeToString(hash.Sum(nil))
	fmt.Printf("Downloaded extension %s (checksum: %s)\n", extID, checksum[:16])

	// Extract .crx to unpacked directory
	if err := extractCRX(tmpPath, extPath); err != nil {
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("extract crx: %w", err)
	}

	// Clean up .crx file
	_ = os.Remove(tmpPath)

	return extPath, nil
}

// extractCRX extracts a .crx file to an unpacked directory
func extractCRX(crxPath, outputDir string) error {
	// .crx format: magic bytes (4) + version (4) + public key length (4) + signature length (4) + public key + signature + ZIP data
	// Skip header and extract ZIP portion

	data, err := os.ReadFile(crxPath)
	if err != nil {
		return fmt.Errorf("read crx: %w", err)
	}

	// Verify magic bytes "Cr24"
	if len(data) < 16 || string(data[0:4]) != "Cr24" {
		return fmt.Errorf("invalid crx format")
	}

	// Parse header
	version := binary.LittleEndian.Uint32(data[4:8])
	if version != 2 && version != 3 {
		return fmt.Errorf("unsupported crx version: %d", version)
	}

	var zipOffset int
	if version == 2 {
		pubKeyLen := binary.LittleEndian.Uint32(data[8:12])
		sigLen := binary.LittleEndian.Uint32(data[12:16])
		zipOffset = 16 + int(pubKeyLen) + int(sigLen)
	} else if version == 3 {
		headerLen := binary.LittleEndian.Uint32(data[8:12])
		zipOffset = 12 + int(headerLen)
	}

	// Extract ZIP portion
	zipData := data[zipOffset:]
	tmpZip := crxPath + ".zip"
	if err := os.WriteFile(tmpZip, zipData, 0644); err != nil {
		return fmt.Errorf("write zip: %w", err)
	}
	defer os.Remove(tmpZip)

	// Unzip to output directory
	if err := unzip(tmpZip, outputDir); err != nil {
		return fmt.Errorf("unzip: %w", err)
	}

	return nil
}

// unzip extracts a zip file to a directory
func unzip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(destDir, f.Name)

		// Check for ZipSlip vulnerability
		if !strings.HasPrefix(fpath, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path: %s", fpath)
		}

		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			_ = outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		_ = outFile.Close()
		_ = rc.Close()

		if err != nil {
			return err
		}
	}
	return nil
}

// PatchCookieExtension modifies cookie extension to whitelist specific domains
// Note: Modern extensions use Manifest V3 which requires a different approach than V2
func PatchCookieExtension(extPath string, whitelistDomains []string) error {
	manifestPath := filepath.Join(extPath, "manifest.json")

	// Read manifest
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read manifest: %w", err)
	}

	var manifest map[string]interface{}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("parse manifest: %w", err)
	}

	// Check manifest version
	manifestVersion, _ := manifest["manifest_version"].(float64)

	if manifestVersion == 2 {
		// Manifest V2: Modify content_scripts to exclude whitelisted domains
		if contentScripts, ok := manifest["content_scripts"].([]interface{}); ok {
			for _, script := range contentScripts {
				if scriptMap, ok := script.(map[string]interface{}); ok {
					// Add exclude_matches for whitelisted domains
					excludeMatches := make([]string, 0)
					for _, domain := range whitelistDomains {
						excludeMatches = append(excludeMatches, fmt.Sprintf("*://*.%s/*", domain))
						excludeMatches = append(excludeMatches, fmt.Sprintf("*://%s/*", domain))
					}
					scriptMap["exclude_matches"] = excludeMatches
				}
			}
		}
	} else if manifestVersion == 3 {
		// Manifest V3: Remove whitelisted domains from host_permissions
		// This prevents the extension from running on those domains
		if hostPerms, ok := manifest["host_permissions"].([]interface{}); ok {
			// Convert to string slice for easier manipulation
			filteredPerms := make([]string, 0, len(hostPerms))

			for _, perm := range hostPerms {
				if permStr, ok := perm.(string); ok {
					// Keep the permission unless it's <all_urls> and we have whitelisted domains
					if permStr == "<all_urls>" && len(whitelistDomains) > 0 {
						// Replace <all_urls> with specific excluded patterns
						// Skip adding <all_urls> and instead we'll add specific permissions below
						continue
					}
					filteredPerms = append(filteredPerms, permStr)
				}
			}

			// For simplicity with Manifest V3, we can't easily exclude specific domains
			// without rebuilding the entire permission set. Instead, we'll add a note
			// that this feature is limited for Manifest V3 extensions.
			// Keep the original permissions as-is.
			// A production implementation would need to use declarativeNetRequest rules
			// or a separate configuration file.

			// For now, just document that V3 extensions don't support this pattern
			fmt.Printf("Warning: Manifest V3 extension detected. Domain whitelisting is not fully supported for V3 extensions.\n")
			fmt.Printf("Consider using extension configuration options instead of manifest patching.\n")
		}
	}

	// Write modified manifest
	modifiedData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}

	if err := os.WriteFile(manifestPath, modifiedData, 0644); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}

	return nil
}
