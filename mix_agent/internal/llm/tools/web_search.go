package tools

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"mix/internal/config"
	"mix/internal/permission"
	"mix/internal/tools"
)

type SearchParams struct {
	Query          string   `json:"query"`
	SearchType     string   `json:"search_type,omitempty"`
	AllowedDomains []string `json:"allowed_domains,omitempty"`
	BlockedDomains []string `json:"blocked_domains,omitempty"`
	SafeSearch     string   `json:"safesearch,omitempty"`
	SpellCheck     *bool    `json:"spellcheck,omitempty"`
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

// Image search response structures
type ImageSearchResponse struct {
	Type    string        `json:"type"`
	Results []ImageResult `json:"results"`
}

type ImageResult struct {
	Type        string                `json:"type"`
	Title       string                `json:"title"`
	URL         string                `json:"url"` // Source page URL
	Source      string                `json:"source"`
	PageFetched string                `json:"page_fetched"`
	Thumbnail   ImageResultThumbnail  `json:"thumbnail"`
	Properties  ImageResultProperties `json:"properties"`
	MetaURL     *ImageResultMetaURL   `json:"meta_url,omitempty"`
	Confidence  string                `json:"confidence"`
}

type ImageResultThumbnail struct {
	Src string `json:"src"`
}

type ImageResultProperties struct {
	URL         string `json:"url"` // Actual image URL
	Placeholder string `json:"placeholder"`
}

type ImageResultMetaURL struct {
	Scheme   string `json:"scheme"`
	Netloc   string `json:"netloc"`
	Hostname string `json:"hostname"`
	Favicon  string `json:"favicon"`
	Path     string `json:"path"`
}

// Video search response structures
type VideoSearchResponse struct {
	Type    string        `json:"type"`
	Results []VideoResult `json:"results"`
}

type VideoResult struct {
	Type        string                `json:"type"`
	Title       string                `json:"title"`
	URL         string                `json:"url"` // Source page URL
	Source      string                `json:"source"`
	PageFetched string                `json:"page_fetched"`
	Thumbnail   VideoResultThumbnail  `json:"thumbnail"`
	Properties  VideoResultProperties `json:"properties"`
	MetaURL     *VideoResultMetaURL   `json:"meta_url,omitempty"`
	Confidence  string                `json:"confidence"`
}

type VideoResultThumbnail struct {
	Src string `json:"src"`
}

type VideoResultProperties struct {
	URL         string `json:"url"`         // Actual video URL
	Duration    string `json:"duration"`    // Video duration
	Views       string `json:"views"`       // View count
	UploadDate  string `json:"upload_date"` // Upload date
	Placeholder string `json:"placeholder"`
}

type VideoResultMetaURL struct {
	Scheme   string `json:"scheme"`
	Netloc   string `json:"netloc"`
	Hostname string `json:"hostname"`
	Favicon  string `json:"favicon"`
	Path     string `json:"path"`
}

type SearchPermissionsParams struct {
	Query          string   `json:"query"`
	SearchType     string   `json:"search_type,omitempty"`
	AllowedDomains []string `json:"allowed_domains,omitempty"`
	BlockedDomains []string `json:"blocked_domains,omitempty"`
	SafeSearch     string   `json:"safesearch,omitempty"`
	SpellCheck     *bool    `json:"spellcheck,omitempty"`
}

type searchTool struct {
	client      *http.Client
	permissions permission.Service
}

const (
	SearchToolName   = "search"
	MaxSearchResults = 3
)

func NewWebSearchTool(permissions permission.Service) BaseTool {
	return &searchTool{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		permissions: permissions,
	}
}

func (t *searchTool) Info() ToolInfo {
	return ToolInfo{
		Name:        SearchToolName,
		Description: LoadToolDescription("web_search"),
		Parameters: map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "The search query to use (minimum 2 characters)",
				"minLength":   2,
			},
			"search_type": map[string]any{
				"type":        "string",
				"description": "Type of search to perform",
				"enum":        []string{"web", "images", "videos"},
				"default":     "web",
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
			"safesearch": map[string]any{
				"type":        "string",
				"description": "Safe search level for image search",
				"enum":        []string{"strict", "moderate", "off"},
				"default":     "strict",
			},
			"spellcheck": map[string]any{
				"type":        "boolean",
				"description": "Enable spell correction",
				"default":     true,
			},
		},
		Required: []string{"query"},
	}
}

func (t *searchTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	var params SearchParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return NewTextErrorResponse("Failed to parse search parameters: " + err.Error()), nil
	}

	if params.Query == "" {
		return NewTextErrorResponse("Query parameter is required"), nil
	}

	if len(params.Query) < 2 {
		return NewTextErrorResponse("Query must be at least 2 characters long"), nil
	}

	// Set defaults for search parameters
	if params.SearchType == "" {
		params.SearchType = "web"
	}

	// Validate search type
	if params.SearchType != "web" && params.SearchType != "images" && params.SearchType != "videos" {
		return NewTextErrorResponse("search_type must be 'web', 'images', or 'videos'"), nil
	}

	// Set defaults for image search parameters
	if params.SearchType == "images" {
		if params.SafeSearch == "" {
			params.SafeSearch = "strict"
		}
		if params.SpellCheck == nil {
			defaultSpellCheck := true
			params.SpellCheck = &defaultSpellCheck
		}
	}

	// Get API key from credentials service
	credentialsService := config.GetAPICredentials()
	if credentialsService == nil {
		return NewTextErrorResponse("Credentials service not available"), nil
	}

	apiKey, err := credentialsService.GetToolAPIKey(ctx, tools.ToolTypeWebSearch, tools.WebSearchBrave)
	if err != nil {
		// Fallback to environment variable for backwards compatibility
		envAPIKey := os.Getenv("BRAVE_SEARCH_API_KEY")
		if envAPIKey == "" {
			return NewTextErrorResponse("Brave Search API key not configured. Please set your API key in Settings > Tools & Agents."), nil
		}
		apiKey = envAPIKey
	}

	sessionID, messageID := GetContextValues(ctx)
	if sessionID == "" || messageID == "" {
		return ToolResponse{}, fmt.Errorf("session ID and message ID are required for web search")
	}

	sessionStorageDir, err := GetSessionStorageDirectory(ctx)
	if err != nil {
		return ToolResponse{}, fmt.Errorf("failed to get session storage directory: %w", err)
	}

	// Request permission for search
	var searchType string
	switch params.SearchType {
	case "images":
		searchType = "image"
	case "videos":
		searchType = "video"
	default:
		searchType = "web"
	}
	p := t.permissions.Request(
		permission.CreatePermissionRequest{
			SessionID:   sessionID,
			Path:        sessionStorageDir,
			ToolName:    SearchToolName,
			Action:      searchType + "_search",
			Description: fmt.Sprintf("Search for %s: %s", searchType, params.Query),
			Params:      SearchPermissionsParams(params),
		},
	)

	if !p {
		return ToolResponse{}, permission.ErrorPermissionDenied
	}

	// Build the request URL based on search type
	var baseURL string
	switch params.SearchType {
	case "images":
		baseURL = "https://api.search.brave.com/res/v1/images/search"
	case "videos":
		baseURL = "https://api.search.brave.com/res/v1/videos/search"
	default:
		baseURL = "https://api.search.brave.com/res/v1/web/search"
	}

	u, err := url.Parse(baseURL)
	if err != nil {
		return ToolResponse{}, fmt.Errorf("failed to parse Brave API URL: %w", err)
	}

	q := u.Query()
	q.Set("q", params.Query)
	q.Set("count", "10")
	q.Set("country", "us")
	q.Set("search_lang", "en")

	// Add image-specific parameters
	if params.SearchType == "images" {
		if params.SafeSearch != "" {
			q.Set("safesearch", params.SafeSearch)
		}
		if params.SpellCheck != nil {
			if *params.SpellCheck {
				q.Set("spellcheck", "1")
			} else {
				q.Set("spellcheck", "0")
			}
		}
	}

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
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")

	// Execute the request
	resp, err := t.client.Do(req)
	if err != nil {
		return ToolResponse{}, fmt.Errorf("failed to execute web search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return NewTextErrorResponse(fmt.Sprintf("Brave Search API returned status code: %d", resp.StatusCode)), nil
	}

	// Handle gzip compression
	var reader io.ReadCloser
	if resp.Header.Get("Content-Encoding") == "gzip" {
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			return NewTextErrorResponse("Failed to decompress search results: " + err.Error()), nil
		}
		defer reader.Close()
	} else {
		reader = resp.Body
	}

	// Read the response body with size limit
	maxSize := int64(5 * 1024 * 1024) // 5MB limit
	body, err := io.ReadAll(io.LimitReader(reader, maxSize))
	if err != nil {
		return NewTextErrorResponse("Failed to read search results: " + err.Error()), nil
	}

	// Parse response based on search type and format results
	switch params.SearchType {
	case "images":
		return t.formatImageResults(body, params.Query)
	case "videos":
		return t.formatVideoResults(body, params.Query)
	default:
		return t.formatWebResults(body, params.Query)
	}
}

func (t *searchTool) formatWebResults(body []byte, query string) (ToolResponse, error) {
	var braveResponse BraveSearchResponse
	if err := json.Unmarshal(body, &braveResponse); err != nil {
		return NewTextErrorResponse("Failed to parse web search results: " + err.Error()), nil
	}

	// Check if we have web results
	if len(braveResponse.Web.Results) == 0 {
		return NewTextResponse("No web search results found."), nil
	}

	// Format results for readability, limited to MaxSearchResults
	resultsToShow := len(braveResponse.Web.Results)
	if resultsToShow > MaxSearchResults {
		resultsToShow = MaxSearchResults
	}

	var formattedOutput strings.Builder
	formattedOutput.WriteString(fmt.Sprintf("Web search results for: %s\n\n", query))

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

func (t *searchTool) formatVideoResults(body []byte, query string) (ToolResponse, error) {
	var videoResponse VideoSearchResponse
	if err := json.Unmarshal(body, &videoResponse); err != nil {
		return NewTextErrorResponse("Failed to parse video search results: " + err.Error()), nil
	}

	// Check if we have video results
	if len(videoResponse.Results) == 0 {
		return NewTextResponse("No video search results found."), nil
	}

	// Format results for readability, limited to MaxSearchResults
	resultsToShow := len(videoResponse.Results)
	if resultsToShow > MaxSearchResults {
		resultsToShow = MaxSearchResults
	}

	var formattedOutput strings.Builder
	formattedOutput.WriteString(fmt.Sprintf("Video search results for: %s\n\n", query))

	for i := 0; i < resultsToShow; i++ {
		result := videoResponse.Results[i]
		formattedOutput.WriteString(fmt.Sprintf("%d. %s\n", i+1, result.Title))

		// Show actual video URL (from properties)
		if result.Properties.URL != "" {
			formattedOutput.WriteString(fmt.Sprintf("   Video URL: %s\n", result.Properties.URL))
		}

		// Show duration if available
		if result.Properties.Duration != "" {
			formattedOutput.WriteString(fmt.Sprintf("   Duration: %s\n", result.Properties.Duration))
		}

		// Show view count if available
		if result.Properties.Views != "" {
			formattedOutput.WriteString(fmt.Sprintf("   Views: %s\n", result.Properties.Views))
		}

		// Show upload date if available
		if result.Properties.UploadDate != "" {
			formattedOutput.WriteString(fmt.Sprintf("   Upload Date: %s\n", result.Properties.UploadDate))
		}

		// Show thumbnail URL
		if result.Thumbnail.Src != "" {
			formattedOutput.WriteString(fmt.Sprintf("   Thumbnail: %s\n", result.Thumbnail.Src))
		}

		// Show source information
		if result.Source != "" {
			formattedOutput.WriteString(fmt.Sprintf("   Source: %s\n", result.Source))
		}

		// Show source page URL
		if result.URL != "" {
			formattedOutput.WriteString(fmt.Sprintf("   Source Page: %s\n", result.URL))
		}

		// Show confidence if available
		if result.Confidence != "" {
			formattedOutput.WriteString(fmt.Sprintf("   Confidence: %s\n", result.Confidence))
		}

		if i < resultsToShow-1 {
			formattedOutput.WriteString("\n---\n\n")
		}
	}

	return NewTextResponse(formattedOutput.String()), nil
}

func (t *searchTool) formatImageResults(body []byte, query string) (ToolResponse, error) {
	var imageResponse ImageSearchResponse
	if err := json.Unmarshal(body, &imageResponse); err != nil {
		return NewTextErrorResponse("Failed to parse image search results: " + err.Error()), nil
	}

	// Check if we have image results
	if len(imageResponse.Results) == 0 {
		return NewTextResponse("No image search results found."), nil
	}

	// Format results for readability, limited to MaxSearchResults
	resultsToShow := len(imageResponse.Results)
	if resultsToShow > MaxSearchResults {
		resultsToShow = MaxSearchResults
	}

	var formattedOutput strings.Builder
	formattedOutput.WriteString(fmt.Sprintf("Image search results for: %s\n\n", query))

	for i := 0; i < resultsToShow; i++ {
		result := imageResponse.Results[i]
		formattedOutput.WriteString(fmt.Sprintf("%d. %s\n", i+1, result.Title))

		// Show actual image URL (from properties)
		if result.Properties.URL != "" {
			formattedOutput.WriteString(fmt.Sprintf("   Image URL: %s\n", result.Properties.URL))
		}

		// Show thumbnail URL
		if result.Thumbnail.Src != "" {
			formattedOutput.WriteString(fmt.Sprintf("   Thumbnail: %s\n", result.Thumbnail.Src))
		}

		// Show source information
		if result.Source != "" {
			formattedOutput.WriteString(fmt.Sprintf("   Source: %s\n", result.Source))
		}

		// Show source page URL
		if result.URL != "" {
			formattedOutput.WriteString(fmt.Sprintf("   Source Page: %s\n", result.URL))
		}

		// Show confidence if available
		if result.Confidence != "" {
			formattedOutput.WriteString(fmt.Sprintf("   Confidence: %s\n", result.Confidence))
		}

		if i < resultsToShow-1 {
			formattedOutput.WriteString("\n---\n\n")
		}
	}

	return NewTextResponse(formattedOutput.String()), nil
}
