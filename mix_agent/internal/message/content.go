package message

import (
	"encoding/base64"
	"fmt"
	"slices"
	"strings"
	"time"

	"mix/internal/llm/models"
)

type MessageRole string

const (
	Assistant MessageRole = "assistant"
	User      MessageRole = "user"
	System    MessageRole = "system"
	Tool      MessageRole = "tool"
)

type FinishReason string

const (
	FinishReasonEndTurn          FinishReason = "end_turn"
	FinishReasonMaxTokens        FinishReason = "max_tokens"
	FinishReasonToolUse          FinishReason = "tool_use"
	FinishReasonCanceled         FinishReason = "canceled"
	FinishReasonError            FinishReason = "error"
	FinishReasonPermissionDenied FinishReason = "permission_denied"

	// Should never happen
	FinishReasonUnknown FinishReason = "unknown"
)

type ContentPart interface {
	isPart()
}

type ReasoningContent struct {
	Thinking string `json:"thinking"`
	Duration int64  `json:"duration"` // Duration in seconds
}

func (tc ReasoningContent) String() string {
	return tc.Thinking
}
func (ReasoningContent) isPart() {}

// ThinkingBlockContent represents thinking blocks from Anthropic API
// Contains the actual thinking content and signature for verification
type ThinkingBlockContent struct {
	Thinking  string `json:"thinking"`
	Signature string `json:"signature"` // Required for API verification
}

func (tbc ThinkingBlockContent) String() string {
	return tbc.Thinking
}
func (ThinkingBlockContent) isPart() {}

// RedactedThinkingContent represents redacted/encrypted thinking blocks
// Contains encrypted thinking data when safety systems flag content
type RedactedThinkingContent struct {
	Data string `json:"data"` // Encrypted thinking data
}

func (rtc RedactedThinkingContent) String() string {
	return "[Redacted Thinking]"
}
func (RedactedThinkingContent) isPart() {}

type TextContent struct {
	Text string `json:"text"`
}

func (tc TextContent) String() string {
	return tc.Text
}

func (TextContent) isPart() {}

type ImageURLContent struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"`
}

func (iuc ImageURLContent) String() string {
	return iuc.URL
}

func (ImageURLContent) isPart() {}

type BinaryContent struct {
	Path        string
	MIMEType    string
	Data        []byte
	StartOffset string // Video metadata: start time offset (e.g., "1250s")
	EndOffset   string // Video metadata: end time offset (e.g., "1570s")
}

func (bc BinaryContent) String(provider models.ModelProvider) string {
	base64Encoded := base64.StdEncoding.EncodeToString(bc.Data)
	if provider == models.ProviderOpenAI {
		return "data:" + bc.MIMEType + ";base64," + base64Encoded
	}
	return base64Encoded
}

func (BinaryContent) isPart() {}

type URIContent struct {
	URI         string `json:"uri"`
	MIMEType    string `json:"mime_type"`
	StartOffset string `json:"start_offset,omitempty"` // Video metadata: start time offset (e.g., "1250s")
	EndOffset   string `json:"end_offset,omitempty"`   // Video metadata: end time offset (e.g., "1570s")
}

func (uc URIContent) String() string {
	return uc.URI
}

func (URIContent) isPart() {}

type ToolCall struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Input    string `json:"input"`
	Type     string `json:"type"`
	Finished bool   `json:"finished"`
}

func (ToolCall) isPart() {}

type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Name       string `json:"name"`
	Content    string `json:"content"`
	Metadata   string `json:"metadata"`
	IsError    bool   `json:"is_error"`
}

func (ToolResult) isPart() {}

type CallbackResult struct {
	ToolCallID         string `json:"tool_call_id"`                   // Links back to the tool call that triggered this callback
	ToolName           string `json:"tool_name"`                      // Name of the tool that triggered callback
	CallbackName       string `json:"callback_name,omitempty"`        // Human-readable name of the callback
	CallbackType       string `json:"callback_type"`                  // "bash_script", "sub_agent", or "send_message"
	Stdout             string `json:"stdout,omitempty"`               // For bash callbacks: stdout output
	Stderr             string `json:"stderr,omitempty"`               // For bash callbacks: stderr output
	ExitCode           int    `json:"exit_code"`                      // For bash callbacks: exit code
	SubAgentID         string `json:"subagent_id,omitempty"`          // For subagent callbacks: ID of spawned session
	SubAgentResult     string `json:"subagent_result,omitempty"`      // For subagent/send_message callbacks: result summary
	Success            bool   `json:"success"`                        // Whether callback succeeded
	Error              string `json:"error,omitempty"`                // Error message if failed
	ExcludeFromContext bool   `json:"exclude_from_context,omitempty"` // Whether to exclude this callback result from agent context
}

func (CallbackResult) isPart() {}

type ThinkingBlock struct {
	Thinking  string `json:"thinking"`
	Signature string `json:"signature"`
}

func (ThinkingBlock) isPart() {}

type Finish struct {
	Reason FinishReason `json:"reason"`
	Time   int64        `json:"time"`
}

func (Finish) isPart() {}

type Message struct {
	ID        string
	Role      MessageRole
	SessionID string
	Parts     []ContentPart
	Model     models.ModelID
	CreatedAt int64
	UpdatedAt int64
}

func (m *Message) Content() TextContent {
	for _, part := range m.Parts {
		if c, ok := part.(TextContent); ok {
			return c
		}
	}
	return TextContent{}
}

func (m *Message) ReasoningContent() ReasoningContent {
	for _, part := range m.Parts {
		if c, ok := part.(ReasoningContent); ok {
			return c
		}
	}
	return ReasoningContent{}
}

// ThinkingBlocks returns all thinking blocks from the message
func (m *Message) ThinkingBlocks() []ThinkingBlockContent {
	var blocks []ThinkingBlockContent
	for _, part := range m.Parts {
		if c, ok := part.(ThinkingBlockContent); ok {
			blocks = append(blocks, c)
		}
	}
	return blocks
}

// RedactedThinkingBlocks returns all redacted thinking blocks from the message
func (m *Message) RedactedThinkingBlocks() []RedactedThinkingContent {
	var blocks []RedactedThinkingContent
	for _, part := range m.Parts {
		if c, ok := part.(RedactedThinkingContent); ok {
			blocks = append(blocks, c)
		}
	}
	return blocks
}

// HasThinkingBlocks checks if message contains any thinking blocks
func (m *Message) HasThinkingBlocks() bool {
	for _, part := range m.Parts {
		if _, ok := part.(ThinkingBlockContent); ok {
			return true
		}
		if _, ok := part.(RedactedThinkingContent); ok {
			return true
		}
	}
	return false
}

func (m *Message) ImageURLContent() []ImageURLContent {
	imageURLContents := make([]ImageURLContent, 0)
	for _, part := range m.Parts {
		if c, ok := part.(ImageURLContent); ok {
			imageURLContents = append(imageURLContents, c)
		}
	}
	return imageURLContents
}

func (m *Message) BinaryContent() []BinaryContent {
	binaryContents := make([]BinaryContent, 0)
	for _, part := range m.Parts {
		if c, ok := part.(BinaryContent); ok {
			binaryContents = append(binaryContents, c)
		}
	}
	return binaryContents
}

func (m *Message) URIContent() []URIContent {
	uriContents := make([]URIContent, 0)
	for _, part := range m.Parts {
		if c, ok := part.(URIContent); ok {
			uriContents = append(uriContents, c)
		}
	}
	return uriContents
}

func (m *Message) ToolCalls() []ToolCall {
	toolCalls := make([]ToolCall, 0)
	for _, part := range m.Parts {
		if c, ok := part.(ToolCall); ok {
			toolCalls = append(toolCalls, c)
		}
	}
	return toolCalls
}

func (m *Message) ToolResults() []ToolResult {
	toolResults := make([]ToolResult, 0)
	for _, part := range m.Parts {
		if c, ok := part.(ToolResult); ok {
			toolResults = append(toolResults, c)
		}
	}
	return toolResults
}

func (m *Message) CallbackResults() []CallbackResult {
	callbackResults := make([]CallbackResult, 0)
	for _, part := range m.Parts {
		if c, ok := part.(CallbackResult); ok {
			callbackResults = append(callbackResults, c)
		}
	}
	return callbackResults
}

func (m *Message) AddCallbackResult(cr CallbackResult) {
	m.Parts = append(m.Parts, cr)
}

func (m *Message) IsFinished() bool {
	for _, part := range m.Parts {
		if _, ok := part.(Finish); ok {
			return true
		}
	}
	return false
}

func (m *Message) FinishPart() *Finish {
	for _, part := range m.Parts {
		if c, ok := part.(Finish); ok {
			return &c
		}
	}
	return nil
}

func (m *Message) FinishReason() FinishReason {
	for _, part := range m.Parts {
		if c, ok := part.(Finish); ok {
			return c.Reason
		}
	}
	return ""
}

// RateLimitInfo contains information about rate limits and retry attempts
type RateLimitInfo struct {
	RetryAfter  int
	Attempt     int
	MaxAttempts int
}

// RateLimitInfo parses error messages for rate limit information
func (m *Message) RateLimitInfo() *RateLimitInfo {
	// Check if this is a rate limit error first
	if m.FinishReason() != "error" {
		return nil
	}

	errMsg := m.Content().Text
	if !strings.Contains(errMsg, "rate_limit_error") && !strings.Contains(errMsg, "rate limit") {
		return nil
	}

	// Default values
	retryInfo := &RateLimitInfo{
		RetryAfter:  60, // Default retry after 60 seconds
		Attempt:     1,  // Default current attempt
		MaxAttempts: 8,  // Default max attempts
	}

	// Try to extract retry attempt information from the message
	if strings.Contains(errMsg, "Retrying due to rate limit") {
		// Try to parse attempt numbers like "attempt 1 of 8"
		// If parsing fails, we just use the default values
		_, _ = fmt.Sscanf(errMsg, "Retrying due to rate limit... attempt %d of %d", &retryInfo.Attempt, &retryInfo.MaxAttempts)
	}

	return retryInfo
}

func (m *Message) IsThinking() bool {
	if m.ReasoningContent().Thinking != "" && m.Content().Text == "" && !m.IsFinished() {
		return true
	}
	return false
}

func (m *Message) AppendContent(delta string) {
	found := false
	for i, part := range m.Parts {
		if c, ok := part.(TextContent); ok {
			m.Parts[i] = TextContent{Text: c.Text + delta}
			found = true
		}
	}
	if !found {
		m.Parts = append(m.Parts, TextContent{Text: delta})
	}
}

func (m *Message) AppendReasoningContent(delta string) {
	found := false
	for i, part := range m.Parts {
		if c, ok := part.(ReasoningContent); ok {
			m.Parts[i] = ReasoningContent{Thinking: c.Thinking + delta, Duration: c.Duration}
			found = true
		}
	}
	if !found {
		m.Parts = append(m.Parts, ReasoningContent{Thinking: delta, Duration: 0})
	}
}

func (m *Message) SetReasoningDuration(duration int64) {
	for i, part := range m.Parts {
		if c, ok := part.(ReasoningContent); ok {
			m.Parts[i] = ReasoningContent{Thinking: c.Thinking, Duration: duration}
			return
		}
	}
}

func (m *Message) FinishToolCall(toolCallID string) {
	for i, part := range m.Parts {
		if c, ok := part.(ToolCall); ok {
			if c.ID == toolCallID {
				m.Parts[i] = ToolCall{
					ID:       c.ID,
					Name:     c.Name,
					Input:    c.Input,
					Type:     c.Type,
					Finished: true,
				}
				return
			}
		}
	}
}

func (m *Message) AppendToolCallInput(toolCallID string, inputDelta string) error {
	for i, part := range m.Parts {
		if c, ok := part.(ToolCall); ok {
			if c.ID == toolCallID {
				m.Parts[i] = ToolCall{
					ID:       c.ID,
					Name:     c.Name,
					Input:    c.Input + inputDelta,
					Type:     c.Type,
					Finished: c.Finished,
				}
				return nil
			}
		}
	}
	return fmt.Errorf("tool call with ID %s not found in message", toolCallID)
}

func (m *Message) AddToolCall(tc ToolCall) {
	for i, part := range m.Parts {
		if c, ok := part.(ToolCall); ok {
			if c.ID == tc.ID {
				m.Parts[i] = tc
				return
			}
		}
	}
	m.Parts = append(m.Parts, tc)
}

func (m *Message) SetToolCalls(tc []ToolCall) {
	// remove any existing tool call part it could have multiple
	parts := make([]ContentPart, 0)
	for _, part := range m.Parts {
		if _, ok := part.(ToolCall); ok {
			continue
		}
		parts = append(parts, part)
	}
	m.Parts = parts
	for _, toolCall := range tc {
		m.Parts = append(m.Parts, toolCall)
	}
}

func (m *Message) AddToolResult(tr ToolResult) {
	m.Parts = append(m.Parts, tr)
}

func (m *Message) SetToolResults(tr []ToolResult) {
	for _, toolResult := range tr {
		m.Parts = append(m.Parts, toolResult)
	}
}

func (m *Message) AddFinish(reason FinishReason) {
	// remove any existing finish part
	for i, part := range m.Parts {
		if _, ok := part.(Finish); ok {
			m.Parts = slices.Delete(m.Parts, i, i+1)
			break
		}
	}
	m.Parts = append(m.Parts, Finish{Reason: reason, Time: time.Now().Unix()})
}

func (m *Message) AddImageURL(url, detail string) {
	m.Parts = append(m.Parts, ImageURLContent{URL: url, Detail: detail})
}

func (m *Message) AddBinary(mimeType string, data []byte) {
	m.Parts = append(m.Parts, BinaryContent{MIMEType: mimeType, Data: data})
}

// AddThinkingBlock adds a thinking block to the message
func (m *Message) AddThinkingBlock(thinking, signature string) {
	m.Parts = append(m.Parts, ThinkingBlockContent{
		Thinking:  thinking,
		Signature: signature,
	})
}

// AddRedactedThinkingBlock adds a redacted thinking block to the message
func (m *Message) AddRedactedThinkingBlock(data string) {
	m.Parts = append(m.Parts, RedactedThinkingContent{
		Data: data,
	})
}
