package config

import (
	"strings"
	"testing"

	"ferryman-agent/internal/data/llm/models"
	"ferryman-agent/internal/provider"
)

func TestValidateProviderConfigRequiresModels(t *testing.T) {
	cfg := Config{
		Providers: []provider.ProviderConfig{
			{Provider: models.ProviderOpenAI},
		},
	}

	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "providers[0] models is required") {
		t.Fatalf("expected models required error, got %v", err)
	}
}

func TestValidateProviderConfigAcceptsModels(t *testing.T) {
	cfg := Config{
		Providers: []provider.ProviderConfig{
			{
				Provider: models.ProviderOpenAI,
				Models: []provider.ModelConfig{
					{ModelID: "o4-mini"},
				},
			},
		},
	}

	if err := Validate(cfg); err != nil {
		t.Fatalf("expected valid config, got %v", err)
	}
}

func TestApplyModelConfigOverridesModel(t *testing.T) {
	model := models.Model{
		ID:              "logical",
		APIModel:        "catalog-model",
		MaxTokens:       4096,
		ReasoningEffort: "low",
	}

	model = provider.ApplyModelConfig(model, provider.ModelConfig{
		APIModel:        "api-model",
		MaxTokens:       8192,
		ReasoningEffort: "high",
	})

	if model.APIModel != "api-model" {
		t.Fatalf("expected APIModel override, got %q", model.APIModel)
	}
	if model.MaxTokens != 8192 {
		t.Fatalf("expected MaxTokens override, got %d", model.MaxTokens)
	}
	if model.ReasoningEffort != "high" {
		t.Fatalf("expected ReasoningEffort override, got %q", model.ReasoningEffort)
	}
}
