package agent

import (
	"context"
	"errors"
	"testing"

	"ferryman-agent/internal/data/llm/models"
	"ferryman-agent/internal/provider"
)

func TestWithProviderConfigsRejectsInvalidConfigs(t *testing.T) {
	cfg := AgentConfig{}
	err := WithProviderConfigs(ModelTarget{}, ProviderConfig{})(&cfg)
	if !errors.Is(err, provider.ErrProviderNotConfigured) {
		t.Fatalf("expected ErrProviderNotConfigured, got %v", err)
	}

	cfg = AgentConfig{}
	err = WithProviderConfigs(ModelTarget{}, ProviderConfig{Provider: models.ProviderMock})(&cfg)
	if !errors.Is(err, provider.ErrModelNotConfigured) {
		t.Fatalf("expected ErrModelNotConfigured, got %v", err)
	}

	cfg = AgentConfig{}
	err = WithProviderConfigs(ModelTarget{}, ProviderConfig{
		Provider: models.ProviderMock,
		Models:   []ModelConfig{{}},
	})(&cfg)
	if !errors.Is(err, provider.ErrModelNotConfigured) {
		t.Fatalf("expected ErrModelNotConfigured for empty model id, got %v", err)
	}
}

func TestWithProviderConfigsSkipsDisabledConfig(t *testing.T) {
	cfg := AgentConfig{}
	err := WithProviderConfigs(ModelTarget{}, ProviderConfig{
		Provider: models.ProviderMock,
		Disabled: true,
	})(&cfg)
	if err != nil {
		t.Fatalf("provider configs: %v", err)
	}
	_, err = cfg.Provider.Router.Route(context.Background(), provider.RouteRequest{Provider: models.ProviderMock, ModelID: "model-a"})
	if !errors.Is(err, provider.ErrProviderTargetNotFound) {
		t.Fatalf("expected ErrProviderTargetNotFound, got %v", err)
	}
}

func TestWithProviderConfigsRejectsDuplicateTarget(t *testing.T) {
	cfg := AgentConfig{}
	err := WithProviderConfigs(ModelTarget{}, ProviderConfig{
		Provider: models.ProviderMock,
		Models: []ModelConfig{
			{ModelID: "model-a"},
			{ModelID: "model-a"},
		},
	})(&cfg)
	if !errors.Is(err, provider.ErrProviderTargetExists) {
		t.Fatalf("expected ErrProviderTargetExists, got %v", err)
	}
}
