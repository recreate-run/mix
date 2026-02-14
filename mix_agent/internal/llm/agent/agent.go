package agent

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"mix/internal/config"
	"os"
	"mix/internal/llm/callbacks"
	"mix/internal/llm/interfaces"
	"mix/internal/llm/models"
	"mix/internal/llm/prompt"
	"mix/internal/llm/provider"
	"mix/internal/llm/tools"
	"mix/internal/logging"
	"mix/internal/message"
	"mix/internal/permission"
	"mix/internal/preferences"
	"mix/internal/pubsub"
	"mix/internal/session"
	"strings"
	"sync"
	"time"
)

// Global operation tracker for session cleanup synchronization
// Maps session ID to *sync.WaitGroup for tracking in-flight operations
var globalOperationTracker sync.Map

func init() {
	// Set up hook for session deletion to wait for in-flight operations
	session.WaitForOperations = WaitForSessionOperations
}

// trackSessionOperation increments the operation counter for a session
func trackSessionOperation(sessionID string) *sync.WaitGroup {
	wg, _ := globalOperationTracker.LoadOrStore(sessionID, &sync.WaitGroup{})
	wgTyped := wg.(*sync.WaitGroup)
	wgTyped.Add(1)
	return wgTyped
}

// WaitForSessionOperations waits for all in-flight operations for a session to complete
// This is called by session.Delete() to ensure no operations are in progress before deletion
func WaitForSessionOperations(sessionID string, timeout time.Duration) {
	wgRaw, exists := globalOperationTracker.Load(sessionID)
	if !exists {
		return // No operations to wait for
	}

	wg := wgRaw.(*sync.WaitGroup)

	// Wait with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All operations completed
		globalOperationTracker.Delete(sessionID)
	case <-time.After(timeout):
		// Timeout - log warning but continue with deletion
		logging.Warn(fmt.Sprintf("Timeout waiting for session %s operations to complete", sessionID))
	}
}

type contextKey string

const sessionRoutingKey contextKey = "session_routing"

type sessionRouting struct {
	RouteTo          string // Where events should be sent
	Origin           string // Where events originated from
	ParentToolCallID string // Which tool call spawned this subagent session
}

// getRoutingOrEmpty extracts existing routing from context or returns empty struct
func getRoutingOrEmpty(ctx context.Context) sessionRouting {
	if r, ok := ctx.Value(sessionRoutingKey).(sessionRouting); ok {
		return r
	}
	return sessionRouting{}
}

func withSessionRouting(ctx context.Context, routeTo, origin string) context.Context {
	routing := getRoutingOrEmpty(ctx)
	routing.RouteTo = routeTo
	routing.Origin = origin
	return context.WithValue(ctx, sessionRoutingKey, routing)
}

func getSessionRouting(ctx context.Context) (routeTo, origin, parentToolCallID string) {
	if r, ok := ctx.Value(sessionRoutingKey).(sessionRouting); ok {
		return r.RouteTo, r.Origin, r.ParentToolCallID
	}
	return "", "", ""
}

// withToolContext wraps context with tool call ID for subagent event tracking
// Package-private since task-tool.go is in the same package
func withToolContext(ctx context.Context, toolCallID string) context.Context {
	routing := getRoutingOrEmpty(ctx)
	routing.ParentToolCallID = toolCallID
	return context.WithValue(ctx, sessionRoutingKey, routing)
}

// Common errors
var (
	// Deprecated: Use specific error types below
	ErrRequestCancelled = errors.New("request cancelled by user")

	// Specific cancellation reasons for better diagnostics
	ErrRequestCancelledByUser     = errors.New("request cancelled by user")
	ErrRequestCancelledTimeout    = errors.New("request cancelled: timeout exceeded")
	ErrRequestCancelledDisconnect = errors.New("request cancelled: client disconnected")
)

// SessionState represents the current state of an agent session
type SessionState string

const (
	SessionStateCreated    SessionState = "created"
	SessionStateProcessing SessionState = "processing"
	SessionStateCompleted  SessionState = "completed"
	SessionStateCancelled  SessionState = "cancelled"
)

type AgentEventType string

const (
	AgentEventTypeError                 AgentEventType = "error"
	AgentEventTypeResponse              AgentEventType = "response"
	AgentEventTypeThinking              AgentEventType = "thinking"
	AgentEventTypeContentDelta          AgentEventType = "content_delta"
	AgentEventTypeToolParameterDelta    AgentEventType = "tool_parameter_delta"
	AgentEventTypeToolExecutionStart    AgentEventType = "tool_execution_start"
	AgentEventTypeToolExecutionComplete AgentEventType = "tool_execution_complete"
	AgentEventTypeUserMessageCreated    AgentEventType = "user_message_created"
)

type AgentEvent struct {
	Type    AgentEventType
	Message message.Message
	Error   error

	// Routing fields
	SessionID        string // What this event is about (provenance/origin)
	RouteTo          string // Where to send this event (destination for SSE)
	ParentToolCallID string // Which tool call spawned this subagent session

	// When summarizing
	Progress string
	Done     bool

	// When thinking
	Thinking string

	// When streaming content
	Content string

	// When executing tools
	ToolCallID string

	// Snapshot of tool call state at event creation time
	// This prevents race conditions where the tool call is mutated after the event is published
	// but before it's broadcast to SSE clients
	ToolCallSnapshot *message.ToolCall
}

type Service interface {
	pubsub.Suscriber[AgentEvent]
	Model() models.Model
	GetBroker() *pubsub.Broker[AgentEvent]
	Run(ctx context.Context, sessionID string, content string, attachments ...message.Attachment) (<-chan AgentEvent, error)
	RunWithPlanMode(ctx context.Context, sessionID string, content string, planMode bool, thinkingBudget *int, maxSteps *int, attachments ...message.Attachment) (<-chan AgentEvent, error)
	Cancel(sessionID string)
	CancelWithReason(sessionID string, reason string)
	Update(agentName config.AgentName, modelID models.ModelID) (models.Model, error)
	ClearAllSessionProviders()
	GetTools() []tools.BaseTool
	Shutdown()
}

type agent struct {
	broker        *pubsub.Broker[AgentEvent]
	sessions      session.Service
	messages      message.Service
	permissions   permission.Service
	storageConfig session.Config

	agentName config.AgentName
	tools     []tools.BaseTool
	provider  interfaces.Provider

	titleProvider interfaces.Provider

	sessionProviders sync.Map // Maps session ID to interfaces.Provider
	activeContexts   sync.Map // Maps session ID to context.CancelFunc for cancellation
	sessionStates    sync.Map // Maps session ID to SessionState for debugging

	accumulator *MessageAccumulator // In-memory message accumulator

	callbackExecutor interfaces.CallbackExecutor // Executes post-tool callbacks

	ctx    context.Context
	cancel context.CancelFunc
}

// NewAgent creates a new agent instance with a new broker
func NewAgent(
	agentName config.AgentName,
	sessions session.Service,
	messages message.Service,
	agentTools []tools.BaseTool,
	storageConfig session.Config,
	permissions ...permission.Service, // Optional for backward compatibility
) (Service, error) {
	return NewAgentWithBroker(agentName, sessions, messages, agentTools, storageConfig, nil, permissions...)
}

// NewAgentWithBroker creates a new agent instance with an optional shared broker
// If broker is nil, a new broker is created. This allows subagents to share the parent's broker.
func NewAgentWithBroker(
	agentName config.AgentName,
	sessions session.Service,
	messages message.Service,
	agentTools []tools.BaseTool,
	storageConfig session.Config,
	broker *pubsub.Broker[AgentEvent],
	permissions ...permission.Service,
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

	ctx, cancel := context.WithCancel(context.Background())

	// Create message accumulator (no periodic flushing)
	accumulator := NewMessageAccumulator(messages)

	// Extract permissions service (may be nil)
	var perms permission.Service
	if len(permissions) > 0 {
		perms = permissions[0]
	}

	// Create new broker if not provided (for subagents sharing parent broker)
	if broker == nil {
		broker = pubsub.NewBroker[AgentEvent]()
	}

	agent := &agent{
		broker:           broker,
		agentName:        agentName,
		provider:         agentProvider,
		messages:         messages,
		sessions:         sessions,
		permissions:      perms,
		storageConfig:    storageConfig,
		tools:            agentTools,
		titleProvider:    titleProvider,
		sessionProviders: sync.Map{},
		activeContexts:   sync.Map{},
		accumulator:      accumulator,
		ctx:              ctx,
		cancel:           cancel,
	}

	// Create callback executor with factory function (if permissions service is provided)
	if perms != nil {
		agent.callbackExecutor = callbacks.NewExecutor(sessions, perms, messages, agent.createSubAgentForCallback)
	}

	// Inject agent reference into task tool (resolves circular dependency)
	for _, tool := range agentTools {
		if taskTool, ok := tool.(*taskTool); ok {
			taskTool.SetAgent(agent)
			break
		}
	}

	// Start session deletion cleanup goroutine
	go agent.handleSessionEvents()

	return agent, nil
}

func (a *agent) Model() models.Model {
	return a.provider.Model()
}

func (a *agent) GetBroker() *pubsub.Broker[AgentEvent] {
	return a.broker
}

func (a *agent) GetTools() []tools.BaseTool {
	return a.tools
}

func (a *agent) Subscribe(ctx context.Context) <-chan pubsub.Event[AgentEvent] {
	return a.broker.Subscribe(ctx)
}

func (a *agent) Cancel(sessionID string) {
	a.CancelWithReason(sessionID, "unknown")
}

func (a *agent) CancelWithReason(sessionID, reason string) {
	// Cancel regular requests
	cancelFunc, exists := a.activeContexts.LoadAndDelete(sessionID)
	if !exists {
		// Nothing to cancel - agent not running or already completed
		return
	}

	if cancel, ok := cancelFunc.(context.CancelFunc); ok {
		cancel()
		a.setSessionState(sessionID, SessionStateCancelled)
	}
}

// setSessionState sets the state of a session and logs the transition
func (a *agent) setSessionState(sessionID string, state SessionState) {
	a.sessionStates.Store(sessionID, state)
}

// getSessionState retrieves the current state of a session
func (a *agent) getSessionState(sessionID string) (SessionState, bool) {
	if state, exists := a.sessionStates.Load(sessionID); exists {
		if s, ok := state.(SessionState); ok {
			return s, true
		}
	}
	return "", false
}


func (a *agent) generateTitle(ctx context.Context, sessionID, content string) error {
	if content == "" {
		return nil
	}
	if a.titleProvider == nil {
		return nil
	}
	sess, err := a.sessions.Get(ctx, sessionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Session was deleted - exit silently
			logging.Debug("Session deleted during title generation, skipping", "sessionID", sessionID)
			return nil
		}
		return err
	}
	ctx = context.WithValue(ctx, tools.SessionIDContextKey, sessionID)

	// Add session storage directory to context
	ctx = tools.SetSessionStorageContext(ctx, sess.ID, a.storageConfig)
	// Add full session object to context
	ctx = context.WithValue(ctx, interfaces.SessionContextKey, sess)

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

	// Enforce maximum title length to prevent UI layout issues
	sess.Title = session.TruncateTitle(title)
	_, err = a.sessions.Save(ctx, sess)
	return err
}

func (a *agent) err(err error) AgentEvent {
	return AgentEvent{
		Type:  AgentEventTypeError,
		Error: err,
	}
}

func (a *agent) Run(ctx context.Context, sessionID, content string, attachments ...message.Attachment) (<-chan AgentEvent, error) {
	return a.RunWithPlanMode(ctx, sessionID, content, false, nil, nil, attachments...)
}

func (a *agent) RunWithPlanMode(ctx context.Context, sessionID, content string, planMode bool, thinkingBudget, maxSteps *int, attachments ...message.Attachment) (<-chan AgentEvent, error) {
	if !a.provider.Model().SupportsAttachments && attachments != nil {
		attachments = nil
	}
	events := make(chan AgentEvent, 10) // Buffered channel for better streaming

	genCtx, cancel := context.WithCancel(ctx)

	// Set up routing context BEFORE storing cancel func
	sess, err := a.sessions.Get(genCtx, sessionID)
	if err != nil {
		cancel()
		return nil, err
	}

	// Route subagent events to parent, otherwise to self
	routeTo := sessionID
	if sess.ParentSessionID != "" {
		routeTo = sess.ParentSessionID
	}
	genCtx = withSessionRouting(genCtx, routeTo, sessionID)

	// Store cancel function for potential cancellation
	a.activeContexts.Store(sessionID, cancel)

	// Set session state to processing
	a.setSessionState(sessionID, SessionStateProcessing)

	// Add plan mode to context
	if planMode {
		genCtx = context.WithValue(genCtx, interfaces.PlanModeContextKey, true)
	}

	// Add thinking budget to context
	if thinkingBudget != nil {
		genCtx = context.WithValue(genCtx, interfaces.ThinkingBudgetContextKey, thinkingBudget)
	}

	// Subscribe to agent events for real-time streaming
	subscription := a.Subscribe(genCtx)

	// Track this operation in the session's WaitGroup
	wg := trackSessionOperation(sessionID)

	go func() {
		defer func() {
			// Mark operation as complete
			wg.Done()

			// Check if session was cancelled or completed normally
			state, _ := a.getSessionState(sessionID)
			if state != SessionStateCancelled {
				a.setSessionState(sessionID, SessionStateCompleted)
			}

			a.activeContexts.Delete(sessionID)
			cancel()
			close(events)
		}()

		defer logging.RecoverPanic("agent.Run", func() {
			events <- a.err(fmt.Errorf("panic while running the agent"))
		})

		var attachmentParts []message.ContentPart
		for _, attachment := range attachments {
			attachmentParts = append(attachmentParts, message.BinaryContent{Path: attachment.FilePath, MIMEType: attachment.MimeType, Data: attachment.Content})
		}

		result := a.processGeneration(genCtx, sessionID, content, attachmentParts, maxSteps)
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
			case <-genCtx.Done():
				return
			case event, ok := <-subscription:
				if !ok {
					return
				}
				// Only forward intermediate events for this specific session (not final completion events)
				// Forward events that originated from OR are routed to this session
				shouldForward := (event.Payload.SessionID == sessionID ||
					event.Payload.RouteTo == sessionID ||
					event.Payload.Message.SessionID == sessionID) && !event.Payload.Done

				if shouldForward {
					select {
					case events <- event.Payload:
					case <-genCtx.Done():
						return
					}
				}
			}
		}
	}()

	return events, nil
}

func (a *agent) processGeneration(ctx context.Context, sessionID, content string, attachmentParts []message.ContentPart, maxSteps *int) AgentEvent {
	// Starting message processing for session
	_ = config.Get()

	// Load conversation history
	msgs, err := a.loadConversationHistory(ctx, sessionID)
	if err != nil {
		return a.err(err)
	}

	// Start title generation asynchronously if this is the first message
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

	userMsg, err := a.createUserMessage(ctx, sessionID, content, attachmentParts)
	if err != nil {
		return a.err(fmt.Errorf("failed to create user message: %w", err))
	}

	// Emit user message created event so frontend can track it
	_ = a.Publish(ctx, pubsub.EventType(sessionID), AgentEvent{
		Type:    AgentEventTypeUserMessageCreated,
		Message: userMsg,
	})

	conversationTurn := 1
	for {
		// Check for cancellation before each iteration
		select {
		case <-ctx.Done():
			return a.err(ctx.Err())
		default:
			// Continue processing
		}

		// RELOAD message history from database at the start of each turn
		// This ensures we always have the latest state including any tool results
		// Database is the single source of truth for message history
		msgHistory, err := a.loadConversationHistory(ctx, sessionID)
		if err != nil {
			return a.err(fmt.Errorf("failed to reload conversation history: %w", err))
		}

		agentMessage, toolResults, err := a.streamAndHandleEvents(ctx, sessionID, msgHistory)
		if err != nil {
			// Stream processing failed for session
			if errors.Is(err, context.DeadlineExceeded) {
				logging.Error("Agent timeout exceeded", "sessionID", sessionID, "conversationTurn", conversationTurn)
				a.finishMessage(&agentMessage)
				return a.err(ErrRequestCancelledTimeout)
			}
			if errors.Is(err, context.Canceled) {
				a.finishMessage(&agentMessage)
				return a.err(ErrRequestCancelled)
			}
			return a.err(fmt.Errorf("failed to process events: %w", err))
		}

		// Tool execution already persists to DB via executeToolsWithDependencies
		// No need to manually append - database is the source of truth
		// Just check if tools were used to decide whether to continue the conversation loop
		hasTools := (agentMessage.FinishReason() == message.FinishReasonToolUse) && toolResults != nil

		// If agent used tools, continue to next turn (DB reload will get the latest state)
		if hasTools {
			conversationTurn++

			// Check if max steps limit has been reached
			if maxSteps != nil && conversationTurn > *maxSteps {
				a.finishMessage(&agentMessage)
				return a.err(fmt.Errorf("maximum iteration limit (%d steps) reached", *maxSteps))
			}

			continue
		}

		// Agent finished with no tools - conversation complete
		routeTo, origin, parentToolCallID := getSessionRouting(ctx)
		finalEvent := AgentEvent{
			Type:             AgentEventTypeResponse,
			Message:          agentMessage,
			SessionID:        origin,
			RouteTo:          routeTo,
			ParentToolCallID: parentToolCallID,
			Done:             true,
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
	if ctx.Value(interfaces.PlanModeContextKey) != nil {
		planModeContent, err := prompt.LoadPrompt(ctx, "plan_mode", nil)
		if err != nil {
			return message.Message{}, fmt.Errorf("failed to load plan mode prompt: %w", err)
		}
		messageContent = content + "\n\n<system-reminder>\n" + planModeContent + "\n</system-reminder>"
	}

	parts := []message.ContentPart{message.TextContent{Text: messageContent}}
	parts = append(parts, attachmentParts...)
	userMsg, err := a.messages.Create(ctx, sessionID, message.CreateMessageParams{
		Role:  message.User,
		Parts: parts,
	})
	return userMsg, err
}

// loadConversationHistory loads all messages for a session
func (a *agent) loadConversationHistory(ctx context.Context, sessionID string) ([]message.Message, error) {
	msgs, err := a.messages.List(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}

	// Filter out messages with excluded callback results
	filteredMsgs := make([]message.Message, 0, len(msgs))
	for i := range msgs {
		if shouldExcludeMessage(msgs[i]) {
			continue
		}
		filteredMsgs = append(filteredMsgs, msgs[i])
	}
	msgs = filteredMsgs

	return msgs, nil
}

// shouldExcludeMessage checks if a message contains ONLY callback results marked for exclusion from context
func shouldExcludeMessage(msg message.Message) bool {
	// Only check Tool messages for excluded callback results
	if msg.Role != message.Tool {
		return false
	}

	// Get both tool results and callback results
	toolResults := msg.ToolResults()
	callbackResults := msg.CallbackResults()

	// If there are any tool_results (from normal tool execution), never exclude the message
	// Tool results must always be paired with their tool_use blocks
	if len(toolResults) > 0 {
		return false
	}

	// Only exclude if message contains exclusively callback results with ExcludeFromContext=true
	if len(callbackResults) == 0 {
		return false
	}

	// Check if all callback results are marked for exclusion
	for i := range callbackResults {
		if !callbackResults[i].ExcludeFromContext {
			// If any callback result is NOT excluded, keep the message
			return false
		}
	}

	// All callback results are excluded, so exclude the entire message
	return true
}

func (a *agent) streamAndHandleEvents(ctx context.Context, sessionID string, msgHistory []message.Message) (assistantMsg message.Message, toolResultMsg *message.Message, err error) {
	var usage interfaces.TokenUsage
	ctx = context.WithValue(ctx, tools.SessionIDContextKey, sessionID)

	// Check authentication before processing
	// Skip authentication check if credentials service is not available (test environment)
	credentialsAvailable := config.GetAPICredentials() != nil
	if credentialsAvailable {
		authenticated, _, authErr := provider.IsAuthenticated(ctx, "")
		if authErr != nil {
			return message.Message{}, nil, fmt.Errorf("failed to check authentication: %w", authErr)
		}
		if !authenticated {
			return message.Message{}, nil, fmt.Errorf("authentication required: please configure your LLM provider credentials using /login or environment variables")
		}
	}

	// Get session and add working directory to context
	sess, err := a.sessions.Get(ctx, sessionID)
	if err != nil {
		return message.Message{}, nil, fmt.Errorf("failed to load session %s: %w", sessionID, err)
	}
	// Add session storage directory to context
	ctx = tools.SetSessionStorageContext(ctx, sess.ID, a.storageConfig)
	// Add full session object to context for tools that need browser mode, etc.
	ctx = context.WithValue(ctx, interfaces.SessionContextKey, sess)

	// Get cached session-specific provider
	sessionProvider, err := a.getOrCreateSessionProvider(ctx, sessionID, &sess)
	if err != nil {
		return message.Message{}, nil, fmt.Errorf("failed to get session provider: %w", err)
	}

	// Filter tools based on plan mode
	availableTools := a.tools
	if ctx.Value(interfaces.PlanModeContextKey) != nil {
		availableTools = filterToolsForPlanMode(a.tools)
	}

	eventChan := sessionProvider.StreamResponse(ctx, msgHistory, availableTools)

	assistantMsg, err = a.messages.Create(ctx, sessionID, message.CreateMessageParams{
		Role:  message.Assistant,
		Parts: []message.ContentPart{},
		Model: sessionProvider.Model().ID,
	})
	if err != nil {
		return assistantMsg, nil, fmt.Errorf("failed to create assistant message: %w", err)
	}

	// Store initial assistant message in accumulator
	a.accumulator.Store(&assistantMsg)

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
		eventUsage, processErr := a.processEvent(ctx, sessionID, &assistantMsg, event)
		if processErr != nil {
			a.finishMessage(&assistantMsg)
			return assistantMsg, nil, processErr
		}
		// Capture usage from completion event
		if eventUsage != nil {
			usage = *eventUsage
		}
		if ctx.Err() != nil {
			a.finishMessage(&assistantMsg)
			return assistantMsg, nil, ctx.Err()
		}
	}

	// Store token usage in message after finalization
	if usage.InputTokens > 0 || usage.OutputTokens > 0 {
		cost := a.calculateMessageCost(sessionProvider.Model(), usage)
		assistantMsg.InputTokens = usage.InputTokens
		assistantMsg.OutputTokens = usage.OutputTokens
		assistantMsg.CacheCreationTokens = usage.CacheCreationTokens
		assistantMsg.CacheReadTokens = usage.CacheReadTokens
		assistantMsg.Cost = cost

		// Update message in database with token data
		if updateErr := a.messages.Update(ctx, assistantMsg); updateErr != nil {
			logging.Error("Failed to update message with token usage", "error", updateErr, "messageID", assistantMsg.ID)
		}
	}

	toolCalls := assistantMsg.ToolCalls()
	if len(toolCalls) == 0 {
		return assistantMsg, nil, nil
	}

	// Filter to only execute finished tool calls
	// Tool calls that haven't received ContentBlockStop from the API should not be executed
	// as they have incomplete parameters and were never fully sent in the API stream
	var finishedToolCalls []message.ToolCall
	for _, tc := range toolCalls {
		if tc.Finished {
			finishedToolCalls = append(finishedToolCalls, tc)
		} else {
			logging.Warn("Skipping unfinished tool call - never received ContentBlockStop",
				"toolCallID", tc.ID,
				"toolName", tc.Name,
				"sessionID", sessionID)
		}
	}

	if len(finishedToolCalls) == 0 {
		// No finished tools to execute
		return assistantMsg, nil, nil
	}

	// Execute all finished tool calls with dependency awareness
	toolResults, toolErr := a.executeToolsWithDependencies(ctx, sessionID, finishedToolCalls, assistantMsg)

	// Always create tool result message, even if some tools failed
	// This prevents orphaned tool_use messages that cause API rejection
	msg, err := a.messages.Create(context.Background(), assistantMsg.SessionID, message.CreateMessageParams{
		Role:  message.Tool,
		Parts: toolResults,
	})
	if err != nil {
		return assistantMsg, nil, fmt.Errorf("failed to create tool result message: %w", err)
	}

	// Log tool execution errors but don't fail the entire flow
	if toolErr != nil {
		logging.Error("Some tools failed during execution", "error", toolErr, "sessionID", sessionID)
	}

	// Execute callbacks NOW that tool_result message is saved to database
	// This ensures proper message ordering: Assistant(tool_use) → Tool(result) → User(injected)
	if a.callbackExecutor != nil {
		sessionStorageDir, _ := tools.GetSessionStorageDirectory(ctx)
		messageID, _ := ctx.Value(tools.MessageIDContextKey).(string)

		// Use WaitGroup to ensure all callbacks complete before returning
		// This prevents race condition where agent checks for injected messages before callbacks finish
		var callbackWg sync.WaitGroup

		for _, toolCall := range toolCalls {
			// Get tool result for this call
			var toolResult interfaces.ToolResponse
			for _, result := range toolResults {
				if tr, ok := result.(message.ToolResult); ok && tr.ToolCallID == toolCall.ID {
					toolResult = interfaces.ToolResponse{
						Content:  tr.Content,
						Metadata: tr.Metadata,
						IsError:  tr.IsError,
					}
					break
				}
			}

			// Skip callbacks for failed tools
			if toolResult.IsError {
				continue
			}

			// Load callbacks for this tool
			sessionCallbacks, err := a.getSessionCallbacks(ctx, sessionID, toolCall.Name)
			if err != nil {
				logging.Error("Failed to load session callbacks", "error", err, "sessionID", sessionID, "tool", toolCall.Name)
				continue
			}

			if len(sessionCallbacks) == 0 {
				continue
			}

			// Track this callback goroutine
			callbackWg.Add(1)

			// Execute all callbacks asynchronously but sequentially
			// This ensures: (1) Callbacks execute in parallel per tool, (2) Callbacks maintain execution order within each tool
			callbackCtx := interfaces.CallbackContext{
				SessionID:         sessionID,
				MessageID:         messageID,
				ToolCall:          interfaces.ToolCall{ID: toolCall.ID, Name: toolCall.Name, Input: toolCall.Input},
				ToolResult:        toolResult,
				SessionStorageDir: sessionStorageDir,
			}

			go func(callbacks []interfaces.CallbackConfig, cbCtx interfaces.CallbackContext, toolName string) {
				defer callbackWg.Done()

				// Execute callbacks sequentially in order
				for i := range callbacks {
					result, err := a.callbackExecutor.Execute(context.Background(), callbacks[i], cbCtx)
					if err != nil {
						logging.Error("Callback execution failed", "tool", toolName, "callback", callbacks[i].Name, "error", err)
					} else if !result.Success {
						logging.Warn("Callback completed with errors", "tool", toolName, "callback", callbacks[i].Name, "error", result.Error)
					}
				}
			}(sessionCallbacks, callbackCtx, toolCall.Name)
		}

		// Wait for all callbacks to complete before returning
		// This ensures injected messages are saved to database before agent checks for them
		callbackWg.Wait()
	}

	return assistantMsg, &msg, nil
}

func (a *agent) finishMessage(msg *message.Message) {
	msg.AddFinish(message.FinishReasonCanceled)

	// Store in accumulator
	a.accumulator.Store(msg)

	// Finalize with the given finish reason - this ensures immediate flush
	_ = a.accumulator.FinalizeMessage(msg.ID, message.FinishReasonCanceled)
}

func (a *agent) processEvent(ctx context.Context, sessionID string, assistantMsg *message.Message, event interfaces.ProviderEvent) (*interfaces.TokenUsage, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		// Continue processing.
	}

	switch event.Type {
	case interfaces.EventContentStart:
		// Content block starting - no action needed
		// The actual content will arrive via EventContentDelta events
		return &interfaces.TokenUsage{}, nil
	case interfaces.EventThinkingDelta:
		// Claude thinking delta received
		assistantMsg.AppendReasoningContent(event.Thinking)

		// Store in accumulator without immediate DB update
		a.accumulator.Store(assistantMsg)

		// Update thinking content in accumulator
		if err := a.accumulator.UpdateThinking(assistantMsg.ID, event.Thinking); err != nil {
			return nil, err
		}

		// Publish thinking event for real-time streaming
		err := a.Publish(ctx, pubsub.CreatedEvent, AgentEvent{
			Type:      AgentEventTypeThinking,
			Message:   *assistantMsg,
			SessionID: sessionID,
			Thinking:  event.Thinking,
		})
		return nil, err
	case interfaces.EventContentDelta:
		assistantMsg.AppendContent(event.Content)

		// Store in accumulator without immediate DB update
		a.accumulator.Store(assistantMsg)

		// Update content in accumulator
		if err := a.accumulator.UpdateContent(assistantMsg.ID, event.Content); err != nil {
			return nil, err
		}

		// Publish content delta event for real-time streaming
		err := a.Publish(ctx, pubsub.CreatedEvent, AgentEvent{
			Type:      AgentEventTypeContentDelta,
			Message:   *assistantMsg,
			SessionID: sessionID,
			Content:   event.Content, // Send only the delta, not accumulated content
		})
		return nil, err
	case interfaces.EventContentStop:
		// Content block finished - no action needed
		// Final content state is already accumulated via EventContentDelta
		// EventComplete will follow with finish reason and token usage
		return &interfaces.TokenUsage{}, nil
	case interfaces.EventToolUseStart:
		assistantMsg.AddToolCall(*event.ToolCall)

		// Store in accumulator
		a.accumulator.Store(assistantMsg)

		// Flush immediately for tool events (they're less frequent)
		if err := a.accumulator.FlushMessage(assistantMsg.ID); err != nil {
			return nil, err
		}

		// Capture snapshot of tool call state at this moment to prevent race conditions
		// This ensures the SSE broadcast sees Finished: false even if the tool completes
		// before the event is broadcast
		toolCallSnapshot := message.ToolCall{
			ID:       event.ToolCall.ID,
			Name:     event.ToolCall.Name,
			Input:    event.ToolCall.Input,
			Type:     event.ToolCall.Type,
			Finished: false, // Explicitly capture that tool is starting
		}

		// Publish tool start event for real-time streaming
		err := a.Publish(ctx, pubsub.CreatedEvent, AgentEvent{
			Type:             AgentEventTypeResponse,
			Message:          *assistantMsg,
			SessionID:        sessionID,
			ToolCallSnapshot: &toolCallSnapshot,
		})
		return nil, err
	case interfaces.EventToolUseDelta:
		// Append partial tool input to the message
		if err := assistantMsg.AppendToolCallInput(event.ToolCall.ID, event.ToolCall.Input); err != nil {
			return nil, fmt.Errorf("failed to append tool call input for tool %s: %w", event.ToolCall.ID, err)
		}

		// Store in accumulator without immediate flush
		// The accumulator will batch DB updates for performance
		a.accumulator.Store(assistantMsg)

		// Publish tool parameter delta event for real-time streaming
		err := a.Publish(ctx, pubsub.CreatedEvent, AgentEvent{
			Type:       AgentEventTypeToolParameterDelta,
			Message:    *assistantMsg,
			SessionID:  sessionID,
			ToolCallID: event.ToolCall.ID,
			Content:    event.ToolCall.Input, // Send the delta JSON
		})
		return nil, err
	case interfaces.EventToolUseStop:
		assistantMsg.FinishToolCall(event.ToolCall.ID)

		// Store in accumulator
		a.accumulator.Store(assistantMsg)

		// Flush immediately for tool events
		if err := a.accumulator.FlushMessage(assistantMsg.ID); err != nil {
			return nil, err
		}

		// Find the completed tool call to capture its final state
		var toolCallSnapshot *message.ToolCall
		for _, tc := range assistantMsg.ToolCalls() {
			if tc.ID == event.ToolCall.ID {
				// Capture snapshot of completed tool call with full input and Finished: true
				snapshot := message.ToolCall{
					ID:       tc.ID,
					Name:     tc.Name,
					Input:    tc.Input, // Full accumulated input
					Type:     tc.Type,
					Finished: true, // Explicitly capture that tool is finished
				}
				toolCallSnapshot = &snapshot
				break
			}
		}

		// Publish tool completion event for real-time streaming
		err := a.Publish(ctx, pubsub.CreatedEvent, AgentEvent{
			Type:             AgentEventTypeResponse,
			Message:          *assistantMsg,
			SessionID:        sessionID,
			ToolCallSnapshot: toolCallSnapshot,
		})
		return nil, err
	case interfaces.EventWarning:
		// Log warning but continue processing
		if event.Error != nil {
			logging.Warn("Provider warning: %v", event.Error)
		}
		return &interfaces.TokenUsage{}, nil
	case interfaces.EventError:
		// Store current state before error
		a.accumulator.Store(assistantMsg)

		if err := a.accumulator.FlushMessage(assistantMsg.ID); err != nil {
			logging.Error("Failed to flush message on error: %v", err)
		}

		if errors.Is(event.Error, context.Canceled) {
			// Event processing canceled for session
			return nil, context.Canceled
		}
		logging.Error(event.Error.Error())
		return nil, event.Error
	case interfaces.EventComplete:
		// Note: We rely on manual accumulation during streaming (AddToolCall/AppendToolCallInput/FinishToolCall)
		// rather than SDK's accumulated tool calls, since we unmarshal delta strings before concatenation.
		// The SDK's toolCalls work too, but manual accumulation is more reliable for our use case.
		assistantMsg.AddFinish(event.Response.FinishReason)

		// Store thinking blocks from the response
		for _, thinkingBlock := range event.Response.ThinkingBlocks {
			assistantMsg.AddThinkingBlock(thinkingBlock.Thinking, thinkingBlock.Signature)
		}
		for _, redactedBlock := range event.Response.RedactedThinkingBlocks {
			assistantMsg.AddRedactedThinkingBlock(redactedBlock.Data)
		}

		// Store final state in accumulator
		a.accumulator.Store(assistantMsg)

		// Finalize message with the finish reason from response
		// The finish reason is already set by AddFinish
		if err := a.accumulator.FinalizeMessage(assistantMsg.ID, event.Response.FinishReason); err != nil {
			return nil, fmt.Errorf("failed to finalize message: %w", err)
		}

		// Track session-level usage
		if err := a.TrackUsage(ctx, sessionID, a.provider.Model(), event.Response.Usage); err != nil {
			return nil, err
		}

		// Return usage so caller can store per-message tokens
		return &event.Response.Usage, nil
	}

	// Unknown event type - should not happen if all event types are handled
	return nil, fmt.Errorf("unhandled event type: %v", event.Type)
}

// calculateMessageCost calculates the cost for a single message based on token usage
func (a *agent) calculateMessageCost(model models.Model, usage interfaces.TokenUsage) float64 {
	cost := model.CostPer1MInCached/1e6*float64(usage.CacheCreationTokens) +
		model.CostPer1MOutCached/1e6*float64(usage.CacheReadTokens) +
		model.CostPer1MIn/1e6*float64(usage.InputTokens) +
		model.CostPer1MOut/1e6*float64(usage.OutputTokens)
	return cost
}

func (a *agent) TrackUsage(ctx context.Context, sessionID string, model models.Model, usage interfaces.TokenUsage) error {
	sess, err := a.sessions.Get(ctx, sessionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Session was deleted - exit silently
			logging.Debug("Session deleted during usage tracking, skipping", "sessionID", sessionID)
			return nil
		}
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

	modelProvider, err := createAgentProvider(agentName)
	if err != nil {
		return models.Model{}, fmt.Errorf("failed to create provider for model %s: %w", modelID, err)
	}

	a.provider = modelProvider

	// Update title provider if this is the main agent
	// Since title provider always uses AgentMain config, we need to update it
	// whenever AgentMain model changes
	if agentName == config.AgentMain {
		// Update title provider if it exists
		if a.titleProvider != nil {
			titleProvider, err := createAgentProvider(config.AgentMain)
			if err != nil {
				logging.Warn("Failed to update title provider", "error", err)
			} else {
				a.titleProvider = titleProvider
			}
		}
	}

	return a.provider.Model(), nil
}

// setIfEmpty sets target to source if target is empty and source is not
func setIfEmpty(target *string, source string) {
	if *target == "" && source != "" {
		*target = source
	}
}

// Publish overrides the embedded Broker.Publish to auto-populate routing fields from context
func (a *agent) Publish(ctx context.Context, t pubsub.EventType, event AgentEvent) error {
	routeTo, origin, parentToolCallID := getSessionRouting(ctx)

	// Auto-populate from context if not already set
	setIfEmpty(&event.RouteTo, routeTo)
	setIfEmpty(&event.SessionID, origin)
	setIfEmpty(&event.ParentToolCallID, parentToolCallID)

	return a.broker.Publish(ctx, t, event)
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
		"ReadText":     true,
		"Grep":         true,
		"Glob":         true,
		"TodoWrite":    true,
		"ExitPlanMode": true,
		"ReadMedia":    true,
		"WebFetch":     true,
		"Search":       true,
		"Task":         true,
	}

	return allowedTools[toolName]
}

// getAPIKeyWithFallback attempts to get API key from database first, then falls back to environment variables
func getAPIKeyWithFallback(ctx context.Context, providerName models.ModelProvider) string {
	// Try database first
	credentialsService := config.GetAPICredentials()
	if credentialsService != nil {
		dbKey, err := credentialsService.GetAPIKey(ctx, providerName)
		if err == nil && dbKey != "" {
			return dbKey
		}
	}

	// Try environment variable fallback for providers that support it
	var envVar string
	switch providerName {
	case models.ProviderGemini:
		envVar = "GEMINI_API_KEY"
	case models.ProviderOpenRouter:
		envVar = "OPENROUTER_API_KEY"
	case models.ProviderGROQ:
		envVar = "GROQ_API_KEY"
	case models.ProviderXAI:
		envVar = "XAI_API_KEY"
	}


	if envVar != "" {
		if envAPIKey := os.Getenv(envVar); envAPIKey != "" {
			return envAPIKey
		}
	}

	// Warn for non-OAuth providers that need API keys
	if providerName != models.ProviderAnthropic && providerName != models.ProviderOpenAI {
		logging.Warn("No API key found in database or environment for provider", "provider", providerName)
	}

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
			Model:     "claude-sonnet-4-5",
			MaxTokens: 4096,
		}
	}

	// Check user's preferred provider if available
	userPrefs := config.GetUserPreferences()
	if userPrefs != nil {
		// Note: We validate the user's preferred provider exists, but currently
		// we always use the model's default provider regardless of user preference
		_, _ = userPrefs.GetPreferredProvider(ctx)
	}
	model, ok := models.SupportedModels[agentConfig.Model]
	if !ok {
		return nil, fmt.Errorf("model %s not supported", agentConfig.Model)
	}
	// Get API key - try database first, then environment variables
	apiKey := getAPIKeyWithFallback(ctx, model.Provider)

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
		return nil, fmt.Errorf("could not create provider: %w", err)
	}

	return agentProvider, nil
}

func createSessionProvider(ctx context.Context, agentName config.AgentName, sess *session.Session, storageConfig session.Config, thinkingBudget *int) (interfaces.Provider, error) {
	// Try to get agent config from database first
	agentConfig, err := config.GetAgentFromDatabase(ctx, agentName)
	if err != nil {
		// Fall back to default agent config
		logging.Warn("Failed to get agent config from database for session, using default", "error", err, "agent", agentName)
		// Use Claude as default model if database not available
		agentConfig = config.Agent{
			Model:     "claude-sonnet-4-5",
			MaxTokens: 4096,
		}
	}

	model, ok := models.SupportedModels[agentConfig.Model]
	if !ok {
		return nil, fmt.Errorf("model %s not supported", agentConfig.Model)
	}

	// Get API key - try database first, then environment variables
	apiKey := getAPIKeyWithFallback(ctx, model.Provider)

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

	// Add server URL for file access from config
	cfg := config.Get()
	sessionVars["server_url"] = cfg.BaseURL

	// Get system prompt with session variables and custom prompt support
	customPrompt := ""
	promptMode := "default"
	if sess != nil {
		customPrompt = sess.CustomSystemPrompt
		promptMode = sess.PromptMode
	}
	systemPrompt, err := prompt.GetAgentPromptWithVars(ctx, agentName, model.Provider, sessionVars, customPrompt, promptMode)
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
		anthropicOpts := []provider.AnthropicOption{
			provider.WithAnthropicThinkingBudgetFn(provider.DefaultThinkingBudgetFn),
			provider.WithAnthropicInterleavedThinking(),
		}
		// Add explicit thinking budget if provided
		if thinkingBudget != nil {
			anthropicOpts = append(anthropicOpts, provider.WithExplicitThinkingBudget(thinkingBudget))
		}
		opts = append(
			opts,
			provider.WithAnthropicOptions(anthropicOpts...),
		)
	}
	sessionProvider, err := provider.NewProvider(
		model.Provider,
		opts...,
	)
	if err != nil {
		return nil, fmt.Errorf("could not create session provider: %w", err)
	}

	return sessionProvider, nil
}

func (a *agent) getOrCreateSessionProvider(ctx context.Context, sessionID string, sess *session.Session) (interfaces.Provider, error) {
	// Extract thinking budget from context if present
	var thinkingBudget *int
	if budgetValue := ctx.Value(interfaces.ThinkingBudgetContextKey); budgetValue != nil {
		thinkingBudget = budgetValue.(*int)
	}

	// If explicit thinking budget is provided, skip caching and create a new provider
	// This ensures request-specific thinking budgets are used correctly
	if thinkingBudget != nil {
		sessionProvider, err := createSessionProvider(ctx, a.agentName, sess, a.storageConfig, thinkingBudget)
		if err != nil {
			logging.Error("Failed to create session provider with thinking budget", "sessionID", sessionID, "budget", *thinkingBudget, "error", err)
			return nil, fmt.Errorf("failed to create session provider: %w", err)
		}
		logging.Debug("Created non-cached provider with explicit thinking budget", "sessionID", sessionID, "budget", *thinkingBudget)
		return sessionProvider, nil
	}

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

	// Create new session provider (no explicit thinking budget, will be cached)
	// Creating new session provider
	sessionProvider, err := createSessionProvider(ctx, a.agentName, sess, a.storageConfig, nil)
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
	// Shutdown message accumulator first to flush pending messages
	if a.accumulator != nil {
		a.accumulator.Shutdown()
	}
	a.cancel()
}

func (a *agent) handleSessionEvents() {
	eventsChan := a.sessions.Subscribe(a.ctx)

	for event := range eventsChan {
		if event.Type == pubsub.DeletedEvent {
			sessionID := event.Payload.ID
			// Remove cached provider for deleted session
			a.sessionProviders.LoadAndDelete(sessionID)

			// Also flush any pending messages for this session
			// Note: This is a best-effort cleanup since we don't track messages by session
			// The accumulator will automatically clean up after the 5-second delay
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
			if cachedProvider, ok := value.(interfaces.Provider); ok {
				logging.Debug("Found cached provider", "sessionID", sessionID,
					"provider", cachedProvider.Model().Provider,
					"model", cachedProvider.Model().ID)
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

	// Refresh title provider (it always uses AgentMain config)
	if a.titleProvider != nil {
		newTitleProvider, err := createAgentProvider(config.AgentMain)
		if err == nil {
			a.titleProvider = newTitleProvider
		} else {
			logging.Error("Failed to refresh title provider", "error", err)
		}
	}
}

// createSubAgentForCallback is a factory function for creating subagents in callbacks
// This function is passed to the callback executor to avoid circular dependencies
func (a *agent) createSubAgentForCallback(subagentType string) (callbacks.SubAgent, error) {
	// Get tools for the subagent type (reuse the same logic as task tool)
	taskToolInstance := &taskTool{
		sessions:    a.sessions,
		messages:    a.messages,
		permissions: a.permissions,
		agent:       a,
	}

	agentTools := taskToolInstance.getToolsForSubagentType(subagentType)

	// Use parent agent's broker so subagent events are published to the same broker
	parentBroker := a.broker

	// Create subagent with the same permissions as parent
	subAgent, err := NewAgentWithBroker("sub", a.sessions, a.messages, agentTools, a.storageConfig, parentBroker, a.permissions)
	if err != nil {
		return nil, fmt.Errorf("error creating sub-agent: %w", err)
	}

	return &subAgentAdapter{service: subAgent}, nil
}

// subAgentAdapter adapts agent.Service to callbacks.SubAgent interface
type subAgentAdapter struct {
	service Service
}

// Run executes the subagent
func (s *subAgentAdapter) Run(ctx context.Context, sessionID, content string) (<-chan callbacks.AgentEvent, error) {
	done, err := s.service.Run(ctx, sessionID, content)
	if err != nil {
		return nil, err
	}

	// Convert the channel from agent.AgentEvent to callbacks.AgentEvent
	convertedChan := make(chan callbacks.AgentEvent)
	go func() {
		defer close(convertedChan)
		for event := range done {
			convertedChan <- callbacks.AgentEvent{
				Error:   event.Error,
				Content: event.Content,
				Message: &messageAdapter{msg: event.Message},
			}
		}
	}()

	return convertedChan, nil
}

// Shutdown stops the subagent
func (s *subAgentAdapter) Shutdown() {
	s.service.Shutdown()
}

// messageAdapter adapts message.Message to callbacks.Message interface
type messageAdapter struct {
	msg message.Message
}

func (m *messageAdapter) FinishReason() string {
	if m.msg.Role == "" {
		return ""
	}
	return string(m.msg.FinishReason())
}

func (m *messageAdapter) Role() string {
	return string(m.msg.Role)
}

func (m *messageAdapter) Content() callbacks.MessageContent {
	if m.msg.Role == "" {
		return &messageContentAdapter{text: ""}
	}
	// Get the content and convert it to string
	// This handles all content types (TextContent, BinaryContent, etc.)
	content := m.msg.Content()
	return &messageContentAdapter{text: content.String()}
}

// messageContentAdapter adapts message content to callbacks.MessageContent interface
type messageContentAdapter struct {
	text string
}

func (m *messageContentAdapter) String() string {
	return m.text
}
