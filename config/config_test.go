package config

import (
	"path/filepath"
	"testing"

	"ferryman-agent/llm/models"
)

func resetConfigForTest() {
	cfg = nil
}

func TestUseAndModelProfile(t *testing.T) {
	resetConfigForTest()
	workingDir := t.TempDir()

	cfg, err := Use(Config{
		WorkingDir: workingDir,
		Data:       Data{Directory: filepath.Join(workingDir, "data")},
		Providers:  map[models.ModelProvider]Provider{"openai": {APIKey: "test-key"}},
		ModelProfiles: map[string]ModelConfig{
			"coder": {
				Provider:        "openai",
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

	runtimeCfg, ok := ModelProfile("coder")
	if !ok {
		t.Fatal("expected coder model profile")
	}
	if runtimeCfg.Model != models.O4Mini {
		t.Fatalf("expected model %q, got %q", models.O4Mini, runtimeCfg.Model)
	}
	if runtimeCfg.MaxTokens != 2048 {
		t.Fatalf("expected max tokens 2048, got %d", runtimeCfg.MaxTokens)
	}
	if runtimeCfg.ReasoningEffort != "high" {
		t.Fatalf("expected reasoning effort high, got %q", runtimeCfg.ReasoningEffort)
	}
	if WorkingDirectory() != workingDir {
		t.Fatalf("expected working directory %q, got %q", workingDir, WorkingDirectory())
	}
}

func TestUseAllowsConfiguredProviderWithArbitraryModel(t *testing.T) {
	resetConfigForTest()
	workingDir := t.TempDir()

	cfg, err := Use(Config{
		WorkingDir: workingDir,
		Providers:  map[models.ModelProvider]Provider{"openai": {APIKey: "test-key"}},
		ModelProfiles: map[string]ModelConfig{
			"coder": {
				Provider:  "openai",
				Model:     "future-model",
				MaxTokens: 1024,
			},
		},
	})
	if err != nil {
		t.Fatalf("Use: %v", err)
	}
	agent := cfg.ModelProfiles["coder"]
	if agent.Model != "future-model" || agent.Provider != models.ProviderOpenAI {
		t.Fatalf("expected arbitrary openai model to remain configured: %+v", agent)
	}
}
