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
	toolspkg "mix/internal/llm/tools"
	"mix/internal/logging"
	"mix/internal/message"

	"github.com/google/uuid"
	"google.golang.org/genai"
)

type geminiOptions struct {
	disableCache bool
}

type GeminiOption func(*geminiOptions)

type geminiClient struct {
	providerOptions providerClientOptions
	options         geminiOptions
	client          *genai.Client
}

type GeminiClient ProviderClient

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
				imageFormat := strings.Split(binaryContent.MIMEType, "/")
				parts = append(parts, &genai.Part{InlineData: &genai.Blob{
					MIMEType: imageFormat[1],
					Data:     binaryContent.Data,
				}})
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

func (g *geminiClient) convertTools(tools []toolspkg.BaseTool) []*genai.Tool {
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
	switch {
	case reason == genai.FinishReasonStop:
		return message.FinishReasonEndTurn
	case reason == genai.FinishReasonMaxTokens:
		return message.FinishReasonMaxTokens
	default:
		return message.FinishReasonUnknown
	}
}

func (g *geminiClient) send(ctx context.Context, messages []message.Message, tools []toolspkg.BaseTool) (*ProviderResponse, error) {
	// Convert messages
	geminiMessages := g.convertMessages(messages)

	cfg := config.Get()
	if cfg.Debug {
		jsonData, _ := json.Marshal(geminiMessages)
		logging.Debug("Prepared messages", "messages", string(jsonData))
	}

	history := geminiMessages[:len(geminiMessages)-1] // All but last message
	lastMsg := geminiMessages[len(geminiMessages)-1]
	config := &genai.GenerateContentConfig{
		MaxOutputTokens: int32(g.providerOptions.maxTokens),
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{{Text: g.providerOptions.systemMessage}},
		},
	}
	if len(tools) > 0 {
		config.Tools = g.convertTools(tools)
	}
	chat, _ := g.client.Chats.Create(ctx, g.providerOptions.model.APIModel, config, history)

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
					content = string(part.Text)
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
			if sessionID, ok := ctx.Value(toolspkg.SessionIDContextKey).(string); ok {
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

		return &ProviderResponse{
			Content:      content,
			ToolCalls:    toolCalls,
			Usage:        g.usage(resp),
			FinishReason: finishReason,
		}, nil
	}
}

func (g *geminiClient) stream(ctx context.Context, messages []message.Message, tools []toolspkg.BaseTool) <-chan ProviderEvent {
	// Convert messages
	geminiMessages := g.convertMessages(messages)

	cfg := config.Get()
	if cfg.Debug {
		jsonData, _ := json.Marshal(geminiMessages)
		logging.Debug("Prepared messages", "messages", string(jsonData))
	}

	history := geminiMessages[:len(geminiMessages)-1] // All but last message
	lastMsg := geminiMessages[len(geminiMessages)-1]
	config := &genai.GenerateContentConfig{
		MaxOutputTokens: int32(g.providerOptions.maxTokens),
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{{Text: g.providerOptions.systemMessage}},
		},
	}
	if len(tools) > 0 {
		config.Tools = g.convertTools(tools)
	}
	chat, _ := g.client.Chats.Create(ctx, g.providerOptions.model.APIModel, config, history)

	attempts := 0
	eventChan := make(chan ProviderEvent)

	go func() {
		defer close(eventChan)

		for {
			attempts++

			currentContent := ""
			toolCalls := []message.ToolCall{}
			var finalResp *genai.GenerateContentResponse

			eventChan <- ProviderEvent{Type: EventContentStart}

			var lastMsgParts []genai.Part

			for _, part := range lastMsg.Parts {
				lastMsgParts = append(lastMsgParts, *part)
			}
			for resp, err := range chat.SendMessageStream(ctx, lastMsgParts...) {
				if err != nil {
					retry, after, retryErr := g.shouldRetry(attempts, err)
					if retryErr != nil {
						eventChan <- ProviderEvent{Type: EventError, Error: retryErr}
						return
					}
					if retry {
						logging.Warn(fmt.Sprintf("Retrying due to rate limit... attempt %d of %d", attempts, maxRetries))
						select {
						case <-ctx.Done():
							if ctx.Err() != nil {
								eventChan <- ProviderEvent{Type: EventError, Error: ctx.Err()}
							}

							return
						case <-time.After(time.Duration(after) * time.Millisecond):
							break
						}
					} else {
						eventChan <- ProviderEvent{Type: EventError, Error: err}
						return
					}
				}

				finalResp = resp

				if len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil {
					for _, part := range resp.Candidates[0].Content.Parts {
						switch {
						case part.Text != "":
							delta := string(part.Text)
							if delta != "" {
								eventChan <- ProviderEvent{
									Type:    EventContentDelta,
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

			eventChan <- ProviderEvent{Type: EventContentStop}

			if finalResp != nil {
				// Check for completely empty response (no content and no tool calls)
				if currentContent == "" && len(toolCalls) == 0 {
					logging.Warn("Gemini returned empty response with no content or tool calls")
					// Extract sessionID from context and log detailed debug information
					if sessionID, ok := ctx.Value(toolspkg.SessionIDContextKey).(string); ok {
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
				eventChan <- ProviderEvent{
					Type: EventComplete,
					Response: &ProviderResponse{
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

func (g *geminiClient) shouldRetry(attempts int, err error) (bool, int64, error) {
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
	isRateLimit := false

	// Check for common rate limit error messages
	if contains(errMsg, "rate limit", "quota exceeded", "too many requests") {
		isRateLimit = true
	}

	if !isRateLimit {
		return false, 0, err
	}

	// Calculate backoff with jitter
	backoffMs := 2000 * (1 << (attempts - 1))
	jitterMs := int(float64(backoffMs) * 0.2)
	retryMs := backoffMs + jitterMs

	return true, int64(retryMs), nil
}

func (g *geminiClient) toolCalls(resp *genai.GenerateContentResponse) []message.ToolCall {
	var toolCalls []message.ToolCall

	if len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil {
		for _, part := range resp.Candidates[0].Content.Parts {
			if part.FunctionCall != nil {
				id := "call_" + uuid.New().String()
				args, _ := json.Marshal(part.FunctionCall.Args)
				toolCalls = append(toolCalls, message.ToolCall{
					ID:    id,
					Name:  part.FunctionCall.Name,
					Input: string(args),
					Type:  "function",
				})
			}
		}
	}

	return toolCalls
}

func (g *geminiClient) usage(resp *genai.GenerateContentResponse) TokenUsage {
	if resp == nil || resp.UsageMetadata == nil {
		return TokenUsage{}
	}

	return TokenUsage{
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
func (g *geminiClient) logEmptyResponseDetails(sessionID string, messages []message.Message, tools []toolspkg.BaseTool, resp *genai.GenerateContentResponse) {
	timestamp := time.Now().Format("20060102-150405")

	// Create log directory if it doesn't exist
	logDir := "debug_logs"
	os.MkdirAll(logDir, 0755)

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
	os.WriteFile(requestFile, requestJSON, 0644)

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
	os.WriteFile(responseFile, responseJSON, 0644)

	logging.Info("Empty response debug files created", "requestFile", requestFile, "responseFile", responseFile)
}
