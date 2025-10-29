package tools

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	lru "github.com/hashicorp/golang-lru/v2"
	"mix/internal/permission"
)

type WebFetchParams struct {
	URL    string `json:"url"`
	Prompt string `json:"prompt"`
}

type WebFetchPermissionsParams struct {
	URL    string `json:"url"`
	Prompt string `json:"prompt"`
}

type WebFetchResponseMetadata struct {
	URL         string `json:"url"`
	FetchedURL  string `json:"fetched_url,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	StartTime   int64  `json:"start_time"`
	EndTime     int64  `json:"end_time"`
}

type webFetchTool struct {
	client      *http.Client
	permissions permission.Service
	cache       *lru.Cache[string, cacheEntry]
}

type cacheEntry struct {
	content     string
	timestamp   time.Time
	contentType string
	fetchTime   time.Time
}

const (
	WebFetchToolName = "WebFetch"
	MaxContentSize   = 5 * 1024 * 1024 // 5MB limit
	CacheTTL         = 15 * time.Minute
	CacheSize        = 100
)

func NewWebFetchTool(permissions permission.Service) BaseTool {
	cache, _ := lru.New[string, cacheEntry](CacheSize)

	return &webFetchTool{
		client: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// Allow up to 10 redirects
				if len(via) >= 10 {
					return fmt.Errorf("stopped after 10 redirects")
				}
				return nil
			},
		},
		permissions: permissions,
		cache:       cache,
	}
}

func (w *webFetchTool) Info() ToolInfo {
	return ToolInfo{
		Name:        WebFetchToolName,
		Description: LoadToolDescription("web_fetch"),
		Parameters: map[string]any{
			"url": map[string]any{
				"type":        "string",
				"description": "The URL to fetch content from",
				"format":      "uri",
			},
			"prompt": map[string]any{
				"type":        "string",
				"description": "The prompt to run on the fetched content",
			},
		},
		Required: []string{"url", "prompt"},
	}
}

// validateWebFetchParams validates the input parameters
func (w *webFetchTool) validateWebFetchParams(params WebFetchParams) error {
	if params.URL == "" {
		return fmt.Errorf("url parameter is required")
	}
	if params.Prompt == "" {
		return fmt.Errorf("prompt parameter is required")
	}
	return nil
}

// requestFetchPermission requests permission to fetch the URL
func (w *webFetchTool) requestFetchPermission(ctx context.Context, normalizedURL, prompt string) error {
	sessionID, messageID := GetContextValues(ctx)
	if sessionID == "" || messageID == "" {
		return fmt.Errorf("session ID and message ID are required for web fetch")
	}

	sessionStorageDir, err := GetSessionStorageDirectory(ctx)
	if err != nil {
		return fmt.Errorf("failed to get session storage directory: %w", err)
	}

	p := w.permissions.Request(
		permission.CreatePermissionRequest{
			SessionID:   sessionID,
			Path:        sessionStorageDir,
			ToolName:    WebFetchToolName,
			Action:      fmt.Sprintf("Fetch URL: %s", normalizedURL),
			Description: fmt.Sprintf("Fetch and analyze content from: %s", normalizedURL),
			Params: WebFetchPermissionsParams{
				URL:    normalizedURL,
				Prompt: prompt,
			},
		},
	)

	if !p {
		return permission.ErrPermissionDenied
	}

	return nil
}

// fetchURL performs the HTTP request and returns the response
func (w *webFetchTool) fetchURL(ctx context.Context, normalizedURL string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, normalizedURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mix/1.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := w.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	return resp, nil
}

// processHTMLContent converts HTML to markdown or returns plain text
func (w *webFetchTool) processHTMLContent(body []byte, contentType string) (string, error) {
	if strings.Contains(contentType, "text/html") {
		converter := md.NewConverter("", true, nil)
		markdown, err := converter.ConvertString(string(body))
		if err != nil {
			return "", fmt.Errorf("failed to convert HTML to markdown: %w", err)
		}
		return markdown, nil
	}
	return string(body), nil
}

func (w *webFetchTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	var params WebFetchParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return NewTextErrorResponse("invalid parameters: " + err.Error()), nil
	}

	if err := w.validateWebFetchParams(params); err != nil {
		return NewTextErrorResponse(err.Error()), nil
	}

	normalizedURL, err := w.normalizeURL(params.URL)
	if err != nil {
		return NewTextErrorResponse("invalid URL: " + err.Error()), nil
	}

	// Check cache
	cacheKey := w.getCacheKey(normalizedURL)
	if cached, ok := w.cache.Get(cacheKey); ok {
		if time.Since(cached.timestamp) < CacheTTL {
			return w.processContent(cached.content, params.Prompt, normalizedURL, normalizedURL, cached.contentType, cached.fetchTime), nil
		}
		w.cache.Remove(cacheKey)
	}

	if err := w.requestFetchPermission(ctx, normalizedURL, params.Prompt); err != nil {
		return ToolResponse{}, err
	}

	startTime := time.Now()
	resp, err := w.fetchURL(ctx, normalizedURL)
	if err != nil {
		return NewTextErrorResponse("Failed to fetch URL: " + err.Error()), nil
	}
	defer func() { _ = resp.Body.Close() }()

	// Check for cross-domain redirect
	finalURL := resp.Request.URL.String()
	parsedNormalized, _ := url.Parse(normalizedURL)
	if parsedNormalized.Host != resp.Request.URL.Host {
		return NewTextResponse(fmt.Sprintf(
			"The URL redirected to a different host.\n\nOriginal: %s\nRedirect: %s\n\nPlease make a new WebFetch request with the redirect URL to fetch the content.",
			normalizedURL,
			finalURL,
		)), nil
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, MaxContentSize))
	if err != nil {
		return NewTextErrorResponse("Failed to read response body: " + err.Error()), nil
	}

	contentType := resp.Header.Get("Content-Type")
	content, err := w.processHTMLContent(body, contentType)
	if err != nil {
		return NewTextErrorResponse(err.Error()), nil
	}

	w.cache.Add(cacheKey, cacheEntry{
		content:     content,
		timestamp:   time.Now(),
		contentType: contentType,
		fetchTime:   startTime,
	})

	return w.processContent(content, params.Prompt, normalizedURL, finalURL, contentType, startTime), nil
}

func (w *webFetchTool) processContent(content, prompt, originalURL, finalURL, contentType string, startTime time.Time) ToolResponse {
	// Format the output with the prompt context
	// The main agent's LLM will naturally process this content in context
	output := fmt.Sprintf("Content from %s (responding to: %s):\n\n%s", originalURL, prompt, content)

	metadata := WebFetchResponseMetadata{
		URL:         originalURL,
		FetchedURL:  finalURL,
		ContentType: contentType,
		StartTime:   startTime.UnixMilli(),
		EndTime:     time.Now().UnixMilli(),
	}

	return WithResponseMetadata(NewTextResponse(output), metadata)
}

func (w *webFetchTool) normalizeURL(urlStr string) (string, error) {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}

	// Upgrade HTTP to HTTPS
	if parsedURL.Scheme == "http" {
		parsedURL.Scheme = "https"
	}

	if parsedURL.Scheme != "https" {
		return "", fmt.Errorf("only HTTPS URLs are supported")
	}

	return parsedURL.String(), nil
}

func (w *webFetchTool) getCacheKey(urlStr string) string {
	hash := sha256.Sum256([]byte(urlStr))
	return fmt.Sprintf("%x", hash)
}
