package config

import (
	"encoding/json"
	"strings"
	"testing"

	"ferryman-agent/internal/data/llm/models"
)

func TestValidateProviderConfigRequiresModels(t *testing.T) {
	cfg := Config{
		Provider: ProviderConfig{
			Provider: models.ProviderOpenAI,
		},
	}

	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "provider models is required") {
		t.Fatalf("expected models required error, got %v", err)
	}
}

func TestValidateProviderConfigRejectsLegacyModelConfig(t *testing.T) {
	var cfg Config
	err := json.Unmarshal([]byte(`{
		"provider": {
			"provider": "openai",
			"modelConfig": {"model_id": "o4-mini"}
		}
	}`), &cfg)
	if err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}

	err = Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "modelConfig is not supported; use models") {
		t.Fatalf("expected legacy modelConfig error, got %v", err)
	}
}

func TestValidateProviderConfigAcceptsModels(t *testing.T) {
	cfg := Config{
		Provider: ProviderConfig{
			Provider: models.ProviderOpenAI,
			Models: []ModelConfig{
				{ModelId: "o4-mini"},
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

	model = ApplyModelConfig(model, ModelConfig{
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
