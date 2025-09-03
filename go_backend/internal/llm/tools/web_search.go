package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"mix/internal/permission"
)

type WebSearchParams struct {
	Query          string   `json:"query"`
	AllowedDomains []string `json:"allowed_domains,omitempty"`
	BlockedDomains []string `json:"blocked_domains,omitempty"`
}

// Brave Search API response structures
type BraveSearchResponse struct {
	Type string     `json:"type"`
	Web  WebResults `json:"web"`
}

type WebResults struct {
	Type    string         `json:"type"`
	Results []SearchResult `json:"results"`
}

type SearchResult struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description"`
}

type WebSearchPermissionsParams struct {
	Query          string   `json:"query"`
	AllowedDomains []string `json:"allowed_domains,omitempty"`
	BlockedDomains []string `json:"blocked_domains,omitempty"`
}

type webSearchTool struct {
	client      *http.Client
	permissions permission.Service
}

const (
	WebSearchToolName = "web_search"
	MaxSearchResults  = 3
)

func NewWebSearchTool(permissions permission.Service) BaseTool {
	return &webSearchTool{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		permissions: permissions,
	}
}

func (t *webSearchTool) Info() ToolInfo {
	return ToolInfo{
		Name:        WebSearchToolName,
		Description: LoadToolDescription("web_search"),
		Parameters: map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "The search query to use (minimum 2 characters)",
				"minLength":   2,
			},
			"allowed_domains": map[string]any{
				"type":        "array",
				"description": "Only include search results from these domains",
				"items": map[string]any{
					"type": "string",
				},
			},
			"blocked_domains": map[string]any{
				"type":        "array",
				"description": "Never include search results from these domains",
				"items": map[string]any{
					"type": "string",
				},
			},
		},
		Required: []string{"query"},
	}
}

func (t *webSearchTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	var params WebSearchParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return NewTextErrorResponse("Failed to parse web search parameters: " + err.Error()), nil
	}

	if params.Query == "" {
		return NewTextErrorResponse("Query parameter is required"), nil
	}

	if len(params.Query) < 2 {
		return NewTextErrorResponse("Query must be at least 2 characters long"), nil
	}

	// Get API key from environment
	apiKey := os.Getenv("BRAVE_SEARCH_API_KEY")
	if apiKey == "" {
		return NewTextErrorResponse("BRAVE_SEARCH_API_KEY environment variable is not set"), nil
	}

	sessionID, messageID := GetContextValues(ctx)
	if sessionID == "" || messageID == "" {
		return ToolResponse{}, fmt.Errorf("session ID and message ID are required for web search")
	}

	workingDir, err := GetWorkingDirectory(ctx)
	if err != nil {
		return ToolResponse{}, fmt.Errorf("failed to get working directory: %w", err)
	}

	// Request permission for web search
	p := t.permissions.Request(
		permission.CreatePermissionRequest{
			SessionID:   sessionID,
			Path:        workingDir,
			ToolName:    WebSearchToolName,
			Action:      "web_search",
			Description: fmt.Sprintf("Search the web for: %s", params.Query),
			Params:      WebSearchPermissionsParams(params),
		},
	)

	if !p {
		return ToolResponse{}, permission.ErrorPermissionDenied
	}

	// Build the request URL
	u, err := url.Parse("https://api.search.brave.com/res/v1/web/search")
	if err != nil {
		return ToolResponse{}, fmt.Errorf("failed to parse Brave API URL: %w", err)
	}

	q := u.Query()
	q.Set("q", params.Query)
	q.Set("count", "10")
	q.Set("country", "us")
	q.Set("search_lang", "en")

	// Add domain filtering if specified
	if len(params.AllowedDomains) > 0 {
		for _, domain := range params.AllowedDomains {
			// Add site: prefix to each allowed domain
			params.Query += fmt.Sprintf(" site:%s", domain)
		}
		q.Set("q", params.Query)
	}

	if len(params.BlockedDomains) > 0 {
		for _, domain := range params.BlockedDomains {
			// Add -site: prefix to each blocked domain
			params.Query += fmt.Sprintf(" -site:%s", domain)
		}
		q.Set("q", params.Query)
	}

	u.RawQuery = q.Encode()

	// Create the request
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return ToolResponse{}, fmt.Errorf("failed to create request: %w", err)
	}

	// Set the required headers
	req.Header.Set("X-Subscription-Token", apiKey)
	req.Header.Set("User-Agent", "mix/1.0")

	// Execute the request
	resp, err := t.client.Do(req)
	if err != nil {
		return ToolResponse{}, fmt.Errorf("failed to execute web search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return NewTextErrorResponse(fmt.Sprintf("Brave Search API returned status code: %d", resp.StatusCode)), nil
	}

	// Read the response body with size limit
	maxSize := int64(5 * 1024 * 1024) // 5MB limit
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxSize))
	if err != nil {
		return NewTextErrorResponse("Failed to read search results: " + err.Error()), nil
	}

	// Parse the Brave API response
	var braveResponse BraveSearchResponse
	if err := json.Unmarshal(body, &braveResponse); err != nil {
		return NewTextErrorResponse("Failed to parse search results: " + err.Error()), nil
	}

	// Check if we have web results
	if len(braveResponse.Web.Results) == 0 {
		return NewTextResponse("No search results found."), nil
	}

	// Format results for readability, limited to MaxSearchResults
	resultsToShow := len(braveResponse.Web.Results)
	if resultsToShow > MaxSearchResults {
		resultsToShow = MaxSearchResults
	}

	var formattedOutput strings.Builder
	formattedOutput.WriteString(fmt.Sprintf("Search results for: %s\n\n", params.Query))

	for i := 0; i < resultsToShow; i++ {
		result := braveResponse.Web.Results[i]
		formattedOutput.WriteString(fmt.Sprintf("%d. %s\n", i+1, result.Title))
		formattedOutput.WriteString(fmt.Sprintf("   URL: %s\n", result.URL))
		formattedOutput.WriteString(fmt.Sprintf("   Description: %s\n", result.Description))
		if i < resultsToShow-1 {
			formattedOutput.WriteString("\n---\n\n")
		}
	}

	return NewTextResponse(formattedOutput.String()), nil
}