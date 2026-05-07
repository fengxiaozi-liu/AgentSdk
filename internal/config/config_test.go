package config

import (
	"testing"

	datadb "ferryman-agent/internal/data/db"
	"ferryman-agent/internal/data/llm/models"
	"ferryman-agent/internal/data/llm/provider"
)

func resetConfigForTest() {
	cfg = nil
}

func TestUseAppliesDefaultsAndWorkingDirectory(t *testing.T) {
	resetConfigForTest()
	workingDir := t.TempDir()

	cfg, err := Use(Config{
		WorkingDir: workingDir,
		Database: datadb.DatabaseConfig{
			Type: datadb.DatabaseSQLite,
			Path: ":memory:",
		},
		Provider: provider.ProviderConfig{
			Provider: models.ProviderOpenAI,
			APIKey:   "test-key",
			ModelConfig: provider.ModelConfig{
				Model:           "o4-mini",
				MaxTokens:       2048,
				ReasoningEffort: "high",
			},
		},
	})
	if err != nil {
		t.Fatalf("Use: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected config to be loaded")
	}
	if WorkingDirectory() != workingDir {
		t.Fatalf("expected working directory %q, got %q", workingDir, WorkingDirectory())
	}
	if cfg.Provider.ModelConfig.Model != models.ModelID("o4-mini") {
		t.Fatalf("expected model %q, got %q", "o4-mini", cfg.Provider.ModelConfig.Model)
	}
	if cfg.Provider.ModelConfig.MaxTokens != 2048 {
		t.Fatalf("expected max tokens 2048, got %d", cfg.Provider.ModelConfig.MaxTokens)
	}
	if cfg.Provider.ModelConfig.ReasoningEffort != "high" {
		t.Fatalf("expected reasoning effort high, got %q", cfg.Provider.ModelConfig.ReasoningEffort)
	}
}

func TestUseAllowsConfiguredProviderWithArbitraryModel(t *testing.T) {
	resetConfigForTest()

	cfg, err := Use(Config{
		WorkingDir: t.TempDir(),
		Provider: provider.ProviderConfig{
			Provider: models.ProviderOpenAI,
			APIKey:   "test-key",
			ModelConfig: provider.ModelConfig{
				Model:     "future-model",
				MaxTokens: 1024,
			},
		},
	})
	if err != nil {
		t.Fatalf("Use: %v", err)
	}
	if cfg.Provider.ModelConfig.Model != "future-model" || cfg.Provider.Provider != models.ProviderOpenAI {
		t.Fatalf("expected arbitrary openai model to remain configured: %+v", cfg.Provider)
	}
}
