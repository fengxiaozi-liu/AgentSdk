package provider

import (
	"context"
	"errors"
	"fmt"
	"sync"

	llmclient "ferryman-agent/internal/data/llm/client"
	clientfactory "ferryman-agent/internal/data/llm/client/factory"
	"ferryman-agent/internal/data/llm/models"
	"ferryman-agent/internal/memory/message"

	"github.com/google/wire"
)

var (
	ErrProviderNotConfigured  = errors.New("provider is not configured")
	ErrModelNotConfigured     = errors.New("model is not configured")
	ErrProviderTargetNotFound = errors.New("provider target not found")
	ErrProviderTargetExists   = errors.New("provider target already exists")
)

var ProviderSet = wire.NewSet(ProvideService, wire.Bind(new(Service), new(*ProviderService)))

type ProviderRegistry map[models.ModelID]map[models.ModelProvider]ProviderClient

type ProviderClient struct {
	Provider models.ModelProvider
	Model    models.Model
	Client   llmclient.Client
	Weight   int
	Priority int
	Disabled bool
}

type Service interface {
	Register(register ProviderRegister) error
	SendMessages(ctx context.Context, request llmclient.Request) (*llmclient.Response, error)
	StreamResponse(ctx context.Context, request llmclient.Request) <-chan llmclient.Event
}

func ProvideService(registers []ProviderRegister) (*ProviderService, error) {
	return NewService(registers...)
}

type ProviderService struct {
	mu      sync.RWMutex
	clients ProviderRegistry
}

func NewService(registers ...ProviderRegister) (*ProviderService, error) {
	service := &ProviderService{
		clients: ProviderRegistry{},
	}
	for _, register := range registers {
		if !register.Configured() || register.Disabled {
			continue
		}
		if err := service.Register(register); err != nil {
			return nil, err
		}
	}
	return service, nil
}

func (s *ProviderService) Register(register ProviderRegister) error {
	if register.Provider == "" {
		return ErrProviderNotConfigured
	}
	if len(register.Models) == 0 {
		return fmt.Errorf("%w: %s", ErrModelNotConfigured, register.Provider)
	}

	vendorClient, err := clientfactory.NewClient(
		register.Provider,
		clientfactory.WithAPIKey(register.APIKey),
		clientfactory.WithBaseURL(register.BaseURL),
	)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, modelCfg := range register.Models {
		if modelCfg.ModelId == "" {
			return fmt.Errorf("%w: empty model_id for %s", ErrModelNotConfigured, register.Provider)
		}
		if s.clients[modelCfg.ModelId] == nil {
			s.clients[modelCfg.ModelId] = map[models.ModelProvider]ProviderClient{}
		}
		if _, exists := s.clients[modelCfg.ModelId][register.Provider]; exists {
			return fmt.Errorf("%w: %s/%s", ErrProviderTargetExists, register.Provider, modelCfg.ModelId)
		}

		model := ApplyModelConfig(models.ResolveModel(register.Provider, modelCfg.ModelId), modelCfg)
		s.clients[model.ID][register.Provider] = ProviderClient{
			Provider: register.Provider,
			Model:    model,
			Client:   vendorClient,
			Weight:   modelCfg.Weight,
			Priority: modelCfg.Priority,
		}
	}
	return nil
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

	s.mu.RLock()
	providers, ok := s.clients[request.ModelID]
	if !ok {
		s.mu.RUnlock()
		return nil, fmt.Errorf("%w: %s", ErrModelNotConfigured, request.ModelID)
	}
	target, ok := providers[request.Provider]
	if !ok || target.Disabled {
		s.mu.RUnlock()
		return nil, fmt.Errorf("%w: %s/%s", ErrProviderTargetNotFound, request.Provider, request.ModelID)
	}
	s.mu.RUnlock()

	request.Model = target.Model
	return target.Client.Send(ctx, request)
}

func (s *ProviderService) StreamResponse(ctx context.Context, request llmclient.Request) <-chan llmclient.Event {
	request.Messages = s.cleanMessages(request.Messages)

	s.mu.RLock()
	providers, ok := s.clients[request.ModelID]
	if !ok {
		s.mu.RUnlock()
		ch := make(chan llmclient.Event, 1)
		ch <- llmclient.Event{Type: llmclient.EventError, Error: fmt.Errorf("%w: %s", ErrModelNotConfigured, request.ModelID)}
		close(ch)
		return ch
	}
	target, ok := providers[request.Provider]
	if !ok || target.Disabled {
		s.mu.RUnlock()
		ch := make(chan llmclient.Event, 1)
		ch <- llmclient.Event{Type: llmclient.EventError, Error: fmt.Errorf("%w: %s/%s", ErrProviderTargetNotFound, request.Provider, request.ModelID)}
		close(ch)
		return ch
	}
	s.mu.RUnlock()

	request.Model = target.Model
	return target.Client.Stream(ctx, request)
}
