package base

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ferryman-agent/config"
	toolcore "ferryman-agent/tools/core"
)

func ensureBaseConfig(t *testing.T) string {
	t.Helper()
	if config.Get() != nil {
		workingDir := config.WorkingDirectory()
		if err := os.MkdirAll(workingDir, 0o755); err != nil {
			t.Fatalf("create configured working directory: %v", err)
		}
		return workingDir
	}

	workingDir := t.TempDir()
	t.Setenv("HOME", workingDir)
	t.Setenv("USERPROFILE", workingDir)
	t.Setenv("XDG_CONFIG_HOME", workingDir)
	t.Setenv("LOCALAPPDATA", workingDir)

	configJSON := `{
  "data": {"directory": "` + filepath.ToSlash(filepath.Join(workingDir, "data")) + `"},
  "providers": {"openai": {"apiKey": "test-key"}},
  "agents": {
    "coder": {"model": "o4-mini", "maxTokens": 2048}
  }
}`
	if err := os.WriteFile(filepath.Join(workingDir, ".opencode.json"), []byte(configJSON), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if _, err := config.Load(workingDir, false); err != nil {
		t.Fatalf("load config: %v", err)
	}
	return workingDir
}

func TestViewToolRunsWithoutExtensions(t *testing.T) {
	workingDir := ensureBaseConfig(t)
	filePath := filepath.Join(workingDir, "sample.txt")
	if err := os.WriteFile(filePath, []byte("alpha\nbeta\n"), 0o644); err != nil {
		t.Fatalf("write sample: %v", err)
	}

	resp, err := NewViewTool().Run(context.Background(), toolcore.ToolCall{
		ID:    "call-1",
		Name:  ViewToolName,
		Input: `{"file_path":"sample.txt"}`,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if resp.IsError {
		t.Fatalf("expected success response, got error: %s", resp.Content)
	}
	if !strings.Contains(resp.Content, "alpha") || !strings.Contains(resp.Content, "beta") {
		t.Fatalf("expected file content in response: %s", resp.Content)
	}
}
