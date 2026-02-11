package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mix/internal/config"
	"mix/internal/llm/interfaces"
	"mix/internal/logging"
	"mix/internal/message"

	"github.com/google/uuid"
	"google.golang.org/genai"
)

type geminiOptions struct {
	disableCache       bool
	responseMIMEType   string
	responseJSONSchema map[string]any
	mediaResolution    *genai.MediaResolution // Optional media resolution override
}

type GeminiOption func(*geminiOptions)

type geminiClient struct {
	providerOptions providerClientOptions
	options         geminiOptions
	client          *genai.Client
}

type GeminiClient interfaces.ProviderClient

func newGeminiClient(opts providerClientOptions) GeminiClient {
	geminiOpts := geminiOptions{}
	for _, o := range opts.geminiOptions {
		o(&geminiOpts)
	}

	client, err := genai.NewClient(context.Background(), &genai.ClientConfig{APIKey: opts.apiKey, Backend: genai.BackendGeminiAPI})
	if err != nil {
		logging.Error("Failed to create Gemini client", "error", err)
		return nil
	}

	return &geminiClient{
		providerOptions: opts,
		options:         geminiOpts,
		client:          client,
	}
}

func (g *geminiClient) convertMessages(messages []message.Message) []*genai.Content {
	var history []*genai.Content
	for _, msg := range messages {
		switch msg.Role {
		case message.User:
			var parts []*genai.Part
			parts = append(parts, &genai.Part{Text: msg.Content().String()})
			for _, binaryContent := range msg.BinaryContent() {
				// Handle images and videos via inline data for supported formats
				if g.isSupportedInlineFormat(binaryContent.MIMEType) || g.isSupportedVideoFormat(binaryContent.MIMEType) {
					part := &genai.Part{InlineData: &genai.Blob{
						MIMEType: binaryContent.MIMEType,
						Data:     binaryContent.Data,
					}}

					// Add video metadata if provided
					if binaryContent.StartOffset != "" && binaryContent.EndOffset != "" {
						startDuration, err := time.ParseDuration(binaryContent.StartOffset)
						if err != nil {
							logging.Warn("Failed to parse video start offset", "offset", binaryContent.StartOffset, "error", err)
						} else {
							endDuration, err := time.ParseDuration(binaryContent.EndOffset)
							if err != nil {
								logging.Warn("Failed to parse video end offset", "offset", binaryContent.EndOffset, "error", err)
							} else {
								part.VideoMetadata = &genai.VideoMetadata{
									StartOffset: startDuration,
									EndOffset:   endDuration,
								}
							}
						}
					}

					parts = append(parts, part)
				} else {
					// For unsupported inline formats, log warning and skip
					// Note: Video upload via File API would require additional implementation
					logging.Warn("Unsupported inline format, skipping file", "mimeType", binaryContent.MIMEType)
					continue
				}
			}
			for _, uriContent := range msg.URIContent() {
				// Handle URIs using Gemini's native URI support
				part := genai.NewPartFromURI(uriContent.URI, uriContent.MIMEType)

				// Add video metadata if provided
				if uriContent.StartOffset != "" && uriContent.EndOffset != "" {
					startDuration, err := time.ParseDuration(uriContent.StartOffset)
					if err != nil {
						logging.Warn("Failed to parse video start offset", "offset", uriContent.StartOffset, "error", err)
					} else {
						endDuration, err := time.ParseDuration(uriContent.EndOffset)
						if err != nil {
							logging.Warn("Failed to parse video end offset", "offset", uriContent.EndOffset, "error", err)
						} else {
							part.VideoMetadata = &genai.VideoMetadata{
								StartOffset: startDuration,
								EndOffset:   endDuration,
							}
						}
					}
				}

				parts = append(parts, part)
			}
			history = append(history, &genai.Content{
				Parts: parts,
				Role:  "user",
			})
		case message.Assistant:
			var assistantParts []*genai.Part

			if msg.Content().String() != "" {
				assistantParts = append(assistantParts, &genai.Part{Text: msg.Content().String()})
			}

			if len(msg.ToolCalls()) > 0 {
				for _, call := range msg.ToolCalls() {
					args, _ := parseJsonToMap(call.Input)
					assistantParts = append(assistantParts, &genai.Part{
						FunctionCall: &genai.FunctionCall{
							Name: call.Name,
							Args: args,
						},
					})
				}
			}

			if len(assistantParts) > 0 {
				history = append(history, &genai.Content{
					Role:  "model",
					Parts: assistantParts,
				})
			}

		case message.Tool:
			for _, result := range msg.ToolResults() {
				response := map[string]interface{}{"result": result.Content}
				parsed, err := parseJsonToMap(result.Content)
				if err == nil {
					response = parsed
				}

				var toolCall message.ToolCall
				for _, m := range messages {
					if m.Role == message.Assistant {
						for _, call := range m.ToolCalls() {
							if call.ID == result.ToolCallID {
								toolCall = call
								break
							}
						}
					}
				}

				history = append(history, &genai.Content{
					Parts: []*genai.Part{
						{
							FunctionResponse: &genai.FunctionResponse{
								Name:     toolCall.Name,
								Response: response,
							},
						},
					},
					Role: "function",
				})
			}
		}
	}

	return history
}

func (g *geminiClient) convertTools(tools []interfaces.BaseTool) []*genai.Tool {
	geminiTool := &genai.Tool{}
	geminiTool.FunctionDeclarations = make([]*genai.FunctionDeclaration, 0, len(tools))

	for _, tool := range tools {
		info := tool.Info()
		declaration := &genai.FunctionDeclaration{
			Name:        info.Name,
			Description: info.Description,
			Parameters: &genai.Schema{
				Type:       genai.TypeObject,
				Properties: convertSchemaProperties(info.Parameters),
				Required:   info.Required,
			},
		}

		geminiTool.FunctionDeclarations = append(geminiTool.FunctionDeclarations, declaration)
	}

	return []*genai.Tool{geminiTool}
}

func (g *geminiClient) finishReason(reason genai.FinishReason) message.FinishReason {
	switch reason {
	case genai.FinishReasonStop:
		return message.FinishReasonEndTurn
	case genai.FinishReasonMaxTokens:
		return message.FinishReasonMaxTokens
	default:
		return message.FinishReasonUnknown
	}
}

func (g *geminiClient) Send(ctx context.Context, messages []message.Message, tools []interfaces.BaseTool) (*interfaces.ProviderResponse, error) {
	// Convert messages
	geminiMessages := g.convertMessages(messages)

	cfg := config.Get()
	if cfg != nil && cfg.Debug {
		jsonData, _ := json.Marshal(geminiMessages)
		logging.Debug("Prepared messages", "messages", string(jsonData))
	}

	history := geminiMessages[:len(geminiMessages)-1] // All but last message
	lastMsg := geminiMessages[len(geminiMessages)-1]
	genCfg := &genai.GenerateContentConfig{
		MaxOutputTokens: int32(g.providerOptions.maxTokens),
	}

	// Set temperature (default to 1.0 if not specified)
	if g.providerOptions.temperature != nil {
		genCfg.Temperature = g.providerOptions.temperature
	} else {
		defaultTemp := float32(1.0)
		genCfg.Temperature = &defaultTemp
	}

	// Set structured output options if provided
	if g.options.responseMIMEType != "" {
		genCfg.ResponseMIMEType = g.options.responseMIMEType
	}
	if g.options.responseJSONSchema != nil {
		genCfg.ResponseJsonSchema = g.options.responseJSONSchema
	}

	// Set media resolution based on content type
	mediaRes := g.getMediaResolutionForMessages(messages)
	if mediaRes != genai.MediaResolutionUnspecified {
		genCfg.MediaResolution = mediaRes
	}

	// Only add system instruction if we have a non-empty system message
	if g.providerOptions.systemMessage != "" {
		genCfg.SystemInstruction = &genai.Content{
			Parts: []*genai.Part{{Text: g.providerOptions.systemMessage}},
			Role:  "user",
		}
	}
	if len(tools) > 0 {
		genCfg.Tools = g.convertTools(tools)
	}
	chat, _ := g.client.Chats.Create(ctx, g.providerOptions.model.APIModel, genCfg, history)

	attempts := 0
	for {
		attempts++
		var toolCalls []message.ToolCall

		var lastMsgParts []genai.Part
		for _, part := range lastMsg.Parts {
			lastMsgParts = append(lastMsgParts, *part)
		}
		resp, err := chat.SendMessage(ctx, lastMsgParts...)
		// If there is an error we are going to see if we can retry the call
		if err != nil {
			retry, after, retryErr := g.shouldRetry(attempts, err)
			if retryErr != nil {
				return nil, retryErr
			}
			if retry {
				logging.Warn(fmt.Sprintf("Retrying due to rate limit... attempt %d of %d", attempts, maxRetries))
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(time.Duration(after) * time.Millisecond):
					continue
				}
			}
			return nil, retryErr
		}

		content := ""

		if len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil {
			for _, part := range resp.Candidates[0].Content.Parts {
				switch {
				case part.Text != "":
					content = part.Text
				case part.FunctionCall != nil:
					id := "call_" + uuid.New().String()
					args, _ := json.Marshal(part.FunctionCall.Args)
					toolCalls = append(toolCalls, message.ToolCall{
						ID:       id,
						Name:     part.FunctionCall.Name,
						Input:    string(args),
						Type:     "function",
						Finished: true,
					})
				}
			}
		}

		// Check for completely empty response (no content and no tool calls)
		if content == "" && len(toolCalls) == 0 {
			logging.Warn("Gemini returned empty response with no content or tool calls")
			// Extract sessionID from context and log detailed debug information
			if sessionID, ok := ctx.Value(interfaces.SessionIDContextKey).(string); ok {
				g.logEmptyResponseDetails(sessionID, messages, tools, resp)
			}
		}

		finishReason := message.FinishReasonEndTurn
		if len(resp.Candidates) > 0 {
			finishReason = g.finishReason(resp.Candidates[0].FinishReason)
		}
		if len(toolCalls) > 0 {
			finishReason = message.FinishReasonToolUse
		}

		return &interfaces.ProviderResponse{
			Content:      content,
			ToolCalls:    toolCalls,
			Usage:        g.usage(resp),
			FinishReason: finishReason,
		}, nil
	}
}

func (g *geminiClient) Stream(ctx context.Context, messages []message.Message, tools []interfaces.BaseTool) <-chan interfaces.ProviderEvent {
	// Convert messages
	geminiMessages := g.convertMessages(messages)

	cfg := config.Get()
	if cfg != nil && cfg.Debug {
		jsonData, _ := json.Marshal(geminiMessages)
		logging.Debug("Prepared messages", "messages", string(jsonData))
	}

	history := geminiMessages[:len(geminiMessages)-1] // All but last message
	lastMsg := geminiMessages[len(geminiMessages)-1]
	genCfg := &genai.GenerateContentConfig{
		MaxOutputTokens: int32(g.providerOptions.maxTokens),
	}

	// Set temperature (default to 1.0 if not specified)
	if g.providerOptions.temperature != nil {
		genCfg.Temperature = g.providerOptions.temperature
	} else {
		defaultTemp := float32(1.0)
		genCfg.Temperature = &defaultTemp
	}

	// Set structured output options if provided
	if g.options.responseMIMEType != "" {
		genCfg.ResponseMIMEType = g.options.responseMIMEType
	}
	if g.options.responseJSONSchema != nil {
		genCfg.ResponseJsonSchema = g.options.responseJSONSchema
	}

	// Set media resolution based on content type
	mediaRes := g.getMediaResolutionForMessages(messages)
	if mediaRes != genai.MediaResolutionUnspecified {
		genCfg.MediaResolution = mediaRes
	}

	// Only add system instruction if we have a non-empty system message
	if g.providerOptions.systemMessage != "" {
		genCfg.SystemInstruction = &genai.Content{
			Parts: []*genai.Part{{Text: g.providerOptions.systemMessage}},
			Role:  "user",
		}
	}
	if len(tools) > 0 {
		genCfg.Tools = g.convertTools(tools)
	}
	chat, _ := g.client.Chats.Create(ctx, g.providerOptions.model.APIModel, genCfg, history)

	attempts := 0
	eventChan := make(chan interfaces.ProviderEvent)

	go func() {
		defer close(eventChan)

		for {
			attempts++

			currentContent := ""
			toolCalls := []message.ToolCall{}
			var finalResp *genai.GenerateContentResponse

			eventChan <- interfaces.ProviderEvent{Type: interfaces.EventContentStart}

			var lastMsgParts []genai.Part

			for _, part := range lastMsg.Parts {
				lastMsgParts = append(lastMsgParts, *part)
			}
			for resp, err := range chat.SendMessageStream(ctx, lastMsgParts...) {
				if err != nil {
					retry, after, retryErr := g.shouldRetry(attempts, err)
					if retryErr != nil {
						eventChan <- interfaces.ProviderEvent{Type: interfaces.EventError, Error: retryErr}
						return
					}
					if retry {
						logging.Warn(fmt.Sprintf("Retrying due to rate limit... attempt %d of %d", attempts, maxRetries))
						select {
						case <-ctx.Done():
							if ctx.Err() != nil {
								eventChan <- interfaces.ProviderEvent{Type: interfaces.EventError, Error: ctx.Err()}
							}

							return
						case <-time.After(time.Duration(after) * time.Millisecond):
							break
						}
					} else {
						eventChan <- interfaces.ProviderEvent{Type: interfaces.EventError, Error: err}
						return
					}
				}

				finalResp = resp

				if len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil {
					for _, part := range resp.Candidates[0].Content.Parts {
						switch {
						case part.Text != "":
							delta := part.Text
							if delta != "" {
								eventChan <- interfaces.ProviderEvent{
									Type:    interfaces.EventContentDelta,
									Content: delta,
								}
								currentContent += delta
							}
						case part.FunctionCall != nil:
							id := "call_" + uuid.New().String()
							args, _ := json.Marshal(part.FunctionCall.Args)
							newCall := message.ToolCall{
								ID:       id,
								Name:     part.FunctionCall.Name,
								Input:    string(args),
								Type:     "function",
								Finished: true,
							}

							isNew := true
							for _, existing := range toolCalls {
								if existing.Name == newCall.Name && existing.Input == newCall.Input {
									isNew = false
									break
								}
							}

							if isNew {
								toolCalls = append(toolCalls, newCall)
							}
						}
					}
				}
			}

			eventChan <- interfaces.ProviderEvent{Type: interfaces.EventContentStop}

			if finalResp != nil {
				// Check for completely empty response (no content and no tool calls)
				if currentContent == "" && len(toolCalls) == 0 {
					logging.Warn("Gemini returned empty response with no content or tool calls")
					// Extract sessionID from context and log detailed debug information
					if sessionID, ok := ctx.Value(interfaces.SessionIDContextKey).(string); ok {
						g.logEmptyResponseDetails(sessionID, messages, tools, finalResp)
					}
				}

				finishReason := message.FinishReasonEndTurn
				if len(finalResp.Candidates) > 0 {
					finishReason = g.finishReason(finalResp.Candidates[0].FinishReason)
				}
				if len(toolCalls) > 0 {
					finishReason = message.FinishReasonToolUse
				}
				eventChan <- interfaces.ProviderEvent{
					Type: interfaces.EventComplete,
					Response: &interfaces.ProviderResponse{
						Content:      currentContent,
						ToolCalls:    toolCalls,
						Usage:        g.usage(finalResp),
						FinishReason: finishReason,
					},
				}
				return
			}
		}
	}()

	return eventChan
}

func (g *geminiClient) shouldRetry(attempts int, err error) (retry bool, retryAfterMs int64, retErr error) {
	// Check if error is a rate limit error
	if attempts > maxRetries {
		return false, 0, fmt.Errorf("maximum retry attempts reached for rate limit: %d retries", maxRetries)
	}

	// Gemini doesn't have a standard error type we can check against
	// So we'll check the error message for rate limit indicators
	if errors.Is(err, io.EOF) {
		return false, 0, err
	}

	errMsg := err.Error()

	// Check for common rate limit error messages
	isRateLimit := contains(errMsg, "rate limit", "quota exceeded", "too many requests")

	if !isRateLimit {
		return false, 0, err
	}

	// Calculate backoff with jitter
	backoffMs := 2000 * (1 << (attempts - 1))
	jitterMs := int(float64(backoffMs) * 0.2)
	retryMs := backoffMs + jitterMs

	return true, int64(retryMs), nil
}

func (g *geminiClient) usage(resp *genai.GenerateContentResponse) interfaces.TokenUsage {
	if resp == nil || resp.UsageMetadata == nil {
		return interfaces.TokenUsage{}
	}

	return interfaces.TokenUsage{
		InputTokens:         int64(resp.UsageMetadata.PromptTokenCount),
		OutputTokens:        int64(resp.UsageMetadata.CandidatesTokenCount),
		CacheCreationTokens: 0, // Not directly provided by Gemini
		CacheReadTokens:     int64(resp.UsageMetadata.CachedContentTokenCount),
	}
}

func WithGeminiDisableCache() GeminiOption {
	return func(options *geminiOptions) {
		options.disableCache = true
	}
}

// WithGeminiResponseMIMEType sets the MIME type for structured output (e.g., "application/json")
func WithGeminiResponseMIMEType(mimeType string) GeminiOption {
	return func(options *geminiOptions) {
		options.responseMIMEType = mimeType
	}
}

// WithGeminiResponseJSONSchema sets the JSON schema for structured output validation
func WithGeminiResponseJSONSchema(schema map[string]any) GeminiOption {
	return func(options *geminiOptions) {
		options.responseJSONSchema = schema
	}
}

// WithGeminiMediaResolution sets the media resolution for images/videos
// Available options: MediaResolutionLow, MediaResolutionMedium, MediaResolutionHigh
func WithGeminiMediaResolution(resolution genai.MediaResolution) GeminiOption {
	return func(options *geminiOptions) {
		options.mediaResolution = &resolution
	}
}

// Helper functions
func parseJsonToMap(jsonStr string) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := json.Unmarshal([]byte(jsonStr), &result)
	return result, err
}

func convertSchemaProperties(parameters map[string]interface{}) map[string]*genai.Schema {
	properties := make(map[string]*genai.Schema)

	for name, param := range parameters {
		properties[name] = convertToSchema(param)
	}

	return properties
}

func convertToSchema(param interface{}) *genai.Schema {
	schema := &genai.Schema{Type: genai.TypeString}

	paramMap, ok := param.(map[string]interface{})
	if !ok {
		return schema
	}

	if desc, ok := paramMap["description"].(string); ok {
		schema.Description = desc
	}

	typeVal, hasType := paramMap["type"]
	if !hasType {
		return schema
	}

	typeStr, ok := typeVal.(string)
	if !ok {
		return schema
	}

	schema.Type = mapJSONTypeToGenAI(typeStr)

	switch typeStr {
	case "array":
		schema.Items = processArrayItems(paramMap)
	case "object":
		if props, ok := paramMap["properties"].(map[string]interface{}); ok {
			schema.Properties = convertSchemaProperties(props)
		}
	}

	return schema
}

func processArrayItems(paramMap map[string]interface{}) *genai.Schema {
	items, ok := paramMap["items"].(map[string]interface{})
	if !ok {
		return nil
	}

	return convertToSchema(items)
}

func mapJSONTypeToGenAI(jsonType string) genai.Type {
	switch jsonType {
	case "string":
		return genai.TypeString
	case "number":
		return genai.TypeNumber
	case "integer":
		return genai.TypeInteger
	case "boolean":
		return genai.TypeBoolean
	case "array":
		return genai.TypeArray
	case "object":
		return genai.TypeObject
	default:
		return genai.TypeString // Default to string for unknown types
	}
}

func contains(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if strings.Contains(strings.ToLower(s), strings.ToLower(substr)) {
			return true
		}
	}
	return false
}

// logEmptyResponseDetails logs detailed request and response information when Gemini returns empty responses
func (g *geminiClient) logEmptyResponseDetails(sessionID string, messages []message.Message, tools []interfaces.BaseTool, resp *genai.GenerateContentResponse) {
	timestamp := time.Now().Format("20060102-150405")

	// Create log directory if it doesn't exist
	logDir := "debug_logs"
	if err := os.MkdirAll(logDir, 0o750); err != nil {
		logging.Warn("Failed to create debug log directory", "error", err)
		return
	}

	// Log request details
	requestFile := filepath.Join(logDir, fmt.Sprintf("gemini-empty-response-%s-%s-request.txt", sessionID, timestamp))
	requestData := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"sessionID": sessionID,
		"messages":  messages,
		"tools": func() interface{} {
			if len(tools) > 0 {
				return g.convertTools(tools)
			}
			return []string{}
		}(),
		"systemMessage": g.providerOptions.systemMessage,
		"model":         g.providerOptions.model,
		"maxTokens":     g.providerOptions.maxTokens,
	}

	requestJSON, _ := json.MarshalIndent(requestData, "", "  ")
	if err := os.WriteFile(requestFile, requestJSON, 0o600); err != nil {
		logging.Warn("Failed to write debug request file", "error", err)
	}

	// Log response details
	responseFile := filepath.Join(logDir, fmt.Sprintf("gemini-empty-response-%s-%s-response.txt", sessionID, timestamp))
	responseData := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"sessionID": sessionID,
		"response":  resp,
		"candidatesCount": func() int {
			if resp != nil && resp.Candidates != nil {
				return len(resp.Candidates)
			}
			return 0
		}(),
		"firstCandidateContent": func() interface{} {
			if resp != nil && resp.Candidates != nil && len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil {
				return resp.Candidates[0].Content
			}
			return nil
		}(),
	}

	responseJSON, _ := json.MarshalIndent(responseData, "", "  ")
	if err := os.WriteFile(responseFile, responseJSON, 0o600); err != nil {
		logging.Warn("Failed to write debug response file", "error", err)
	}

	logging.Info("Empty response debug files created", "requestFile", requestFile, "responseFile", responseFile)
}

// isSupportedInlineFormat checks if the MIME type is supported for inline data
func (g *geminiClient) isSupportedInlineFormat(mimeType string) bool {
	// Supported inline formats according to Gemini API docs
	supportedInlineFormats := []string{
		"image/png",
		"image/jpeg",
		"image/webp",
		"image/heic",
		"image/heif",
		"application/pdf",
	}

	for _, supported := range supportedInlineFormats {
		if mimeType == supported {
			return true
		}
	}
	return false
}

// isSupportedVideoFormat checks if the MIME type is a supported video format
// Note: Video support would require File API implementation in future SDK versions
func (g *geminiClient) isSupportedVideoFormat(mimeType string) bool {
	supportedVideoFormats := []string{
		"video/mp4",
		"video/mpeg",
		"video/mov",
		"video/avi",
		"video/x-flv",
		"video/mpg",
		"video/webm",
		"video/wmv",
		"video/3gpp",
	}

	for _, supported := range supportedVideoFormats {
		if mimeType == supported {
			return true
		}
	}
	return false
}

// determineMediaResolution returns the appropriate MediaResolution based on MIME type
// Follows Gemini 3 best practices:
// - Images: HIGH (1120 tokens) - recommended for most image analysis tasks
// - PDFs: MEDIUM (560 tokens) - optimal for document understanding
// - Videos: LOW (70 tokens/frame) - sufficient for action recognition
func (g *geminiClient) determineMediaResolution(mimeType string) genai.MediaResolution {
	// If explicitly set via options, use that
	if g.options.mediaResolution != nil {
		return *g.options.mediaResolution
	}

	// Determine based on MIME type
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return genai.MediaResolutionHigh
	case mimeType == "application/pdf":
		return genai.MediaResolutionMedium
	case strings.HasPrefix(mimeType, "video/"):
		return genai.MediaResolutionLow
	default:
		// For unknown types, use medium as a safe default
		return genai.MediaResolutionMedium
	}
}

// getMediaResolutionForMessages determines the appropriate media resolution for the conversation
// Scans all messages for media content and returns the highest resolution needed
func (g *geminiClient) getMediaResolutionForMessages(messages []message.Message) genai.MediaResolution {
	// If explicitly set via options, use that for all media
	if g.options.mediaResolution != nil {
		return *g.options.mediaResolution
	}

	// Scan messages for media content
	highestResolution := genai.MediaResolutionUnspecified

	for _, msg := range messages {
		// Check binary content
		for _, binaryContent := range msg.BinaryContent() {
			resolution := g.determineMediaResolution(binaryContent.MIMEType)
			if shouldUpgradeResolution(highestResolution, resolution) {
				highestResolution = resolution
			}
		}

		// Check URI content
		for _, uriContent := range msg.URIContent() {
			resolution := g.determineMediaResolution(uriContent.MIMEType)
			if shouldUpgradeResolution(highestResolution, resolution) {
				highestResolution = resolution
			}
		}
	}

	// Return the highest resolution needed, or unspecified if no media
	return highestResolution
}

// shouldUpgradeResolution determines if we should upgrade to a higher resolution
// Resolution priority: HIGH > MEDIUM > LOW > UNSPECIFIED
func shouldUpgradeResolution(current, target genai.MediaResolution) bool {
	priority := map[genai.MediaResolution]int{
		genai.MediaResolutionUnspecified: 0,
		genai.MediaResolutionLow:         1,
		genai.MediaResolutionMedium:      2,
		genai.MediaResolutionHigh:        3,
	}

	return priority[target] > priority[current]
}
