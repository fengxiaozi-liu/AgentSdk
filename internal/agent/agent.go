package agent

import (
	"context"
	"errors"
	"ferryman-agent/internal/data/llm/client"
	"ferryman-agent/internal/data/llm/models"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	datadb "ferryman-agent/internal/data/db"
	"ferryman-agent/internal/data/logging"
	"ferryman-agent/internal/data/repo"
	"ferryman-agent/internal/memory/history"
	"ferryman-agent/internal/memory/message"
	"ferryman-agent/internal/memory/session"
	"ferryman-agent/internal/prompt"
	providersvc "ferryman-agent/internal/provider"
	"ferryman-agent/internal/pubsub"
	"ferryman-agent/internal/security/permission"
	toolcore "ferryman-agent/internal/tools"
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
	Type      AgentEventType
	Message   message.Message
	Error     error
	SessionID string
	Progress  string
	Done      bool
}

type Service interface {
	pubsub.Suscriber[AgentEvent]
	Model() models.Model
	Run(ctx context.Context, sessionID string, content string, attachments ...message.Attachment) (<-chan AgentEvent, error)
	Cancel(sessionID string)
	IsSessionBusy(sessionID string) bool
	IsBusy() bool
	Update(profileKey string, modelID models.ModelID) (models.Model, error)
	Summarize(ctx context.Context, sessionID string) error
}

type agent struct {
	*pubsub.Broker[AgentEvent]
	config         AgentConfig
	sessions       session.Service //
	messages       message.Service
	history        history.Service
	tools          []toolcore.BaseTool
	prompt         prompt.Service
	promptKey      string
	providerRouter providersvc.Router
	mainAgent      llmAgentRuntime
	titleAgent     *llmAgentRuntime
	summarizeAgent *llmAgentRuntime
	activeRequests sync.Map
}

type llmAgentRuntime struct {
	modelID       models.ModelID
	provider      models.ModelProvider
	systemMessage string
	tools         []toolcore.BaseTool
}

func New(opts ...Option) (Service, error) {
	cfg := DefaultAgentConfig()
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(&cfg); err != nil {
			return nil, err
		}
	}
	return newFromConfig(cfg)
}

func newFromConfig(cfg AgentConfig) (Service, error) {
	if err := normalizeConfig(&cfg); err != nil {
		return nil, err
	}
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}
	prompts := cfg.Prompt.Prompt
	systemPrompt, err := resolveSystemPromptKey(context.Background(), prompts, cfg.Prompt.AgentSystemKey)
	if err != nil {
		return nil, err
	}

	mainRuntime, err := newAgentRuntime("agent", cfg.Provider.AgentProvider, systemPrompt, cfg.Tools)
	if err != nil {
		return nil, err
	}

	titlePrompt, err := resolveSystemPromptKey(context.Background(), prompts, prompt.KeyTitle)
	if err != nil {
		return nil, err
	}
	titleRuntime, err := newAgentRuntime("titleAgent", cfg.Provider.AgentProvider, titlePrompt, make([]toolcore.BaseTool, 0))
	if err != nil {
		return nil, err
	}
	summarizePrompt, err := resolveSystemPromptKey(context.Background(), prompts, prompt.KeySummarizer)
	if err != nil {
		return nil, err
	}
	summarizeRuntime, err := newAgentRuntime("summarizeAgent", cfg.Provider.AgentProvider, summarizePrompt, make([]toolcore.BaseTool, 0))
	if err != nil {
		return nil, err
	}
	return &agent{
		Broker:         pubsub.NewBroker[AgentEvent](),
		config:         cfg,
		sessions:       cfg.Memory.Session,
		messages:       cfg.Memory.Messages,
		history:        cfg.Memory.History,
		tools:          cfg.Tools,
		prompt:         prompts,
		promptKey:      cfg.Prompt.AgentSystemKey,
		providerRouter: cfg.Provider.Router,
		mainAgent:      mainRuntime,
		titleAgent:     &titleRuntime,
		summarizeAgent: &summarizeRuntime,
		activeRequests: sync.Map{},
	}, nil
}

func newAgentRuntime(name string, cfg AgentProvider, systemMessage string, tools []toolcore.BaseTool) (llmAgentRuntime, error) {
	if cfg.ModelID == "" {
		return llmAgentRuntime{}, fmt.Errorf("%s model_id is required", name)
	}
	return llmAgentRuntime{
		modelID:       cfg.ModelID,
		provider:      cfg.Provider,
		systemMessage: systemMessage,
		tools:         tools,
	}, nil
}

func resolveSystemPromptKey(ctx context.Context, service prompt.Service, key string) (string, error) {
	if strings.TrimSpace(key) == "" {
		return "", nil
	}
	if service == nil {
		return "", fmt.Errorf("%w: prompt service is required for %s", prompt.ErrPromptConfigInvalid, key)
	}
	return service.GetSystemPrompt(ctx, key)
}

func normalizeConfig(cfg *AgentConfig) error {
	if strings.TrimSpace(cfg.WorkingDir) == "" {
		if wd, err := os.Getwd(); err == nil {
			cfg.WorkingDir = wd
		}
	}
	if cfg.Prompt.Prompt == nil {
		cfg.Prompt.Prompt = prompt.NewDefault()
	}
	if strings.TrimSpace(cfg.Prompt.AgentSystemKey) == "" {
		cfg.Prompt.AgentSystemKey = prompt.KeyCoder
	}
	if cfg.Memory.Session == nil && cfg.Memory.Messages == nil && cfg.Memory.History == nil {
		memory, err := memoryFromDatabase(datadb.DatabaseConfig{Type: datadb.DatabaseSQLite, AutoMigrate: true})
		if err != nil {
			return err
		}
		cfg.Memory = memory
	}
	return nil
}

func validateConfig(cfg AgentConfig) error {
	if cfg.Memory.Session == nil || cfg.Memory.Messages == nil || cfg.Memory.History == nil {
		return fmt.Errorf("memory session, messages, and history services must be configured together")
	}
	if cfg.Provider.Router == nil {
		return fmt.Errorf("provider router is required")
	}
	if cfg.Provider.AgentProvider.Provider == "" || cfg.Provider.AgentProvider.ModelID == "" {
		return fmt.Errorf("agent provider and model are required")
	}
	if cfg.Prompt.Prompt == nil {
		return fmt.Errorf("prompt service is required")
	}
	return nil
}

func memoryFromDatabase(database datadb.DatabaseConfig) (MemoryConfig, error) {
	dbClient, err := datadb.NewDbClient(database)
	if err != nil {
		return MemoryConfig{}, err
	}
	if database.AutoMigrate {
		if err := dbClient.AutoMigrate(&repo.SessionRecord{}, &repo.MessageRecord{}, &repo.HistoryRecord{}); err != nil {
			return MemoryConfig{}, err
		}
	}
	sessionRepo := repo.NewSessionRepo(dbClient)
	messageRepo := repo.NewMessageRepo(dbClient)
	historyRepo := repo.NewHistoryRepo(dbClient)
	return MemoryConfig{
		Session:  session.NewService(sessionRepo),
		Messages: message.NewService(messageRepo),
		History:  history.NewService(historyRepo),
	}, nil
}

func (a *agent) Model() models.Model {
	return a.modelFor(a.mainAgent)
}

func (a *agent) modelFor(runtime llmAgentRuntime) models.Model {
	return models.ResolveModel(runtime.provider, runtime.modelID)
}

func (a *agent) providerRequest(runtime llmAgentRuntime, messages []message.Message) client.Request {
	return client.Request{
		ModelID:       runtime.modelID,
		Provider:      runtime.provider,
		SystemMessage: runtime.systemMessage,
		Debug:         a.config.Debug,
		Messages:      messages,
		Tools:         runtime.tools,
	}
}

func (a *agent) routeProvider(ctx context.Context, runtime llmAgentRuntime) (providersvc.ProviderClient, error) {
	return a.providerRouter.Route(ctx, providersvc.RouteRequest{
		Provider: runtime.provider,
		ModelID:  runtime.modelID,
	})
}

func (a *agent) sendMessages(ctx context.Context, runtime llmAgentRuntime, messages []message.Message) (*client.Response, error) {
	target, err := a.routeProvider(ctx, runtime)
	if err != nil {
		return nil, err
	}
	return target.SendMessages(ctx, a.providerRequest(runtime, messages))
}

func (a *agent) streamResponse(ctx context.Context, runtime llmAgentRuntime, messages []message.Message) <-chan client.Event {
	target, err := a.routeProvider(ctx, runtime)
	if err != nil {
		ch := make(chan client.Event, 1)
		ch <- client.Event{Type: client.EventError, Error: err}
		close(ch)
		return ch
	}
	return target.StreamResponse(ctx, a.providerRequest(runtime, messages))
}

func (a *agent) Cancel(sessionID string) {
	// Cancel regular requests
	if cancelFunc, exists := a.activeRequests.LoadAndDelete(sessionID); exists {
		if cancel, ok := cancelFunc.(context.CancelFunc); ok {
			logging.InfoPersist(fmt.Sprintf("Request cancellation initiated for session: %s", sessionID))
			cancel()
		}
	}

	// Also check for summarize requests
	if cancelFunc, exists := a.activeRequests.LoadAndDelete(sessionID + "-summarize"); exists {
		if cancel, ok := cancelFunc.(context.CancelFunc); ok {
			logging.InfoPersist(fmt.Sprintf("Summarize cancellation initiated for session: %s", sessionID))
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
	if a.titleAgent == nil {
		return nil
	}
	session, err := a.sessions.Get(ctx, sessionID)
	if err != nil {
		return err
	}
	ctx = context.WithValue(ctx, toolcore.SessionIDContextKey, sessionID)
	parts := []message.ContentPart{message.TextContent{Text: content}}
	response, err := a.sendMessages(ctx, *a.titleAgent, []message.Message{
		{
			Role:  message.User,
			Parts: parts,
		},
	})
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
	if !a.Model().SupportsAttachments && attachments != nil {
		attachments = nil
	}
	events := make(chan AgentEvent)
	if a.IsSessionBusy(sessionID) {
		return nil, ErrSessionBusy
	}

	genCtx, cancel := context.WithCancel(ctx)

	a.activeRequests.Store(sessionID, cancel)
	go func() {
		logging.Debug("Request started", "sessionID", sessionID)
		defer logging.RecoverPanic("agent.Run", func() {
			events <- a.err(fmt.Errorf("panic while running the agent"))
		})
		var attachmentParts []message.ContentPart
		for _, attachment := range attachments {
			attachmentParts = append(attachmentParts, message.BinaryContent{Path: attachment.FilePath, MIMEType: attachment.MimeType, Data: attachment.Content})
		}
		result := a.processGeneration(genCtx, sessionID, content, attachmentParts)
		if result.Error != nil && !errors.Is(result.Error, ErrRequestCancelled) && !errors.Is(result.Error, context.Canceled) {
			logging.ErrorPersist(result.Error.Error())
		}
		logging.Debug("Request completed", "sessionID", sessionID)
		a.activeRequests.Delete(sessionID)
		cancel()
		a.Publish(pubsub.CreatedEvent, result)
		events <- result
		close(events)
	}()
	return events, nil
}

func (a *agent) processGeneration(ctx context.Context, sessionID, content string, attachmentParts []message.ContentPart) AgentEvent {
	debugEnabled := a.config.Debug
	// List existing messages; if none, start title generation asynchronously.
	msgs, err := a.messages.List(ctx, sessionID)
	if err != nil {
		return a.err(fmt.Errorf("failed to list messages: %w", err))
	}
	if len(msgs) == 0 {
		go func() {
			defer logging.RecoverPanic("agent.Run", func() {
				logging.ErrorPersist("panic while generating title")
			})
			titleErr := a.generateTitle(context.Background(), sessionID, content)
			if titleErr != nil {
				logging.ErrorPersist(fmt.Sprintf("failed to generate title: %v", titleErr))
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
			if errors.Is(err, context.Canceled) {
				agentMessage.AddFinish(message.FinishReasonCanceled)
				a.messages.Update(context.Background(), agentMessage)
				return a.err(ErrRequestCancelled)
			}
			return a.err(fmt.Errorf("failed to process events: %w", err))
		}
		if debugEnabled {
			seqId := (len(msgHistory) + 1) / 2
			toolResultFilepath := logging.WriteToolResultsJson(sessionID, seqId, toolResults)
			logging.Info("Result", "message", agentMessage.FinishReason(), "toolResults", "{}", "filepath", toolResultFilepath)
		} else {
			logging.Info("Result", "message", agentMessage.FinishReason(), "toolResults", toolResults)
		}
		if (agentMessage.FinishReason() == message.FinishReasonToolUse) && toolResults != nil {
			// We are not done, we need to respond with the tool response
			msgHistory = append(msgHistory, agentMessage, *toolResults)
			continue
		}
		return AgentEvent{
			Type:    AgentEventTypeResponse,
			Message: agentMessage,
			Done:    true,
		}
	}
}

func (a *agent) createUserMessage(ctx context.Context, sessionID, content string, attachmentParts []message.ContentPart) (message.Message, error) {
	parts := []message.ContentPart{message.TextContent{Text: content}}
	parts = append(parts, attachmentParts...)
	return a.messages.Create(ctx, sessionID, message.CreateMessageParams{
		Role:  message.User,
		Parts: parts,
	})
}

func (a *agent) streamAndHandleEvents(ctx context.Context, sessionID string, msgHistory []message.Message) (message.Message, *message.Message, error) {
	ctx = context.WithValue(ctx, toolcore.SessionIDContextKey, sessionID)
	eventChan := a.streamResponse(ctx, a.mainAgent, msgHistory)

	assistantMsg, err := a.messages.Create(ctx, sessionID, message.CreateMessageParams{
		Role:  message.Assistant,
		Parts: []message.ContentPart{},
		Model: a.Model().ID,
	})
	if err != nil {
		return assistantMsg, nil, fmt.Errorf("failed to create assistant message: %w", err)
	}

	// Add the session and message ID into the context if needed by tools.
	ctx = context.WithValue(ctx, toolcore.MessageIDContextKey, assistantMsg.ID)

	// Process each event in the stream.
	for event := range eventChan {
		if processErr := a.processEvent(ctx, sessionID, &assistantMsg, event, a.mainAgent); processErr != nil {
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
	for i, toolCall := range toolCalls {
		select {
		case <-ctx.Done():
			a.finishMessage(context.Background(), &assistantMsg, message.FinishReasonCanceled)
			// Make all future tool calls cancelled
			for j := i; j < len(toolCalls); j++ {
				toolResults[j] = message.ToolResult{
					ToolCallID: toolCalls[j].ID,
					Content:    "Tool execution canceled by user",
					IsError:    true,
				}
			}
			goto out
		default:
			// Continue processing
			var tool toolcore.BaseTool
			for _, availableTool := range a.tools {
				if availableTool.Info().Name == toolCall.Name {
					tool = availableTool
					break
				}
				// Monkey patch for Copilot Sonnet-4 tool repetition obfuscation
				// if strings.HasPrefix(toolCall.Name, availableTool.Info().Name) &&
				// 	strings.HasPrefix(toolCall.Name, availableTool.Info().Name+availableTool.Info().Name) {
				// 	tool = availableTool
				// 	break
				// }
			}

			// Tool not found
			if tool == nil {
				toolResults[i] = message.ToolResult{
					ToolCallID: toolCall.ID,
					Content:    fmt.Sprintf("Tool not found: %s", toolCall.Name),
					IsError:    true,
				}
				continue
			}
			toolResult, toolErr := tool.Run(ctx, toolcore.ToolCall{
				ID:    toolCall.ID,
				Name:  toolCall.Name,
				Input: toolCall.Input,
			})
			if toolErr != nil {
				if errors.Is(toolErr, permission.ErrorPermissionDenied) {
					toolResults[i] = message.ToolResult{
						ToolCallID: toolCall.ID,
						Content:    "Permission denied",
						IsError:    true,
					}
					for j := i + 1; j < len(toolCalls); j++ {
						toolResults[j] = message.ToolResult{
							ToolCallID: toolCalls[j].ID,
							Content:    "Tool execution canceled by user",
							IsError:    true,
						}
					}
					a.finishMessage(ctx, &assistantMsg, message.FinishReasonPermissionDenied)
					break
				}
			}
			toolResults[i] = message.ToolResult{
				ToolCallID: toolCall.ID,
				Content:    toolResult.Content,
				Metadata:   toolResult.Metadata,
				IsError:    toolResult.IsError,
			}
		}
	}
out:
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

func (a *agent) processEvent(ctx context.Context, sessionID string, assistantMsg *message.Message, event client.Event, runtime llmAgentRuntime) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Continue processing.
	}

	switch event.Type {
	case client.EventThinkingDelta:
		assistantMsg.AppendReasoningContent(event.Content)
		return a.messages.Update(ctx, *assistantMsg)
	case client.EventContentDelta:
		assistantMsg.AppendContent(event.Content)
		return a.messages.Update(ctx, *assistantMsg)
	case client.EventToolUseStart:
		assistantMsg.AddToolCall(*event.ToolCall)
		return a.messages.Update(ctx, *assistantMsg)
	// TODO: see how to handle this
	// case client.EventToolUseDelta:
	// 	tm := time.Unix(assistantMsg.UpdatedAt, 0)
	// 	assistantMsg.AppendToolCallInput(event.ToolCall.ID, event.ToolCall.Input)
	// 	if time.Since(tm) > 1000*time.Millisecond {
	// 		err := a.messages.Update(ctx, *assistantMsg)
	// 		assistantMsg.UpdatedAt = time.Now().Unix()
	// 		return err
	// 	}
	case client.EventToolUseStop:
		assistantMsg.FinishToolCall(event.ToolCall.ID)
		return a.messages.Update(ctx, *assistantMsg)
	case client.EventError:
		if errors.Is(event.Error, context.Canceled) {
			logging.InfoPersist(fmt.Sprintf("Event processing canceled for session: %s", sessionID))
			return context.Canceled
		}
		logging.ErrorPersist(event.Error.Error())
		return event.Error
	case client.EventComplete:
		assistantMsg.SetToolCalls(event.Response.ToolCalls)
		assistantMsg.AddFinish(event.Response.FinishReason)
		if err := a.messages.Update(ctx, *assistantMsg); err != nil {
			return fmt.Errorf("failed to update message: %w", err)
		}
		return a.TrackUsage(ctx, sessionID, a.modelFor(runtime), event.Response.Usage)
	}

	return nil
}

func (a *agent) TrackUsage(ctx context.Context, sessionID string, model models.Model, usage client.TokenUsage) error {
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

func (a *agent) Update(profileKey string, modelID models.ModelID) (models.Model, error) {
	if a.IsBusy() {
		return models.Model{}, fmt.Errorf("cannot change model while processing requests")
	}
	if profileKey != "" && profileKey != "coder" {
		return models.Model{}, fmt.Errorf("model profile %s not supported", profileKey)
	}
	a.config.Provider.AgentProvider.ModelID = modelID
	a.mainAgent.modelID = modelID

	return a.Model(), nil
}

func (a *agent) Summarize(ctx context.Context, sessionID string) error {
	if a.summarizeAgent == nil {
		return fmt.Errorf("summarize agent not available")
	}

	// Check if session is busy
	if a.IsSessionBusy(sessionID) {
		return ErrSessionBusy
	}

	// Create a new context with cancellation
	summarizeCtx, cancel := context.WithCancel(ctx)

	// Store the cancel function in activeRequests to allow cancellation
	a.activeRequests.Store(sessionID+"-summarize", cancel)

	go func() {
		defer a.activeRequests.Delete(sessionID + "-summarize")
		defer cancel()
		event := AgentEvent{
			Type:     AgentEventTypeSummarize,
			Progress: "Starting summarization...",
		}

		a.Publish(pubsub.CreatedEvent, event)
		// Get all messages from the session
		msgs, err := a.messages.List(summarizeCtx, sessionID)
		if err != nil {
			event = AgentEvent{
				Type:  AgentEventTypeError,
				Error: fmt.Errorf("failed to list messages: %w", err),
				Done:  true,
			}
			a.Publish(pubsub.CreatedEvent, event)
			return
		}
		summarizeCtx = context.WithValue(summarizeCtx, toolcore.SessionIDContextKey, sessionID)

		if len(msgs) == 0 {
			event = AgentEvent{
				Type:  AgentEventTypeError,
				Error: fmt.Errorf("no messages to summarize"),
				Done:  true,
			}
			a.Publish(pubsub.CreatedEvent, event)
			return
		}

		event = AgentEvent{
			Type:     AgentEventTypeSummarize,
			Progress: "Analyzing conversation...",
		}
		a.Publish(pubsub.CreatedEvent, event)

		summarizePrompt := a.summarizeAgent.systemMessage
		if strings.TrimSpace(summarizePrompt) == "" {
			event = AgentEvent{
				Type:  AgentEventTypeError,
				Error: fmt.Errorf("summarize prompt is empty"),
				Done:  true,
			}
			a.Publish(pubsub.CreatedEvent, event)
			return
		}

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

		a.Publish(pubsub.CreatedEvent, event)

		// Send the messages to the summarize provider
		response, err := a.sendMessages(summarizeCtx, *a.summarizeAgent, msgsWithPrompt)
		if err != nil {
			event = AgentEvent{
				Type:  AgentEventTypeError,
				Error: fmt.Errorf("failed to summarize: %w", err),
				Done:  true,
			}
			a.Publish(pubsub.CreatedEvent, event)
			return
		}

		summary := strings.TrimSpace(response.Content)
		if summary == "" {
			event = AgentEvent{
				Type:  AgentEventTypeError,
				Error: fmt.Errorf("empty summary returned"),
				Done:  true,
			}
			a.Publish(pubsub.CreatedEvent, event)
			return
		}
		event = AgentEvent{
			Type:     AgentEventTypeSummarize,
			Progress: "Creating new session...",
		}

		a.Publish(pubsub.CreatedEvent, event)
		oldSession, err := a.sessions.Get(summarizeCtx, sessionID)
		if err != nil {
			event = AgentEvent{
				Type:  AgentEventTypeError,
				Error: fmt.Errorf("failed to get session: %w", err),
				Done:  true,
			}

			a.Publish(pubsub.CreatedEvent, event)
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
			Model: a.modelFor(*a.summarizeAgent).ID,
		})
		if err != nil {
			event = AgentEvent{
				Type:  AgentEventTypeError,
				Error: fmt.Errorf("failed to create summary message: %w", err),
				Done:  true,
			}

			a.Publish(pubsub.CreatedEvent, event)
			return
		}
		oldSession.SummaryMessageID = msg.ID
		oldSession.CompletionTokens = response.Usage.OutputTokens
		oldSession.PromptTokens = 0
		model := a.modelFor(*a.summarizeAgent)
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
			a.Publish(pubsub.CreatedEvent, event)
		}

		event = AgentEvent{
			Type:      AgentEventTypeSummarize,
			SessionID: oldSession.ID,
			Progress:  "Summary complete",
			Done:      true,
		}
		a.Publish(pubsub.CreatedEvent, event)
		// Send final success event with the new session ID
	}()

	return nil
}
