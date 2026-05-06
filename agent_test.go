package agent

import (
	"os"
	"path/filepath"
	"testing"

	sdkconfig "ferryman-agent/config"
	"ferryman-agent/llm/models"
	"github.com/spf13/viper"
)

func TestCreateAgentProviderAllowsConfiguredArbitraryModel(t *testing.T) {
	sdkconfigTestReset()
	workingDir := t.TempDir()
	t.Setenv("HOME", workingDir)
	t.Setenv("USERPROFILE", workingDir)
	t.Setenv("XDG_CONFIG_HOME", workingDir)
	t.Setenv("LOCALAPPDATA", workingDir)

	configJSON := `{
  "data": {"directory": "` + filepath.ToSlash(filepath.Join(workingDir, "data")) + `"},
  "providers": {"__mock": {"apiKey": "test-key"}},
  "agents": {
    "coder": {
      "model": "future-model",
      "provider": "__mock",
      "maxTokens": 128
    }
  }
}`
	if err := os.WriteFile(filepath.Join(workingDir, ".opencode.json"), []byte(configJSON), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if _, err := sdkconfig.Load(workingDir, false); err != nil {
		t.Fatalf("load config: %v", err)
	}

	provider, err := createAgentProvider(sdkconfig.AgentCoder)
	if err != nil {
		t.Fatalf("createAgentProvider: %v", err)
	}
	model := provider.Model()
	if model.ID != models.ModelID("future-model") || model.Provider != models.ProviderMock {
		t.Fatalf("unexpected provider model: %+v", model)
	}
}

func sdkconfigTestReset() {
	viper.Reset()
}
