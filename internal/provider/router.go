package provider

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"ferryman-agent/internal/data/llm/models"
)

var (
	ErrProviderNotConfigured  = errors.New("provider is not configured")
	ErrModelNotConfigured     = errors.New("model is not configured")
	ErrProviderTargetNotFound = errors.New("provider target not found")
	ErrProviderTargetExists   = errors.New("provider target already exists")
)

type RouteRequest struct {
	Provider models.ModelProvider
	ModelID  models.ModelID
}

type Router interface {
	Route(ctx context.Context, req RouteRequest) (ProviderClient, error)
}

type DefaultRouter struct {
	mu      sync.RWMutex
	targets map[models.ModelProvider]map[models.ModelID]ProviderClient
}

func NewDefaultRouter(targets map[models.ModelProvider]map[models.ModelID]ProviderClient) *DefaultRouter {
	if targets == nil {
		targets = map[models.ModelProvider]map[models.ModelID]ProviderClient{}
	}
	return &DefaultRouter{targets: targets}
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
