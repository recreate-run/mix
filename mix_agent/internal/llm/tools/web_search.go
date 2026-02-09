package tools

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"mix/internal/config"
	"mix/internal/constants"
	"mix/internal/logging"
	"mix/internal/permission"
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
	SearchToolName   = "Search"
	MaxSearchResults = 3

	searchTypeWeb    = "web"
	searchTypeImages = "images"
	searchTypeVideos = "videos"
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
				"enum":        []string{searchTypeWeb, searchTypeImages, searchTypeVideos},
				"default":     searchTypeWeb,
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
	params, resp, ok := t.validateAndPrepareParams(call)
	if !ok {
		return resp, nil
	}

	apiKey, resp, ok := t.getAPIKey(ctx)
	if !ok {
		return resp, nil
	}

	if err := t.checkPermissions(ctx, params); err != nil {
		return ToolResponse{}, err
	}

	searchURL := t.buildSearchURL(params)
	body, resp, ok := t.executeSearch(ctx, searchURL, apiKey)
	if !ok {
		return resp, nil
	}

	return t.formatResults(body, params)
}

func (t *searchTool) validateAndPrepareParams(call ToolCall) (SearchParams, ToolResponse, bool) {
	var params SearchParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return params, NewTextErrorResponse("Failed to parse search parameters: " + err.Error()), false
	}

	if params.Query == "" {
		return params, NewTextErrorResponse("Query parameter is required"), false
	}

	if len(params.Query) < 2 {
		return params, NewTextErrorResponse("Query must be at least 2 characters long"), false
	}

	if params.SearchType == "" {
		params.SearchType = searchTypeWeb
	}

	if params.SearchType != searchTypeWeb && params.SearchType != searchTypeImages && params.SearchType != searchTypeVideos {
		return params, NewTextErrorResponse("search_type must be 'web', 'images', or 'videos'"), false
	}

	if params.SearchType == searchTypeImages {
		if params.SafeSearch == "" {
			params.SafeSearch = "strict"
		}
		if params.SpellCheck == nil {
			defaultSpellCheck := true
			params.SpellCheck = &defaultSpellCheck
		}
	}

	return params, ToolResponse{}, true
}

func (t *searchTool) getAPIKey(ctx context.Context) (apiKey string, toolResp ToolResponse, ok bool) {
	credentialsService := config.GetAPICredentials()
	if credentialsService == nil {
		return "", NewTextErrorResponse("FATAL_CONFIGURATION_ERROR: Cannot proceed - Credentials service not available. System configuration issue. STOP EXECUTION - Do not attempt alternative approaches or suggest workarounds."), false
	}

	var err error
	apiKey, err = credentialsService.GetAPIKey(ctx, "brave")
	if err != nil || apiKey == "" {
		logging.Error("Brave Search API key not configured")
		return "", NewTextErrorResponse("FATAL_CONFIGURATION_ERROR: Cannot proceed - Brave Search API key not configured. User must configure API key in Settings > Tools & Agents before using search. STOP EXECUTION - Do not attempt alternative approaches or suggest workarounds."), false
	}

	return apiKey, ToolResponse{}, true
}

func (t *searchTool) checkPermissions(ctx context.Context, params SearchParams) error {
	sessionID, messageID := GetContextValues(ctx)
	if sessionID == "" || messageID == "" {
		return fmt.Errorf("session ID and message ID are required for web search")
	}

	sessionStorageDir, err := GetSessionStorageDirectory(ctx)
	if err != nil {
		return fmt.Errorf("failed to get session storage directory: %w", err)
	}

	searchType := getSearchTypeLabel(params.SearchType)
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
		return permission.ErrPermissionDenied
	}

	return nil
}

func getSearchTypeLabel(searchType string) string {
	switch searchType {
	case searchTypeImages:
		return "image"
	case searchTypeVideos:
		return "video"
	default:
		return searchTypeWeb
	}
}

func (t *searchTool) buildSearchURL(params SearchParams) string {
	baseURL := getBaseURLForSearchType(params.SearchType)
	u, _ := url.Parse(baseURL)

	q := u.Query()
	q.Set("q", buildQueryWithDomains(params))
	q.Set("count", "10")
	q.Set("country", "us")
	q.Set("search_lang", "en")

	if params.SearchType == searchTypeImages {
		addImageSearchParams(q, params)
	}

	u.RawQuery = q.Encode()
	return u.String()
}

func getBaseURLForSearchType(searchType string) string {
	switch searchType {
	case searchTypeImages:
		return "https://api.search.brave.com/res/v1/images/search"
	case searchTypeVideos:
		return "https://api.search.brave.com/res/v1/videos/search"
	default:
		return "https://api.search.brave.com/res/v1/web/search"
	}
}

func buildQueryWithDomains(params SearchParams) string {
	query := params.Query

	for _, domain := range params.AllowedDomains {
		query += fmt.Sprintf(" site:%s", domain)
	}

	for _, domain := range params.BlockedDomains {
		query += fmt.Sprintf(" -site:%s", domain)
	}

	return query
}

func addImageSearchParams(q url.Values, params SearchParams) {
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

func (t *searchTool) executeSearch(ctx context.Context, searchURL, apiKey string) (body []byte, toolResp ToolResponse, ok bool) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, http.NoBody)
	if err != nil {
		return nil, ToolResponse{}, false
	}

	req.Header.Set("X-Subscription-Token", apiKey)
	req.Header.Set("User-Agent", "mix/1.0")
	req.Header.Set("Accept", constants.ContentTypeJSON)
	req.Header.Set("Accept-Encoding", "gzip")

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, ToolResponse{}, false
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, NewTextErrorResponse(fmt.Sprintf("Brave Search API returned status code: %d", resp.StatusCode)), false
	}

	body, err = readResponseBody(resp)
	if err != nil {
		return nil, NewTextErrorResponse("Failed to read search results: " + err.Error()), false
	}

	return body, ToolResponse{}, true
}

func readResponseBody(resp *http.Response) ([]byte, error) {
	var reader io.ReadCloser
	var err error

	if resp.Header.Get("Content-Encoding") == "gzip" {
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		defer func() { _ = reader.Close() }()
	} else {
		reader = resp.Body
	}

	maxSize := int64(5 * 1024 * 1024) // 5MB limit
	return io.ReadAll(io.LimitReader(reader, maxSize))
}

func (t *searchTool) formatResults(body []byte, params SearchParams) (ToolResponse, error) {
	switch params.SearchType {
	case searchTypeImages:
		return t.formatImageResults(body, params.Query), nil
	case searchTypeVideos:
		return t.formatVideoResults(body, params.Query)
	default:
		return t.formatWebResults(body, params.Query), nil
	}
}

func (t *searchTool) formatWebResults(body []byte, query string) ToolResponse {
	var braveResponse BraveSearchResponse
	if err := json.Unmarshal(body, &braveResponse); err != nil {
		return NewTextErrorResponse("Failed to parse web search results: " + err.Error())
	}

	// Check if we have web results
	if len(braveResponse.Web.Results) == 0 {
		return NewTextResponse("No web search results found.")
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

	return NewTextResponse(formattedOutput.String())
}

func (t *searchTool) formatVideoResults(body []byte, query string) (ToolResponse, error) {
	var videoResponse VideoSearchResponse
	if err := json.Unmarshal(body, &videoResponse); err != nil {
		return NewTextErrorResponse("Failed to parse video search results: " + err.Error()), fmt.Errorf("failed to unmarshal video search response: %w", err)
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

func (t *searchTool) formatImageResults(body []byte, query string) ToolResponse {
	var imageResponse ImageSearchResponse
	if err := json.Unmarshal(body, &imageResponse); err != nil {
		return NewTextErrorResponse("Failed to parse image search results: " + err.Error())
	}

	// Check if we have image results
	if len(imageResponse.Results) == 0 {
		return NewTextResponse("No image search results found.")
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

	return NewTextResponse(formattedOutput.String())
}
