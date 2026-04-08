package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/opencode-ai/opencode/agent/llm/models"
)

func TestLoadAndRuntimeFor(t *testing.T) {
	workingDir := t.TempDir()
	t.Setenv("HOME", workingDir)
	t.Setenv("USERPROFILE", workingDir)
	t.Setenv("XDG_CONFIG_HOME", workingDir)
	t.Setenv("LOCALAPPDATA", workingDir)

	configJSON := `{
  "data": {"directory": "` + filepath.ToSlash(filepath.Join(workingDir, "data")) + `"},
  "providers": {"openai": {"apiKey": "test-key"}},
  "agents": {
    "coder": {
      "model": "o4-mini",
      "maxTokens": 2048,
      "reasoningEffort": "high"
    }
  }
}`
	if err := os.WriteFile(filepath.Join(workingDir, ".opencode.json"), []byte(configJSON), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(workingDir, false)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected config to be loaded")
	}

	runtimeCfg, ok := RuntimeFor(AgentCoder)
	if !ok {
		t.Fatal("expected coder runtime config")
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
