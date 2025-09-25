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
	"mix/internal/llm/interfaces"
	"mix/internal/llm/provider"
	"mix/internal/llm/tools"
	"mix/internal/logging"
	"mix/internal/message"
	"mix/internal/preferences"
	"mix/internal/pubsub"
	"mix/internal/session"
)

// Common errors
var (
	ErrRequestCancelled = errors.New("request cancelled by user")
)

type AgentEventType string

const (
	AgentEventTypeError                 AgentEventType = "error"
	AgentEventTypeResponse              AgentEventType = "response"
	AgentEventTypeSummarize             AgentEventType = "summarize"
	AgentEventTypeThinking              AgentEventType = "thinking"
	AgentEventTypeContentDelta          AgentEventType = "content_delta"
	AgentEventTypeToolExecutionStart    AgentEventType = "tool_execution_start"
	AgentEventTypeToolExecutionComplete AgentEventType = "tool_execution_complete"
)

type AgentEvent struct {
	Type    AgentEventType
	Message message.Message
	Error   error

	// When summarizing
	SessionID string
	Progress  string
	Done      bool

	// When thinking
	Thinking string

	// When streaming content
	Content string

	// When executing tools
	ToolCallID string
}

type Service interface {
	pubsub.Suscriber[AgentEvent]
	Model() models.Model
	Run(ctx context.Context, sessionID string, content string, attachments ...message.Attachment) (<-chan AgentEvent, error)
	RunWithPlanMode(ctx context.Context, sessionID string, content string, planMode bool, attachments ...message.Attachment) (<-chan AgentEvent, error)
	Cancel(sessionID string)
	Update(agentName config.AgentName, modelID models.ModelID) (models.Model, error)
	Summarize(ctx context.Context, sessionID string) error
	ClearAllSessionProviders()
	Shutdown()
}

type agent struct {
	*pubsub.Broker[AgentEvent]
	sessions      session.Service
	messages      message.Service
	storageConfig session.Config

	agentName config.AgentName
	tools     []tools.BaseTool
	provider  interfaces.Provider

	titleProvider     interfaces.Provider
	summarizeProvider interfaces.Provider

	sessionProviders sync.Map // Maps session ID to interfaces.Provider
	activeContexts   sync.Map  // Maps session ID to context.CancelFunc for cancellation

	ctx    context.Context
	cancel context.CancelFunc
}

func NewAgent(
	agentName config.AgentName,
	sessions session.Service,
	messages message.Service,
	agentTools []tools.BaseTool,
	storageConfig session.Config,
) (Service, error) {
	agentProvider, err := createAgentProvider(agentName)
	if err != nil {
		return nil, err
	}
	var titleProvider interfaces.Provider
	// Only generate titles for the main agent
	if agentName == config.AgentMain {
		titleProvider, err = createAgentProvider(config.AgentMain)
		if err != nil {
			return nil, err
		}
	}
	var summarizeProvider interfaces.Provider
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
		storageConfig:     storageConfig,
		tools:             agentTools,
		titleProvider:     titleProvider,
		summarizeProvider: summarizeProvider,
		sessionProviders:  sync.Map{},
		activeContexts:    sync.Map{},
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
	if cancelFunc, exists := a.activeContexts.LoadAndDelete(sessionID); exists {
		if cancel, ok := cancelFunc.(context.CancelFunc); ok {
			// Request cancellation initiated
			cancel()
		}
	}

	// Also check for summarize requests
	if cancelFunc, exists := a.activeContexts.LoadAndDelete(sessionID + "-summarize"); exists {
		if cancel, ok := cancelFunc.(context.CancelFunc); ok {
			// Summarize cancellation initiated
			cancel()
		}
	}
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

	// Add session storage directory to context
	ctx = tools.SetSessionStorageContext(ctx, session.ID, a.storageConfig)

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
	// Store cancel function for potential cancellation
	a.activeContexts.Store(sessionID, cancel)

	// Add plan mode to context
	if planMode {
		genCtx = context.WithValue(genCtx, "plan_mode", true)
	}

	// Subscribe to agent events for real-time streaming
	subscription := a.Subscribe(genCtx)

	go func() {
		defer func() {
			logging.Debug("Request completed", "sessionID", sessionID)
			a.activeContexts.Delete(sessionID)
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
	// Starting message processing for session
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

	conversationTurn := 1
	for {
		// Check for cancellation before each iteration
		select {
		case <-ctx.Done():
			return a.err(ctx.Err())
		default:
			// Continue processing
		}

		// Starting conversation turn

		agentMessage, toolResults, err := a.streamAndHandleEvents(ctx, sessionID, msgHistory)
		if err != nil {
			// Stream processing failed for session
			if errors.Is(err, context.Canceled) {
				agentMessage.AddFinish(message.FinishReasonCanceled)
				a.messages.Update(context.Background(), agentMessage)
				return a.err(ErrRequestCancelled)
			}
			return a.err(fmt.Errorf("failed to process events: %w", err))
		}

		// Enhanced tool results logging for debugging
		if toolResults != nil {
			// Tool results processed
		}
		if (agentMessage.FinishReason() == message.FinishReasonToolUse) && toolResults != nil {
			// We are not done, we need to respond with the tool response
			// Tool execution completed, continuing conversation
			msgHistory = append(msgHistory, agentMessage, *toolResults)
			conversationTurn++
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
		planModeContent, err := prompt.LoadPrompt(ctx, "plan_mode", nil)
		if err != nil {
			return message.Message{}, fmt.Errorf("failed to load plan mode prompt: %w", err)
		}
		messageContent = content + "\n\n<system-reminder>\n" + planModeContent + "\n</system-reminder>"
	}

	parts := []message.ContentPart{message.TextContent{Text: messageContent}}
	parts = append(parts, attachmentParts...)
	return a.messages.Create(ctx, sessionID, message.CreateMessageParams{
		Role:  message.User,
		Parts: parts,
	})
}

func (a *agent) streamAndHandleEvents(ctx context.Context, sessionID string, msgHistory []message.Message) (message.Message, *message.Message, error) {
	ctx = context.WithValue(ctx, tools.SessionIDContextKey, sessionID)

	// Get session and add working directory to context
	session, err := a.sessions.Get(ctx, sessionID)
	if err != nil {
		return message.Message{}, nil, fmt.Errorf("failed to load session %s: %w", sessionID, err)
	}
	// Add session storage directory to context
	ctx = tools.SetSessionStorageContext(ctx, session.ID, a.storageConfig)

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

	toolCalls := assistantMsg.ToolCalls()
	if len(toolCalls) == 0 {
		return assistantMsg, nil, nil
	}

	// Execute all tool calls with dependency awareness
	logging.Debug("Processing tool calls", "count", len(toolCalls), "sessionID", sessionID)

	toolResults, err := a.executeToolsWithDependencies(ctx, sessionID, toolCalls, assistantMsg)
	if err != nil {
		return assistantMsg, nil, fmt.Errorf("failed to execute tools: %w", err)
	}

	// Create tool result message with all results
	msg, err := a.messages.Create(context.Background(), assistantMsg.SessionID, message.CreateMessageParams{
		Role:  message.Tool,
		Parts: toolResults,
	})
	if err != nil {
		return assistantMsg, nil, fmt.Errorf("failed to create tool result message: %w", err)
	}

	// Publish completion event
	err = a.Publish(ctx, pubsub.CreatedEvent, AgentEvent{
		Type:      AgentEventTypeResponse,
		Message:   assistantMsg,
		SessionID: sessionID,
	})
	if err != nil {
		logging.Error("Failed to publish agent event", "error", err)
	}

	return assistantMsg, &msg, nil
}

func (a *agent) finishMessage(ctx context.Context, msg *message.Message, finishReson message.FinishReason) {
	msg.AddFinish(finishReson)
	_ = a.messages.Update(context.Background(), *msg)
}

func (a *agent) processEvent(ctx context.Context, sessionID string, assistantMsg *message.Message, event interfaces.ProviderEvent) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Continue processing.
	}

	switch event.Type {
	case interfaces.EventThinkingDelta:
		// Claude thinking delta received
		assistantMsg.AppendReasoningContent(event.Thinking)
		// Publish thinking event for real-time streaming
		err := a.Publish(ctx, pubsub.CreatedEvent, AgentEvent{
			Type:      AgentEventTypeThinking,
			Message:   *assistantMsg,
			SessionID: sessionID,
			Thinking:  event.Thinking,
		})
		if err != nil {
			return err
		}
		return a.messages.Update(context.Background(), *assistantMsg)
	case interfaces.EventContentDelta:
		assistantMsg.AppendContent(event.Content)
		// Publish content delta event for real-time streaming
		err := a.Publish(ctx, pubsub.CreatedEvent, AgentEvent{
			Type:      AgentEventTypeContentDelta,
			Message:   *assistantMsg,
			SessionID: sessionID,
			Content:   event.Content, // Send only the delta, not accumulated content
		})
		if err != nil {
			return err
		}
		return a.messages.Update(context.Background(), *assistantMsg)
	case interfaces.EventToolUseStart:
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
		return a.messages.Update(context.Background(), *assistantMsg)
	// TODO: see how to handle this
	// case interfaces.EventToolUseDelta:
	// 	tm := time.Unix(assistantMsg.UpdatedAt, 0)
	// 	assistantMsg.AppendToolCallInput(event.ToolCall.ID, event.ToolCall.Input)
	// 	if time.Since(tm) > 1000*time.Millisecond {
	// 		err := a.messages.Update(ctx, *assistantMsg)
	// 		assistantMsg.UpdatedAt = time.Now().Unix()
	// 		return err
	// 	}
	case interfaces.EventToolUseStop:
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
		return a.messages.Update(context.Background(), *assistantMsg)
	case interfaces.EventError:
		if errors.Is(event.Error, context.Canceled) {
			// Event processing canceled for session
			return context.Canceled
		}
		logging.Error(event.Error.Error())
		return event.Error
	case interfaces.EventComplete:
		assistantMsg.SetToolCalls(event.Response.ToolCalls)
		assistantMsg.AddFinish(event.Response.FinishReason)

		// Store thinking blocks from the response
		for _, thinkingBlock := range event.Response.ThinkingBlocks {
			assistantMsg.AddThinkingBlock(thinkingBlock.Thinking, thinkingBlock.Signature)
		}
		for _, redactedBlock := range event.Response.RedactedThinkingBlocks {
			assistantMsg.AddRedactedThinkingBlock(redactedBlock.Data)
		}

		if err := a.messages.Update(context.Background(), *assistantMsg); err != nil {
			return fmt.Errorf("failed to update message: %w", err)
		}
		return a.TrackUsage(ctx, sessionID, a.provider.Model(), event.Response.Usage)
	}

	return nil
}

func (a *agent) TrackUsage(ctx context.Context, sessionID string, model models.Model, usage interfaces.TokenUsage) error {
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
	// Allow model changes at any time since operations are no longer globally synchronized

	// Update agent model in database instead of config file
	userPrefs := config.GetUserPreferences()
	if userPrefs == nil {
		return models.Model{}, fmt.Errorf("user preferences service not available")
	}

	model, ok := models.SupportedModels[modelID]
	if !ok {
		return models.Model{}, fmt.Errorf("model %s not supported", modelID)
	}

	ctx := context.Background()
	var err error

	switch agentName {
	case config.AgentMain:
		err = userPrefs.UpdateMainAgentPreferences(ctx, modelID, model.DefaultMaxTokens, "")
	case config.AgentSub:
		err = userPrefs.UpdateSubAgentPreferences(ctx, modelID, model.DefaultMaxTokens, "")
	default:
		return models.Model{}, fmt.Errorf("unknown agent name: %s", agentName)
	}

	if err != nil {
		return models.Model{}, fmt.Errorf("failed to update agent preferences in database: %w", err)
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

	// Store cancel function for potential cancellation
	a.activeContexts.Store(sessionID+"-summarize", cancel)

	go func() {
		defer a.activeContexts.Delete(sessionID + "-summarize")
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
			summarizeCtx = tools.SetSessionStorageContext(summarizeCtx, session.ID, a.storageConfig)
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
		"ReadText":             true,
		"ls":                   true,
		"grep":                 true,
		"glob":                 true,
		"todo_write":           true,
		"exit_plan_mode":       true,
		"fetch":                true,
		"ReadMedia":            true,
	}

	return allowedTools[toolName]
}

// This function has been deprecated - we're now using database only for credentials
// Keeping the function signature to avoid breaking code elsewhere
func getProviderAPIKeyFromEnv(modelProvider models.ModelProvider) string {
	// Return empty string to force database-only credential lookup
	return ""
}

func createAgentProvider(agentName config.AgentName) (interfaces.Provider, error) {
	// Try to get agent config from database first
	ctx := context.Background()
	agentConfig, err := config.GetAgentFromDatabase(ctx, agentName)
	if err != nil {
		// Fall back to default agent config
		logging.Warn("Failed to get agent config from database, using default", "error", err, "agent", agentName)
		// Use Claude as default model if database not available
		agentConfig = config.Agent{
			Model:     "claude-4-sonnet",
			MaxTokens: 4096,
		}
	}

	// Check user's preferred provider if available
	userPrefs := config.GetUserPreferences()
	if userPrefs != nil {
		preferredProvider, providerErr := userPrefs.GetPreferredProvider(ctx)
		if providerErr == nil && preferredProvider != "" {
			// Validate that the selected model is available on the preferred provider
			model, modelExists := models.SupportedModels[agentConfig.Model]
			if modelExists && model.Provider != preferredProvider {
				// Model not available on preferred provider, using model's default provider
			}
		}
	}
	model, ok := models.SupportedModels[agentConfig.Model]
	if !ok {
		return nil, fmt.Errorf("model %s not supported", agentConfig.Model)
	}

	// Get API key - ONLY from database, no fallbacks to config or env
	var apiKey string

	credentialsService := config.GetAPICredentials()
	if credentialsService != nil {
		dbKey, err := credentialsService.GetAPIKey(ctx, model.Provider)
		if err == nil && dbKey != "" {
			apiKey = dbKey
			// Using database-stored API key
		} else {
			// No key in database, we won't use environment or config fallbacks
			// For OAuth providers, we'll let the client check for OAuth tokens
			if model.Provider != models.ProviderAnthropic && model.Provider != models.ProviderOpenAI {
				logging.Warn("No API key found in database for provider", "provider", model.Provider)
			}
		}
	}

	// Set up provider options
	maxTokens := model.DefaultMaxTokens
	if agentConfig.MaxTokens > 0 {
		maxTokens = agentConfig.MaxTokens
	}

	opts := []provider.ProviderClientOption{
		provider.WithAPIKey(apiKey),
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
	} else if model.Provider == models.ProviderAnthropic {
		opts = append(
			opts,
			provider.WithAnthropicOptions(
				provider.WithAnthropicThinkingBudgetFn(provider.DefaultThinkingBudgetFn),
				provider.WithAnthropicInterleavedThinking(),
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

func createSessionProvider(ctx context.Context, agentName config.AgentName, sess *session.Session, storageConfig session.Config) (interfaces.Provider, error) {
	// Try to get agent config from database first
	agentConfig, err := config.GetAgentFromDatabase(ctx, agentName)
	if err != nil {
		// Fall back to default agent config
		logging.Warn("Failed to get agent config from database for session, using default", "error", err, "agent", agentName)
		// Use Claude as default model if database not available
		agentConfig = config.Agent{
			Model:     "claude-4-sonnet",
			MaxTokens: 4096,
		}
	}

	model, ok := models.SupportedModels[agentConfig.Model]
	if !ok {
		return nil, fmt.Errorf("model %s not supported", agentConfig.Model)
	}

	// Get API key - ONLY from database, no fallbacks to config or env
	var apiKey string

	// Get from database only
	credentialsService := config.GetAPICredentials()
	if credentialsService != nil {
		dbKey, err := credentialsService.GetAPIKey(ctx, model.Provider)
		if err == nil && dbKey != "" {
			apiKey = dbKey
			// Using database-stored API key for session provider
		} else {
			// No key in database, we won't use environment or config fallbacks
			// For OAuth providers, we'll let the client check for OAuth tokens
			if model.Provider != models.ProviderAnthropic && model.Provider != models.ProviderOpenAI {
				logging.Warn("No API key found in database for provider in session provider", "provider", model.Provider)
			}
		}
	}

	maxTokens := model.DefaultMaxTokens
	if agentConfig.MaxTokens > 0 {
		maxTokens = agentConfig.MaxTokens
	}

	// Create session-specific variables
	sessionVars := map[string]string{}
	if sess != nil {
		sessionVars["session_id"] = sess.ID
		sessionVars["workdir"] = session.GetSessionStoragePath(sess.ID, storageConfig)
	}

	// Get system prompt with session variables
	systemPrompt, err := prompt.GetAgentPromptWithVars(ctx, agentName, model.Provider, sessionVars)
	if err != nil {
		return nil, fmt.Errorf("failed to load system prompt: %w", err)
	}

	opts := []provider.ProviderClientOption{
		provider.WithAPIKey(apiKey),
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
				provider.WithAnthropicThinkingBudgetFn(provider.DefaultThinkingBudgetFn),
				provider.WithAnthropicInterleavedThinking(),
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

func (a *agent) getOrCreateSessionProvider(ctx context.Context, sessionID string, session *session.Session) (interfaces.Provider, error) {
	// Get user preferences to log current settings
	userPrefs := config.GetUserPreferences()
	var preferredProvider models.ModelProvider
	var mainAgentModel models.ModelID
	if userPrefs != nil {
		pref, err := userPrefs.GetPreferredProvider(ctx)
		if err == nil {
			preferredProvider = pref
		}
		agentCfg, err := userPrefs.GetAgentConfig(ctx, preferences.AgentMain)
		if err == nil {
			mainAgentModel = agentCfg.Model
		}
	}
	// Current user preferences logged

	// Check if we already have a cached provider
	cached, exists := a.sessionProviders.Load(sessionID)
	if exists {
		cachedProvider := cached.(interfaces.Provider)
		currentModel := cachedProvider.Model()
		// Found cached provider

		// Important: Check if the cached provider matches current preferences
		isMatch := true

		// Only check for preferred provider if it's actually set
		if preferredProvider != "" && currentModel.Provider != preferredProvider {
			// Cached provider does not match current preferred provider
			isMatch = false
		}

		// Only check for model match if using main agent (sub agents might use different models)
		if a.agentName == config.AgentMain && mainAgentModel != "" && currentModel.ID != mainAgentModel {
			// Cached model does not match current preferred model
			isMatch = false
		}

		// If cache doesn't match current preferences, don't use it
		if !isMatch {
			// Discarding outdated cached provider due to preference mismatch
			// Remove the outdated provider from cache
			a.sessionProviders.Delete(sessionID)
		} else {
			// Cache is valid, use it
			return cachedProvider, nil
		}
	}

	// Create new session provider
	// Creating new session provider
	sessionProvider, err := createSessionProvider(ctx, a.agentName, session, a.storageConfig)
	if err != nil {
		logging.Error("Failed to create session provider", "sessionID", sessionID, "error", err)
		return nil, fmt.Errorf("failed to create session provider: %w", err)
	}

	// Created new provider

	// Store the new provider in cache
	a.sessionProviders.Store(sessionID, sessionProvider)
	// Successfully stored new provider in cache
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
				// Cleaned up session provider cache
			}
		}
	}
}

// ClearAllSessionProviders removes all cached providers from memory,
// forcing them to be recreated with the latest preferences on next use
func (a *agent) ClearAllSessionProviders() {
	// Log the count of cached providers
	cachedCount := 0
	a.sessionProviders.Range(func(key, value interface{}) bool {
		cachedCount++
		return true
	})

	// Create a list of all keys to delete
	keysToDelete := []string{}
	a.sessionProviders.Range(func(key, value interface{}) bool {
		if sessionID, ok := key.(string); ok {
			keysToDelete = append(keysToDelete, sessionID)
			// Log cached provider details for debugging
			if provider, ok := value.(interfaces.Provider); ok {
				logging.Debug("Found cached provider", "sessionID", sessionID,
					"provider", provider.Model().Provider,
					"model", provider.Model().ID)
			}
		}
		return true // Continue iterating
	})

	// Delete all keys
	for _, sessionID := range keysToDelete {
		a.sessionProviders.Delete(sessionID)
		// Cleared provider cache for session
	}

	// Verify cache was cleared
	remainCount := 0
	a.sessionProviders.Range(func(key, value interface{}) bool {
		remainCount++
		return true
	})

	// Force a refresh of the main provider as well
	// Try to update the main provider
	newProvider, err := createAgentProvider(a.agentName)
	if err == nil {
		a.provider = newProvider
		// Updated main agent provider after preference change
	} else {
		logging.Error("Failed to update main agent provider", "error", err)
	}
}
