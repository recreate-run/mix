package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"mix/internal/config"
	"mix/internal/llm/models"
	"mix/internal/llm/prompt"
	"mix/internal/llm/provider"
	"mix/internal/llm/tools"
	"mix/internal/logging"
	"mix/internal/message"
	"mix/internal/permission"
	"mix/internal/pubsub"
	"mix/internal/session"
)

// Common errors
var (
	ErrRequestCancelled = errors.New("request cancelled by user")
	ErrSessionBusy      = errors.New("session is currently processing another request")
)

type AgentEventType string

const (
	AgentEventTypeError     AgentEventType = "error"
	AgentEventTypeResponse  AgentEventType = "response"
	AgentEventTypeSummarize AgentEventType = "summarize"
)

type AgentEvent struct {
	Type    AgentEventType
	Message message.Message
	Error   error

	// When summarizing
	SessionID string
	Progress  string
	Done      bool
}

type Service interface {
	pubsub.Suscriber[AgentEvent]
	Model() models.Model
	Run(ctx context.Context, sessionID string, content string, attachments ...message.Attachment) (<-chan AgentEvent, error)
	RunWithPlanMode(ctx context.Context, sessionID string, content string, planMode bool, attachments ...message.Attachment) (<-chan AgentEvent, error)
	Cancel(sessionID string)
	IsSessionBusy(sessionID string) bool
	IsBusy() bool
	Update(agentName config.AgentName, modelID models.ModelID) (models.Model, error)
	Summarize(ctx context.Context, sessionID string) error
	Shutdown()
}

type agent struct {
	*pubsub.Broker[AgentEvent]
	sessions session.Service
	messages message.Service

	agentName config.AgentName
	tools     []tools.BaseTool
	provider  provider.Provider

	titleProvider     provider.Provider
	summarizeProvider provider.Provider

	sessionProviders sync.Map // Maps session ID to provider.Provider
	activeRequests   sync.Map

	ctx    context.Context
	cancel context.CancelFunc
}

func NewAgent(
	agentName config.AgentName,
	sessions session.Service,
	messages message.Service,
	agentTools []tools.BaseTool,
) (Service, error) {
	agentProvider, err := createAgentProvider(agentName)
	if err != nil {
		return nil, err
	}
	var titleProvider provider.Provider
	// Only generate titles for the main agent
	if agentName == config.AgentMain {
		titleProvider, err = createAgentProvider(config.AgentMain)
		if err != nil {
			return nil, err
		}
	}
	var summarizeProvider provider.Provider
	if agentName == config.AgentMain {
		summarizeProvider, err = createAgentProvider(config.AgentMain)
		if err != nil {
			return nil, err
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	agent := &agent{
		Broker:            pubsub.NewBroker[AgentEvent](),
		agentName:         agentName,
		provider:          agentProvider,
		messages:          messages,
		sessions:          sessions,
		tools:             agentTools,
		titleProvider:     titleProvider,
		summarizeProvider: summarizeProvider,
		sessionProviders:  sync.Map{},
		activeRequests:    sync.Map{},
		ctx:               ctx,
		cancel:            cancel,
	}

	// Start session deletion cleanup goroutine
	go agent.handleSessionEvents()

	return agent, nil
}

func (a *agent) Model() models.Model {
	return a.provider.Model()
}

func (a *agent) Cancel(sessionID string) {
	// Cancel regular requests
	if cancelFunc, exists := a.activeRequests.LoadAndDelete(sessionID); exists {
		if cancel, ok := cancelFunc.(context.CancelFunc); ok {
			logging.Info("Request cancellation initiated for session", "sessionID", sessionID)
			cancel()
		}
	}

	// Also check for summarize requests
	if cancelFunc, exists := a.activeRequests.LoadAndDelete(sessionID + "-summarize"); exists {
		if cancel, ok := cancelFunc.(context.CancelFunc); ok {
			logging.Info("Summarize cancellation initiated for session", "sessionID", sessionID)
			cancel()
		}
	}
}

func (a *agent) IsBusy() bool {
	busy := false
	a.activeRequests.Range(func(key, value interface{}) bool {
		if cancelFunc, ok := value.(context.CancelFunc); ok {
			if cancelFunc != nil {
				busy = true
				return false // Stop iterating
			}
		}
		return true // Continue iterating
	})
	return busy
}

func (a *agent) IsSessionBusy(sessionID string) bool {
	_, busy := a.activeRequests.Load(sessionID)
	return busy
}

func (a *agent) generateTitle(ctx context.Context, sessionID string, content string) error {
	if content == "" {
		return nil
	}
	if a.titleProvider == nil {
		return nil
	}
	session, err := a.sessions.Get(ctx, sessionID)
	if err != nil {
		return err
	}
	ctx = context.WithValue(ctx, tools.SessionIDContextKey, sessionID)
	
	// Add session working directory to context
	ctx = context.WithValue(ctx, tools.WorkingDirectoryContextKey, session.WorkingDirectory)
	
	parts := []message.ContentPart{message.TextContent{Text: content}}
	response, err := a.titleProvider.SendMessages(
		ctx,
		[]message.Message{
			{
				Role:  message.User,
				Parts: parts,
			},
		},
		make([]tools.BaseTool, 0),
	)
	if err != nil {
		return err
	}

	title := strings.TrimSpace(strings.ReplaceAll(response.Content, "\n", " "))
	if title == "" {
		return nil
	}

	session.Title = title
	_, err = a.sessions.Save(ctx, session)
	return err
}

func (a *agent) err(err error) AgentEvent {
	return AgentEvent{
		Type:  AgentEventTypeError,
		Error: err,
	}
}

func (a *agent) Run(ctx context.Context, sessionID string, content string, attachments ...message.Attachment) (<-chan AgentEvent, error) {
	return a.RunWithPlanMode(ctx, sessionID, content, false, attachments...)
}

func (a *agent) RunWithPlanMode(ctx context.Context, sessionID string, content string, planMode bool, attachments ...message.Attachment) (<-chan AgentEvent, error) {
	if !a.provider.Model().SupportsAttachments && attachments != nil {
		attachments = nil
	}
	events := make(chan AgentEvent, 10) // Buffered channel for better streaming

	genCtx, cancel := context.WithCancel(ctx)
	if _, loaded := a.activeRequests.LoadOrStore(sessionID, cancel); loaded {
		cancel() // Clean up unused cancel function
		return nil, ErrSessionBusy
	}

	// Add plan mode to context
	if planMode {
		genCtx = context.WithValue(genCtx, "plan_mode", true)
	}

	// Subscribe to agent events for real-time streaming
	subscription := a.Subscribe(genCtx)

	go func() {
		defer func() {
			logging.Debug("Request completed", "sessionID", sessionID)
			a.activeRequests.Delete(sessionID)
			cancel()
			close(events)
		}()

		logging.Debug("Request started", "sessionID", sessionID, "planMode", planMode)
		defer logging.RecoverPanic("agent.Run", func() {
			events <- a.err(fmt.Errorf("panic while running the agent"))
		})

		var attachmentParts []message.ContentPart
		for _, attachment := range attachments {
			attachmentParts = append(attachmentParts, message.BinaryContent{Path: attachment.FilePath, MIMEType: attachment.MimeType, Data: attachment.Content})
		}

		result := a.processGeneration(genCtx, sessionID, content, attachmentParts)
		if result.Error != nil && !errors.Is(result.Error, ErrRequestCancelled) && !errors.Is(result.Error, context.Canceled) {
			logging.Error(result.Error.Error())
		}
		// Always send the final result directly to ensure CLI mode receives it
		events <- result
	}()

	// Forward intermediate events from subscription to the events channel
	go func() {
		defer logging.RecoverPanic("agent.Run-subscription", nil)
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-subscription:
				if !ok {
					return
				}
				// Only forward intermediate events for this specific session (not final completion events)
				if (event.Payload.SessionID == sessionID || event.Payload.Message.SessionID == sessionID) && !event.Payload.Done {
					select {
					case events <- event.Payload:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	return events, nil
}

func (a *agent) processGeneration(ctx context.Context, sessionID, content string, attachmentParts []message.ContentPart) AgentEvent {
	logging.Info("[Agent] Starting message processing for session", "sessionID", sessionID, "contentPreview", fmt.Sprintf("%.100s...", content))
	_ = config.Get()
	// List existing messages; if none, start title generation asynchronously.
	msgs, err := a.messages.List(ctx, sessionID)
	if err != nil {
		return a.err(fmt.Errorf("failed to list messages: %w", err))
	}
	if len(msgs) == 0 {
		go func() {
			defer logging.RecoverPanic("agent.Run", func() {
				logging.Error("panic while generating title")
			})
			titleErr := a.generateTitle(context.Background(), sessionID, content)
			if titleErr != nil {
				logging.Error(fmt.Sprintf("failed to generate title: %v", titleErr))
			}
		}()
	}
	session, err := a.sessions.Get(ctx, sessionID)
	if err != nil {
		return a.err(fmt.Errorf("failed to get session: %w", err))
	}
	if session.SummaryMessageID != "" {
		summaryMsgInex := -1
		for i, msg := range msgs {
			if msg.ID == session.SummaryMessageID {
				summaryMsgInex = i
				break
			}
		}
		if summaryMsgInex != -1 {
			msgs = msgs[summaryMsgInex:]
			msgs[0].Role = message.User
		}
	}

	userMsg, err := a.createUserMessage(ctx, sessionID, content, attachmentParts)
	if err != nil {
		return a.err(fmt.Errorf("failed to create user message: %w", err))
	}
	// Append the new user message to the conversation history.
	msgHistory := append(msgs, userMsg)

	for {
		// Check for cancellation before each iteration
		select {
		case <-ctx.Done():
			return a.err(ctx.Err())
		default:
			// Continue processing
		}
		agentMessage, toolResults, err := a.streamAndHandleEvents(ctx, sessionID, msgHistory)
		if err != nil {
			logging.Info("[Agent] Stream processing failed for session", "sessionID", sessionID, "error", err)
			if errors.Is(err, context.Canceled) {
				agentMessage.AddFinish(message.FinishReasonCanceled)
				a.messages.Update(context.Background(), agentMessage)
				return a.err(ErrRequestCancelled)
			}
			return a.err(fmt.Errorf("failed to process events: %w", err))
		}

		// Enhanced tool results logging for debugging
		if toolResults != nil {
			for i, result := range toolResults.ToolCalls() {
				logging.Info("[Agent] Detailed tool result", "sessionID", sessionID, "toolIndex", i, "toolCallID", result.ID, "toolName", result.Name, "inputLength", len(result.Input), "input", result.Input)
			}
		}
		if (agentMessage.FinishReason() == message.FinishReasonToolUse) && toolResults != nil {
			// We are not done, we need to respond with the tool response
			msgHistory = append(msgHistory, agentMessage, *toolResults)
			continue
		}
		// Publish final completion event

		finalEvent := AgentEvent{
			Type:      AgentEventTypeResponse,
			Message:   agentMessage,
			SessionID: sessionID,
			Done:      true,
		}
		err = a.Publish(ctx, pubsub.CreatedEvent, finalEvent)
		if err != nil {
			return a.err(err)
		}
		return finalEvent
	}
}

func (a *agent) createUserMessage(ctx context.Context, sessionID, content string, attachmentParts []message.ContentPart) (message.Message, error) {
	// Check if plan mode is active and append system-reminder
	messageContent := content
	if ctx.Value("plan_mode") != nil {
		planModeContent := prompt.LoadPrompt("plan_mode")
		messageContent = content + "\n\n<system-reminder>\n" + planModeContent + "\n</system-reminder>"
	}
	
	parts := []message.ContentPart{message.TextContent{Text: messageContent}}
	parts = append(parts, attachmentParts...)
	return a.messages.Create(ctx, sessionID, message.CreateMessageParams{
		Role:  message.User,
		Parts: parts,
	})
}

type toolExecResult struct {
	index            int
	result           message.ToolResult
	permissionDenied bool
}

func (a *agent) streamAndHandleEvents(ctx context.Context, sessionID string, msgHistory []message.Message) (message.Message, *message.Message, error) {
	ctx = context.WithValue(ctx, tools.SessionIDContextKey, sessionID)
	
	// Get session and add working directory to context
	session, err := a.sessions.Get(ctx, sessionID)
	if err != nil {
		return message.Message{}, nil, fmt.Errorf("failed to load session %s: %w", sessionID, err)
	}
	// Add session working directory to context
	ctx = context.WithValue(ctx, tools.WorkingDirectoryContextKey, session.WorkingDirectory)
	
	// Get cached session-specific provider
	sessionProvider, err := a.getOrCreateSessionProvider(ctx, sessionID, &session)
	if err != nil {
		return message.Message{}, nil, fmt.Errorf("failed to get session provider: %w", err)
	}
	
	// Filter tools based on plan mode
	availableTools := a.tools
	if ctx.Value("plan_mode") != nil {
		availableTools = filterToolsForPlanMode(a.tools)
	}
	
	eventChan := sessionProvider.StreamResponse(ctx, msgHistory, availableTools)

	assistantMsg, err := a.messages.Create(ctx, sessionID, message.CreateMessageParams{
		Role:  message.Assistant,
		Parts: []message.ContentPart{},
		Model: sessionProvider.Model().ID,
	})
	if err != nil {
		return assistantMsg, nil, fmt.Errorf("failed to create assistant message: %w", err)
	}

	// Add the session and message ID into the context if needed by tools.
	ctx = context.WithValue(ctx, tools.MessageIDContextKey, assistantMsg.ID)

	// Track reasoning start time and ensure cleanup
	reasoningStartTime := time.Now()
	defer func() {
		// Calculate reasoning duration if we have reasoning content
		if assistantMsg.ReasoningContent().Thinking != "" {
			duration := int64(time.Since(reasoningStartTime).Seconds())
			assistantMsg.SetReasoningDuration(duration)
		}
	}()

	// Process each event in the stream.
	for event := range eventChan {
		if processErr := a.processEvent(ctx, sessionID, &assistantMsg, event); processErr != nil {
			a.finishMessage(ctx, &assistantMsg, message.FinishReasonCanceled)
			return assistantMsg, nil, processErr
		}
		if ctx.Err() != nil {
			a.finishMessage(context.Background(), &assistantMsg, message.FinishReasonCanceled)
			return assistantMsg, nil, ctx.Err()
		}
	}

	toolResults := make([]message.ToolResult, len(assistantMsg.ToolCalls()))
	toolCalls := assistantMsg.ToolCalls()

	// Create channel for collecting results from parallel tool execution
	resultChan := make(chan toolExecResult, len(toolCalls))

	// Launch goroutines for parallel tool execution
	var wg sync.WaitGroup
	for i, toolCall := range toolCalls {
		wg.Add(1)
		go func(index int, tc message.ToolCall) {
			defer wg.Done()

			// Check for context cancellation first
			select {
			case <-ctx.Done():
				resultChan <- toolExecResult{
					index: index,
					result: message.ToolResult{
						ToolCallID: tc.ID,
						Content:    "Tool execution canceled by user",
						IsError:    true,
					},
				}
				return
			default:
			}

			// Find tool
			var tool tools.BaseTool
			for _, availableTool := range a.tools {
				if availableTool.Info().Name == tc.Name {
					tool = availableTool
					break
				}
			}

			// Tool not found
			if tool == nil {
				resultChan <- toolExecResult{
					index: index,
					result: message.ToolResult{
						ToolCallID: tc.ID,
						Content:    fmt.Sprintf("Tool not found: %s", tc.Name),
						IsError:    true,
					},
				}
				return
			}

			// Check if tool is available in plan mode
			if ctx.Value("plan_mode") != nil && !isToolAllowedInPlanMode(tool) {
				resultChan <- toolExecResult{
					index: index,
					result: message.ToolResult{
						ToolCallID: tc.ID,
						Content:    "Tool not available in plan mode. Use exit_plan_mode to proceed with execution.",
						IsError:    true,
					},
				}
				return
			}

			logging.Info("[Agent] Executing tool", "toolName", tc.Name, "sessionID", sessionID, "toolCallID", tc.ID, "inputSize", len(tc.Input), "inputContent", tc.Input)

			toolStartTime := time.Now()
			toolResult, toolErr := tool.Run(ctx, tools.ToolCall{
				ID:    tc.ID,
				Name:  tc.Name,
				Input: tc.Input,
			})
			toolDuration := time.Since(toolStartTime)

			logging.Info("[Agent] Tool execution result", "toolName", tc.Name, "sessionID", sessionID, "toolCallID", tc.ID, "duration", toolDuration, "error", toolErr, "resultLength", len(toolResult.Content), "resultContent", toolResult.Content, "resultIsError", toolResult.IsError)

			permissionDenied := false
			if toolErr != nil {
				logging.Info("[Agent] TOOL EXECUTION ERROR", "toolName", tc.Name, "sessionID", sessionID, "toolCallID", tc.ID, "error", toolErr)

				if errors.Is(toolErr, permission.ErrorPermissionDenied) {
					logging.Info("[Agent] TOOL PERMISSION DENIED", "toolName", tc.Name, "sessionID", sessionID, "toolCallID", tc.ID)
					permissionDenied = true
				}
			}

			// Log tool execution result
			isError := toolErr != nil
			if isError {
				logging.Error("[Agent] Tool execution failed", "toolName", tc.Name, "sessionID", sessionID, "toolCallID", tc.ID, "hasError", isError)
			}

			result := message.ToolResult{
				ToolCallID: tc.ID,
				Content:    toolResult.Content,
				Metadata:   toolResult.Metadata,
				IsError:    toolResult.IsError,
			}

			if permissionDenied {
				result.Content = "Permission denied"
				result.IsError = true
			}

			resultChan <- toolExecResult{
				index:            index,
				result:           result,
				permissionDenied: permissionDenied,
			}
		}(i, toolCall)
	}

	// Close channel when all goroutines complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results with simple state tracking
	cancelled := false
	permissionDenied := false

	// Always drain the entire channel - no early exits
	for result := range resultChan {
		// Check for cancellation on each result
		if ctx.Err() != nil {
			cancelled = true
		}

		// Check for permission denied
		if result.permissionDenied {
			permissionDenied = true
		}

		// Only store result if not cancelled and no permission denied
		if !cancelled && !permissionDenied {
			toolResults[result.index] = result.result
		}

		// Only publish events if everything is still OK
		if !cancelled && !permissionDenied {
			err := a.Publish(ctx, pubsub.CreatedEvent, AgentEvent{
				Type:      AgentEventTypeResponse,
				Message:   assistantMsg,
				SessionID: sessionID,
			})
			if err != nil {
				logging.Error("Failed to publish agent event", "error", err)
			}
		}
	}

	// Handle finish messages after all goroutines complete
	if cancelled {
		a.finishMessage(context.Background(), &assistantMsg, message.FinishReasonCanceled)
	} else if permissionDenied {
		a.finishMessage(ctx, &assistantMsg, message.FinishReasonPermissionDenied)
	}

	// Fill any missing results with appropriate error messages
	for i := range toolResults {
		if toolResults[i].ToolCallID == "" {
			content := "Tool execution canceled by user"
			if permissionDenied {
				content = "Tool execution canceled due to permission denied"
			}
			toolResults[i] = message.ToolResult{
				ToolCallID: toolCalls[i].ID,
				Content:    content,
				IsError:    true,
			}
		}
	}

	if len(toolResults) == 0 {
		return assistantMsg, nil, nil
	}
	parts := make([]message.ContentPart, 0)
	for _, tr := range toolResults {
		parts = append(parts, tr)
	}
	msg, err := a.messages.Create(context.Background(), assistantMsg.SessionID, message.CreateMessageParams{
		Role:  message.Tool,
		Parts: parts,
	})
	if err != nil {
		return assistantMsg, nil, fmt.Errorf("failed to create cancelled tool message: %w", err)
	}

	return assistantMsg, &msg, err
}

func (a *agent) finishMessage(ctx context.Context, msg *message.Message, finishReson message.FinishReason) {
	msg.AddFinish(finishReson)
	_ = a.messages.Update(ctx, *msg)
}

func (a *agent) processEvent(ctx context.Context, sessionID string, assistantMsg *message.Message, event provider.ProviderEvent) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Continue processing.
	}

	switch event.Type {
	case provider.EventThinkingDelta:
		assistantMsg.AppendReasoningContent(event.Thinking)
		// Publish thinking event for real-time streaming
		err := a.Publish(ctx, pubsub.CreatedEvent, AgentEvent{
			Type:      AgentEventTypeResponse,
			Message:   *assistantMsg,
			SessionID: sessionID,
		})
		if err != nil {
			return err
		}
		return a.messages.Update(ctx, *assistantMsg)
	case provider.EventContentDelta:
		assistantMsg.AppendContent(event.Content)
		// Content delta streaming removed - only final content will be sent
		return a.messages.Update(ctx, *assistantMsg)
	case provider.EventToolUseStart:
		assistantMsg.AddToolCall(*event.ToolCall)
		// Publish tool start event for real-time streaming
		err := a.Publish(ctx, pubsub.CreatedEvent, AgentEvent{
			Type:      AgentEventTypeResponse,
			Message:   *assistantMsg,
			SessionID: sessionID,
		})
		if err != nil {
			return err
		}
		return a.messages.Update(ctx, *assistantMsg)
	// TODO: see how to handle this
	// case provider.EventToolUseDelta:
	// 	tm := time.Unix(assistantMsg.UpdatedAt, 0)
	// 	assistantMsg.AppendToolCallInput(event.ToolCall.ID, event.ToolCall.Input)
	// 	if time.Since(tm) > 1000*time.Millisecond {
	// 		err := a.messages.Update(ctx, *assistantMsg)
	// 		assistantMsg.UpdatedAt = time.Now().Unix()
	// 		return err
	// 	}
	case provider.EventToolUseStop:
		assistantMsg.FinishToolCall(event.ToolCall.ID)
		// Publish tool completion event for real-time streaming
		err := a.Publish(ctx, pubsub.CreatedEvent, AgentEvent{
			Type:      AgentEventTypeResponse,
			Message:   *assistantMsg,
			SessionID: sessionID,
		})
		if err != nil {
			return err
		}
		return a.messages.Update(ctx, *assistantMsg)
	case provider.EventError:
		if errors.Is(event.Error, context.Canceled) {
			logging.Info("Event processing canceled for session", "sessionID", sessionID)
			return context.Canceled
		}
		logging.Error(event.Error.Error())
		return event.Error
	case provider.EventComplete:
		assistantMsg.SetToolCalls(event.Response.ToolCalls)
		assistantMsg.AddFinish(event.Response.FinishReason)
		if err := a.messages.Update(ctx, *assistantMsg); err != nil {
			return fmt.Errorf("failed to update message: %w", err)
		}
		return a.TrackUsage(ctx, sessionID, a.provider.Model(), event.Response.Usage)
	}

	return nil
}

func (a *agent) TrackUsage(ctx context.Context, sessionID string, model models.Model, usage provider.TokenUsage) error {
	sess, err := a.sessions.Get(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	cost := model.CostPer1MInCached/1e6*float64(usage.CacheCreationTokens) +
		model.CostPer1MOutCached/1e6*float64(usage.CacheReadTokens) +
		model.CostPer1MIn/1e6*float64(usage.InputTokens) +
		model.CostPer1MOut/1e6*float64(usage.OutputTokens)

	sess.Cost += cost
	sess.CompletionTokens = usage.OutputTokens + usage.CacheReadTokens
	sess.PromptTokens = usage.InputTokens + usage.CacheCreationTokens

	_, err = a.sessions.Save(ctx, sess)
	if err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}
	return nil
}

func (a *agent) Update(agentName config.AgentName, modelID models.ModelID) (models.Model, error) {
	if a.IsBusy() {
		return models.Model{}, fmt.Errorf("cannot change model while processing requests")
	}

	if err := config.UpdateAgentModel(agentName, modelID); err != nil {
		return models.Model{}, fmt.Errorf("failed to update config: %w", err)
	}

	provider, err := createAgentProvider(agentName)
	if err != nil {
		return models.Model{}, fmt.Errorf("failed to create provider for model %s: %w", modelID, err)
	}

	a.provider = provider

	return a.provider.Model(), nil
}

func (a *agent) Summarize(ctx context.Context, sessionID string) error {
	if a.summarizeProvider == nil {
		return fmt.Errorf("summarize provider not available")
	}

	// Create a new context with cancellation
	summarizeCtx, cancel := context.WithCancel(ctx)

	// Atomically check and store the cancel function to avoid race conditions
	if _, loaded := a.activeRequests.LoadOrStore(sessionID+"-summarize", cancel); loaded {
		cancel() // Clean up unused cancel function
		return ErrSessionBusy
	}

	go func() {
		defer a.activeRequests.Delete(sessionID + "-summarize")
		defer cancel()
		event := AgentEvent{
			Type:     AgentEventTypeSummarize,
			Progress: "Starting summarization...",
		}

		err := a.Publish(summarizeCtx, pubsub.CreatedEvent, event)
		if err != nil {
			logging.Error("Failed to publish summarize start event", "error", err)
		}
		// Get all messages from the session
		msgs, err := a.messages.List(summarizeCtx, sessionID)
		if err != nil {
			event = AgentEvent{
				Type:  AgentEventTypeError,
				Error: fmt.Errorf("failed to list messages: %w", err),
				Done:  true,
			}
			publishErr := a.Publish(summarizeCtx, pubsub.CreatedEvent, event)
			if publishErr != nil {
				logging.Error("Failed to publish error event", "error", publishErr)
			}
			return
		}
		summarizeCtx = context.WithValue(summarizeCtx, tools.SessionIDContextKey, sessionID)
		
		// Get session working directory and add to context
		session, err := a.sessions.Get(summarizeCtx, sessionID)
		if err == nil {
			summarizeCtx = context.WithValue(summarizeCtx, tools.WorkingDirectoryContextKey, session.WorkingDirectory)
		}

		if len(msgs) == 0 {
			event = AgentEvent{
				Type:  AgentEventTypeError,
				Error: fmt.Errorf("no messages to summarize"),
				Done:  true,
			}
			publishErr := a.Publish(summarizeCtx, pubsub.CreatedEvent, event)
			if publishErr != nil {
				logging.Error("Failed to publish error event", "error", publishErr)
			}
			return
		}

		event = AgentEvent{
			Type:     AgentEventTypeSummarize,
			Progress: "Analyzing conversation...",
		}
		err = a.Publish(summarizeCtx, pubsub.CreatedEvent, event)
		if err != nil {
			logging.Error("Failed to publish analyze event", "error", err)
		}

		// Add a system message to guide the summarization
		summarizePrompt := "Provide a detailed but concise summary of our conversation above. Focus on information that would be helpful for continuing the conversation, including what we did, what we're doing, which files we're working on, and what we're going to do next."

		// Create a new message with the summarize prompt
		promptMsg := message.Message{
			Role:  message.User,
			Parts: []message.ContentPart{message.TextContent{Text: summarizePrompt}},
		}

		// Append the prompt to the messages
		msgsWithPrompt := append(msgs, promptMsg)

		event = AgentEvent{
			Type:     AgentEventTypeSummarize,
			Progress: "Generating summary...",
		}

		err = a.Publish(summarizeCtx, pubsub.CreatedEvent, event)
		if err != nil {
			logging.Error("Failed to publish generate event", "error", err)
		}

		// Send the messages to the summarize provider
		response, err := a.summarizeProvider.SendMessages(
			summarizeCtx,
			msgsWithPrompt,
			make([]tools.BaseTool, 0),
		)
		if err != nil {
			event = AgentEvent{
				Type:  AgentEventTypeError,
				Error: fmt.Errorf("failed to summarize: %w", err),
				Done:  true,
			}
			publishErr := a.Publish(summarizeCtx, pubsub.CreatedEvent, event)
			if publishErr != nil {
				logging.Error("Failed to publish error event", "error", publishErr)
			}
			return
		}

		summary := strings.TrimSpace(response.Content)
		if summary == "" {
			event = AgentEvent{
				Type:  AgentEventTypeError,
				Error: fmt.Errorf("empty summary returned"),
				Done:  true,
			}
			publishErr := a.Publish(summarizeCtx, pubsub.CreatedEvent, event)
			if publishErr != nil {
				logging.Error("Failed to publish error event", "error", publishErr)
			}
			return
		}
		event = AgentEvent{
			Type:     AgentEventTypeSummarize,
			Progress: "Creating new session...",
		}

		err = a.Publish(summarizeCtx, pubsub.CreatedEvent, event)
		if err != nil {
			logging.Error("Failed to publish create session event", "error", err)
		}
		oldSession, err := a.sessions.Get(summarizeCtx, sessionID)
		if err != nil {
			event = AgentEvent{
				Type:  AgentEventTypeError,
				Error: fmt.Errorf("failed to get session: %w", err),
				Done:  true,
			}

			publishErr := a.Publish(summarizeCtx, pubsub.CreatedEvent, event)
			if publishErr != nil {
				logging.Error("Failed to publish error event", "error", publishErr)
			}
			return
		}
		// Create a message in the new session with the summary
		msg, err := a.messages.Create(summarizeCtx, oldSession.ID, message.CreateMessageParams{
			Role: message.Assistant,
			Parts: []message.ContentPart{
				message.TextContent{Text: summary},
				message.Finish{
					Reason: message.FinishReasonEndTurn,
					Time:   time.Now().Unix(),
				},
			},
			Model: a.summarizeProvider.Model().ID,
		})
		if err != nil {
			event = AgentEvent{
				Type:  AgentEventTypeError,
				Error: fmt.Errorf("failed to create summary message: %w", err),
				Done:  true,
			}

			publishErr := a.Publish(summarizeCtx, pubsub.CreatedEvent, event)
			if publishErr != nil {
				logging.Error("Failed to publish error event", "error", publishErr)
			}
			return
		}
		oldSession.SummaryMessageID = msg.ID
		oldSession.CompletionTokens = response.Usage.OutputTokens
		oldSession.PromptTokens = 0
		model := a.summarizeProvider.Model()
		usage := response.Usage
		cost := model.CostPer1MInCached/1e6*float64(usage.CacheCreationTokens) +
			model.CostPer1MOutCached/1e6*float64(usage.CacheReadTokens) +
			model.CostPer1MIn/1e6*float64(usage.InputTokens) +
			model.CostPer1MOut/1e6*float64(usage.OutputTokens)
		oldSession.Cost += cost
		_, err = a.sessions.Save(summarizeCtx, oldSession)
		if err != nil {
			event = AgentEvent{
				Type:  AgentEventTypeError,
				Error: fmt.Errorf("failed to save session: %w", err),
				Done:  true,
			}
			publishErr := a.Publish(summarizeCtx, pubsub.CreatedEvent, event)
			if publishErr != nil {
				logging.Error("Failed to publish error event", "error", publishErr)
			}
		}

		event = AgentEvent{
			Type:      AgentEventTypeSummarize,
			SessionID: oldSession.ID,
			Progress:  "Summary complete",
			Done:      true,
		}
		err = a.Publish(summarizeCtx, pubsub.CreatedEvent, event)
		if err != nil {
			logging.Error("Failed to publish complete event", "error", err)
		}
		// Send final success event with the new session ID
	}()

	return nil
}

// filterToolsForPlanMode returns only read-only and planning tools for plan mode
func filterToolsForPlanMode(allTools []tools.BaseTool) []tools.BaseTool {
	var planModeTools []tools.BaseTool
	for _, tool := range allTools {
		if isToolAllowedInPlanMode(tool) {
			planModeTools = append(planModeTools, tool)
		}
	}
	return planModeTools
}

// isToolAllowedInPlanMode checks if a tool is allowed in plan mode
func isToolAllowedInPlanMode(tool tools.BaseTool) bool {
	toolName := tool.Info().Name
	
	// Allow read-only and planning tools
	allowedTools := map[string]bool{
		"view":           true,
		"ls":             true,
		"grep":           true,
		"glob":           true,
		"todo_write":     true,
		"exit_plan_mode": true,
		"fetch":          true,
	}
	
	return allowedTools[toolName]
}

func createAgentProvider(agentName config.AgentName) (provider.Provider, error) {
	cfg := config.Get()
	agentConfig, ok := cfg.Agents[agentName]
	if !ok {
		return nil, fmt.Errorf("agent %s not found", agentName)
	}
	model, ok := models.SupportedModels[agentConfig.Model]
	if !ok {
		return nil, fmt.Errorf("model %s not supported", agentConfig.Model)
	}

	providerCfg, ok := cfg.Providers[model.Provider]
	if !ok {
		return nil, fmt.Errorf("provider %s not supported", model.Provider)
	}
	if providerCfg.Disabled {
		return nil, fmt.Errorf("provider %s is not enabled", model.Provider)
	}
	// Note: API key validation removed - let provider client handle authentication
	// This allows providers to support multiple authentication methods (OAuth, API key, etc.)
	maxTokens := model.DefaultMaxTokens
	if agentConfig.MaxTokens > 0 {
		maxTokens = agentConfig.MaxTokens
	}
	opts := []provider.ProviderClientOption{
		provider.WithAPIKey(providerCfg.APIKey),
		provider.WithModel(model),
		provider.WithMaxTokens(maxTokens),
	}
	if model.Provider == models.ProviderOpenAI || model.Provider == models.ProviderLocal && model.CanReason {
		opts = append(
			opts,
			provider.WithOpenAIOptions(
				provider.WithReasoningEffort(agentConfig.ReasoningEffort),
			),
		)
	} else if model.Provider == models.ProviderAnthropic && model.CanReason && agentName == config.AgentMain {
		opts = append(
			opts,
			provider.WithAnthropicOptions(
				provider.WithAnthropicShouldThinkFn(provider.DefaultShouldThinkFn),
			),
		)
	}
	agentProvider, err := provider.NewProvider(
		model.Provider,
		opts...,
	)
	if err != nil {
		return nil, fmt.Errorf("could not create provider: %v", err)
	}

	return agentProvider, nil
}

func createSessionProvider(ctx context.Context, agentName config.AgentName, sess *session.Session) (provider.Provider, error) {
	cfg := config.Get()
	agentConfig, ok := cfg.Agents[agentName]
	if !ok {
		return nil, fmt.Errorf("agent %s not found", agentName)
	}
	model, ok := models.SupportedModels[agentConfig.Model]
	if !ok {
		return nil, fmt.Errorf("model %s not supported", agentConfig.Model)
	}

	providerCfg, ok := cfg.Providers[model.Provider]
	if !ok {
		return nil, fmt.Errorf("provider %s not supported", model.Provider)
	}
	if providerCfg.Disabled {
		return nil, fmt.Errorf("provider %s is not enabled", model.Provider)
	}

	maxTokens := model.DefaultMaxTokens
	if agentConfig.MaxTokens > 0 {
		maxTokens = agentConfig.MaxTokens
	}

	// Create session-specific variables
	sessionVars := map[string]string{}
	if sess != nil {
		sessionVars["session_id"] = sess.ID
		sessionVars["session_workdir"] = sess.WorkingDirectory
	}

	// Get system prompt with session variables
	systemPrompt := prompt.GetAgentPromptWithVars(ctx, agentName, model.Provider, sessionVars)

	opts := []provider.ProviderClientOption{
		provider.WithAPIKey(providerCfg.APIKey),
		provider.WithModel(model),
		provider.WithSystemMessage(systemPrompt),
		provider.WithMaxTokens(maxTokens),
	}
	if model.Provider == models.ProviderOpenAI || model.Provider == models.ProviderLocal && model.CanReason {
		opts = append(
			opts,
			provider.WithOpenAIOptions(
				provider.WithReasoningEffort(agentConfig.ReasoningEffort),
			),
		)
	} else if model.Provider == models.ProviderAnthropic && model.CanReason && agentName == config.AgentMain {
		opts = append(
			opts,
			provider.WithAnthropicOptions(
				provider.WithAnthropicShouldThinkFn(provider.DefaultShouldThinkFn),
			),
		)
	}
	sessionProvider, err := provider.NewProvider(
		model.Provider,
		opts...,
	)
	if err != nil {
		return nil, fmt.Errorf("could not create session provider: %v", err)
	}

	return sessionProvider, nil
}

func (a *agent) getOrCreateSessionProvider(ctx context.Context, sessionID string, session *session.Session) (provider.Provider, error) {
	// Create new session provider
	sessionProvider, err := createSessionProvider(ctx, a.agentName, session)
	if err != nil {
		return nil, fmt.Errorf("failed to create session provider: %w", err)
	}

	// Atomically store if not exists, or load existing
	actual, loaded := a.sessionProviders.LoadOrStore(sessionID, sessionProvider)
	if loaded {
		// Another goroutine already created one, use theirs
		return actual.(provider.Provider), nil
	}

	// We successfully stored our provider
	return sessionProvider, nil
}

func (a *agent) Shutdown() {
	a.cancel()
}

func (a *agent) handleSessionEvents() {
	eventsChan := a.sessions.Subscribe(a.ctx)

	for event := range eventsChan {
		if event.Type == pubsub.DeletedEvent {
			sessionID := event.Payload.ID
			// Remove cached provider for deleted session
			if _, existed := a.sessionProviders.LoadAndDelete(sessionID); existed {
				logging.Info("Cleaned up session provider cache", "sessionID", sessionID)
			}
		}
	}
}
