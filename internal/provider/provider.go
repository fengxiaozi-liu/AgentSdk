package provider

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"sync"

	"ferryman-agent/internal/config"
	llmclient "ferryman-agent/internal/data/llm/client"
	anthropicclient "ferryman-agent/internal/data/llm/client/anthropic"
	azureclient "ferryman-agent/internal/data/llm/client/azure"
	bedrockclient "ferryman-agent/internal/data/llm/client/bedrock"
	copilotclient "ferryman-agent/internal/data/llm/client/copilot"
	geminiclient "ferryman-agent/internal/data/llm/client/gemini"
	mockclient "ferryman-agent/internal/data/llm/client/mock"
	openaiclient "ferryman-agent/internal/data/llm/client/openai"
	vertexaiclient "ferryman-agent/internal/data/llm/client/vertexai"
	"ferryman-agent/internal/data/llm/models"
	"ferryman-agent/internal/memory/message"

	"github.com/google/wire"
)

var (
	ErrProviderNotConfigured    = errors.New("provider is not configured")
	ErrModelNotConfigured       = errors.New("model is not configured")
	ErrProviderTargetNotFound   = errors.New("provider target not found")
	ErrNoFallbackModelAvailable = errors.New("no fallback model available")
)

var ProviderSet = wire.NewSet(ProvideService, wire.Bind(new(Service), new(*ProviderService)))

type ProviderClient struct {
	Provider models.ModelProvider
	Model    models.Model
	Client   llmclient.Client
	Weight   int
	Priority int
	Disabled bool
}

type Service interface {
	SendMessages(ctx context.Context, request llmclient.Request) (*llmclient.Response, error)
	StreamResponse(ctx context.Context, request llmclient.Request) <-chan llmclient.Event
	ActiveTarget(modelID models.ModelID) (ProviderClient, bool)
	AvailableTargets(modelID models.ModelID) []ProviderClient
}

func ProvideService(cfg *config.Config) (*ProviderService, error) {
	if cfg == nil {
		return NewService()
	}
	return NewService(config.ProviderConfigs(*cfg)...)
}

type ProviderService struct {
	mu            sync.RWMutex
	clients       map[models.ModelID][]ProviderClient
	activeTargets map[models.ModelID]ProviderClient
}

func NewService(providerConfigs ...config.ProviderConfig) (*ProviderService, error) {
	service := &ProviderService{
		clients:       map[models.ModelID][]ProviderClient{},
		activeTargets: map[models.ModelID]ProviderClient{},
	}
	for _, providerCfg := range providerConfigs {
		if !providerCfg.Configured() || providerCfg.Disabled {
			continue
		}
		if err := service.addProvider(providerCfg); err != nil {
			return nil, err
		}
	}
	return service, nil
}

func (s *ProviderService) addProvider(providerCfg config.ProviderConfig) error {
	if providerCfg.Provider == "" {
		return ErrProviderNotConfigured
	}
	if len(providerCfg.Models) == 0 {
		return fmt.Errorf("%w: %s", ErrModelNotConfigured, providerCfg.Provider)
	}

	vendorClient, err := newVendorClient(providerCfg)
	if err != nil {
		return err
	}

	for _, modelCfg := range providerCfg.Models {
		if modelCfg.ModelId == "" {
			return fmt.Errorf("%w: empty model_id for %s", ErrModelNotConfigured, providerCfg.Provider)
		}
		model := config.ApplyModelConfig(models.ResolveModel(providerCfg.Provider, modelCfg.ModelId), modelCfg)
		target := ProviderClient{
			Provider: providerCfg.Provider,
			Model:    model,
			Client:   vendorClient,
			Weight:   modelCfg.Weight,
			Priority: modelCfg.Priority,
		}
		s.clients[model.ID] = append(s.clients[model.ID], target)
	}
	for modelID := range s.clients {
		sortTargets(s.clients[modelID])
	}
	return nil
}

func newVendorClient(providerCfg config.ProviderConfig) (llmclient.Client, error) {
	switch providerCfg.Provider {
	case models.ProviderCopilot:
		return copilotclient.NewClient(providerCfg.APIKey), nil
	case models.ProviderAnthropic:
		return anthropicclient.NewClient(providerCfg.APIKey, anthropicclient.WithShouldThinkFn(anthropicclient.DefaultShouldThinkFn)), nil
	case models.ProviderOpenAI:
		return openaiclient.NewClient(providerCfg.APIKey, openAIBaseURLOption(providerCfg.BaseURL)...), nil
	case models.ProviderGemini:
		return geminiclient.NewClient(providerCfg.APIKey), nil
	case models.ProviderBedrock:
		return bedrockclient.NewClient(providerCfg.APIKey), nil
	case models.ProviderGROQ:
		return openaiclient.NewClient(providerCfg.APIKey, openaiclient.WithDefaultBaseURL("https://api.groq.com/openai/v1")), nil
	case models.ProviderAzure:
		return azureclient.NewClient(providerCfg.APIKey), nil
	case models.ProviderVertexAI:
		return vertexaiclient.NewClient(providerCfg.APIKey), nil
	case models.ProviderOpenRouter:
		return openaiclient.NewClient(
			providerCfg.APIKey,
			openaiclient.WithDefaultBaseURL("https://openrouter.ai/api/v1"),
			openaiclient.WithExtraHeaders(map[string]string{
				"HTTP-Referer": "ferryer.ai",
				"X-Title":      "Ferryer",
			}),
		), nil
	case models.ProviderXAI:
		return openaiclient.NewClient(providerCfg.APIKey, openaiclient.WithDefaultBaseURL("https://api.x.ai/v1")), nil
	case models.ProviderLocal:
		baseURL := providerCfg.BaseURL
		if baseURL == "" {
			baseURL = os.Getenv("LOCAL_ENDPOINT")
		}
		return openaiclient.NewClient(providerCfg.APIKey, openaiclient.WithDefaultBaseURL(baseURL)), nil
	case models.ProviderMock:
		return mockclient.NewClient(), nil
	default:
		return nil, fmt.Errorf("provider not supported: %s", providerCfg.Provider)
	}
}

func openAIBaseURLOption(baseURL string) []openaiclient.Option {
	if baseURL == "" {
		return nil
	}
	return []openaiclient.Option{openaiclient.WithBaseURL(baseURL)}
}

func sortTargets(targets []ProviderClient) {
	sort.SliceStable(targets, func(i, j int) bool {
		return targets[i].Priority < targets[j].Priority
	})
}

func (s *ProviderService) cleanMessages(messages []message.Message) (cleaned []message.Message) {
	for _, msg := range messages {
		if len(msg.Parts) == 0 {
			continue
		}
		cleaned = append(cleaned, msg)
	}
	return
}

func (s *ProviderService) SendMessages(ctx context.Context, request llmclient.Request) (*llmclient.Response, error) {
	request.Messages = s.cleanMessages(request.Messages)
	requestedModelID := request.ModelID
	targets, err := s.selectTargets(request.ModelID, request.Provider)
	if err != nil {
		return nil, err
	}

	var lastErr error
	for _, target := range targets {
		request.Model = target.Model
		s.setActiveTarget(request.ModelID, target)
		response, err := target.Client.Send(ctx, request)
		if err == nil {
			return response, nil
		}
		lastErr = err
	}

	fallback, err := s.randomFallbackTarget(request.ModelID)
	if err != nil {
		if lastErr != nil {
			return nil, lastErr
		}
		return nil, err
	}
	request.ModelID = fallback.Model.ID
	request.Model = fallback.Model
	s.setActiveTarget(requestedModelID, fallback)
	s.setActiveTarget(request.ModelID, fallback)
	return fallback.Client.Send(ctx, request)
}

func (s *ProviderService) StreamResponse(ctx context.Context, request llmclient.Request) <-chan llmclient.Event {
	request.Messages = s.cleanMessages(request.Messages)
	target, err := s.selectTarget(request.ModelID, request.Provider)
	if err != nil {
		ch := make(chan llmclient.Event, 1)
		ch <- llmclient.Event{Type: llmclient.EventError, Error: err}
		close(ch)
		return ch
	}
	request.Model = target.Model
	s.setActiveTarget(request.ModelID, target)
	return target.Client.Stream(ctx, request)
}

func (s *ProviderService) ActiveTarget(modelID models.ModelID) (ProviderClient, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	target, ok := s.activeTargets[modelID]
	return target, ok
}

func (s *ProviderService) AvailableTargets(modelID models.ModelID) []ProviderClient {
	s.mu.RLock()
	defer s.mu.RUnlock()
	targets := append([]ProviderClient(nil), s.clients[modelID]...)
	return targets
}

func (s *ProviderService) setActiveTarget(modelID models.ModelID, target ProviderClient) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.activeTargets[modelID] = target
}

func (s *ProviderService) selectTarget(modelID models.ModelID, provider models.ModelProvider) (ProviderClient, error) {
	targets, err := s.selectTargets(modelID, provider)
	if err != nil {
		return ProviderClient{}, err
	}
	return targets[0], nil
}

func (s *ProviderService) selectTargets(modelID models.ModelID, provider models.ModelProvider) ([]ProviderClient, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	candidates := s.clients[modelID]
	if len(candidates) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrModelNotConfigured, modelID)
	}

	targets := make([]ProviderClient, 0, len(candidates))
	for _, target := range candidates {
		if target.Disabled {
			continue
		}
		if provider != "" && target.Provider != provider {
			continue
		}
		targets = append(targets, target)
	}
	if len(targets) == 0 {
		if provider != "" {
			return nil, fmt.Errorf("%w: %s/%s", ErrProviderTargetNotFound, provider, modelID)
		}
		return nil, fmt.Errorf("%w: %s", ErrProviderTargetNotFound, modelID)
	}
	return weightedTargets(targets), nil
}

func weightedTargets(targets []ProviderClient) []ProviderClient {
	sortTargets(targets)
	if len(targets) <= 1 {
		return targets
	}
	firstPriority := targets[0].Priority
	samePriority := []ProviderClient{}
	remaining := []ProviderClient{}
	for _, target := range targets {
		if target.Priority == firstPriority {
			samePriority = append(samePriority, target)
		} else {
			remaining = append(remaining, target)
		}
	}
	if len(samePriority) <= 1 {
		return targets
	}
	start := weightedIndex(samePriority)
	ordered := append([]ProviderClient{}, samePriority[start:]...)
	ordered = append(ordered, samePriority[:start]...)
	ordered = append(ordered, remaining...)
	return ordered
}

func weightedIndex(targets []ProviderClient) int {
	total := 0
	for _, target := range targets {
		weight := target.Weight
		if weight <= 0 {
			weight = 1
		}
		total += weight
	}
	pick := rand.Intn(total)
	for i, target := range targets {
		weight := target.Weight
		if weight <= 0 {
			weight = 1
		}
		if pick < weight {
			return i
		}
		pick -= weight
	}
	return 0
}

func (s *ProviderService) randomFallbackTarget(modelID models.ModelID) (ProviderClient, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	candidates := []ProviderClient{}
	for candidateModelID, targets := range s.clients {
		if candidateModelID == modelID {
			continue
		}
		for _, target := range targets {
			if !target.Disabled {
				candidates = append(candidates, target)
			}
		}
	}
	if len(candidates) == 0 {
		return ProviderClient{}, ErrNoFallbackModelAvailable
	}
	return candidates[rand.Intn(len(candidates))], nil
}
