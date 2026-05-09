package provider

import (
	"context"
	"errors"
	"fmt"
	"sync"

	clientfactory "ferryman-agent/internal/data/llm/client/factory"
	"ferryman-agent/internal/data/llm/models"
)

var (
	ErrProviderNotConfigured  = errors.New("provider is not configured")
	ErrModelNotConfigured     = errors.New("model is not configured")
	ErrProviderTargetNotFound = errors.New("provider target not found")
	ErrProviderTargetExists   = errors.New("provider target already exists")
)

type ProviderConfig struct {
	Provider models.ModelProvider `json:"provider"`
	APIKey   string               `json:"apiKey,omitempty"`
	BaseURL  string               `json:"baseURL,omitempty"`
	Models   []ModelConfig        `json:"models"`
	Disabled bool                 `json:"disabled,omitempty"`
}

type ModelConfig struct {
	ModelID         models.ModelID `json:"model_id"`
	APIModel        string         `json:"api_model,omitempty"`
	MaxTokens       int64          `json:"maxTokens,omitempty"`
	ReasoningEffort string         `json:"reasoning_effort,omitempty"`
}

func (p ProviderConfig) Configured() bool {
	return p.Provider != "" || p.APIKey != "" || p.BaseURL != "" || len(p.Models) > 0
}

func (p ProviderConfig) PrimaryModelConfig() (ModelConfig, bool) {
	if len(p.Models) == 0 {
		return ModelConfig{}, false
	}
	return p.Models[0], true
}

func ApplyModelConfig(model models.Model, modelCfg ModelConfig) models.Model {
	if modelCfg.APIModel != "" {
		model.APIModel = modelCfg.APIModel
	}
	if modelCfg.MaxTokens > 0 {
		model.MaxTokens = modelCfg.MaxTokens
	}
	if modelCfg.ReasoningEffort != "" {
		model.ReasoningEffort = modelCfg.ReasoningEffort
	}
	return model
}

type RouteRequest struct {
	Provider models.ModelProvider
	ModelID  models.ModelID
}

type Router interface {
	Route(ctx context.Context, req RouteRequest) (ProviderClient, error)
}

type ProviderTargets map[models.ModelProvider]map[models.ModelID]ProviderClient

type DefaultRouter struct {
	mu      sync.RWMutex
	targets ProviderTargets
}

func NewDefaultRouter(configs ...ProviderConfig) (*DefaultRouter, error) {
	r := &DefaultRouter{targets: ProviderTargets{}}
	for _, cfg := range configs {
		if cfg.Disabled {
			continue
		}
		if cfg.Provider == "" {
			return nil, ErrProviderNotConfigured
		}
		if len(cfg.Models) == 0 {
			return nil, fmt.Errorf("%w: %s", ErrModelNotConfigured, cfg.Provider)
		}

		vendorClient, err := clientfactory.NewClient(
			cfg.Provider,
			clientfactory.WithAPIKey(cfg.APIKey),
			clientfactory.WithBaseURL(cfg.BaseURL),
		)
		if err != nil {
			return nil, err
		}

		for _, modelCfg := range cfg.Models {
			if modelCfg.ModelID == "" {
				return nil, fmt.Errorf("%w: empty model_id for %s", ErrModelNotConfigured, cfg.Provider)
			}

			model := ApplyModelConfig(models.ResolveModel(cfg.Provider, modelCfg.ModelID), modelCfg)
			if r.targets[cfg.Provider] == nil {
				r.targets[cfg.Provider] = map[models.ModelID]ProviderClient{}
			}
			if _, exists := r.targets[cfg.Provider][model.ID]; exists {
				return nil, fmt.Errorf("%w: %s/%s", ErrProviderTargetExists, cfg.Provider, model.ID)
			}

			r.targets[cfg.Provider][model.ID] = ProviderClient{
				Provider: cfg.Provider,
				Model:    model,
				Client:   vendorClient,
			}
		}
	}
	return r, nil
}

func NewDefaultRouterFromConfigs(configs ...ProviderConfig) (*DefaultRouter, error) {
	return NewDefaultRouter(configs...)
}

func (r *DefaultRouter) Route(ctx context.Context, req RouteRequest) (ProviderClient, error) {
	_ = ctx

	r.mu.RLock()
	defer r.mu.RUnlock()

	modelsByProvider, ok := r.targets[req.Provider]
	if !ok {
		return ProviderClient{}, fmt.Errorf("%w: %s/%s", ErrProviderTargetNotFound, req.Provider, req.ModelID)
	}
	target, ok := modelsByProvider[req.ModelID]
	if !ok {
		return ProviderClient{}, fmt.Errorf("%w: %s/%s", ErrProviderTargetNotFound, req.Provider, req.ModelID)
	}
	return target, nil
}
