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
	workingDir := t.TempDir()
	if _, err := config.Use(config.Config{WorkingDir: workingDir}); err != nil {
		t.Fatalf("use config: %v", err)
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
