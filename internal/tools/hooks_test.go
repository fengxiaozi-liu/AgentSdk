package tools

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

type hookFunc func(context.Context, FileEvent) (*HookResult, error)

func (f hookFunc) OnFileEvent(ctx context.Context, event FileEvent) (*HookResult, error) {
	return f(ctx, event)
}

func TestFileHookDispatcherCollectsResultsAndErrors(t *testing.T) {
	dispatcher := NewFileHookDispatcher(
		hookFunc(func(context.Context, FileEvent) (*HookResult, error) {
			return &HookResult{Content: "ok", Metadata: map[string]any{"name": "first"}}, nil
		}),
		hookFunc(func(context.Context, FileEvent) (*HookResult, error) {
			return nil, errors.New("hook failed")
		}),
	)

	results := dispatcher.Dispatch(context.Background(), FileEvent{Type: FileViewed})
	if len(results) != 2 {
		t.Fatalf("expected 2 hook results, got %d", len(results))
	}
	if results[0].Content != "ok" {
		t.Fatalf("unexpected first hook content: %q", results[0].Content)
	}
	if results[1].Error != "hook failed" {
		t.Fatalf("unexpected hook error: %q", results[1].Error)
	}
}

func TestWithHookResultsMergesContentAndMetadata(t *testing.T) {
	response := WithResponseMetadata(NewTextResponse("tool response"), map[string]any{"diff": "sample"})

	merged := WithHookResults(response, []HookResult{{
		Content:  "hook response",
		Metadata: map[string]any{"source": "test"},
	}})

	if !strings.Contains(merged.Content, "tool response") || !strings.Contains(merged.Content, "hook response") {
		t.Fatalf("expected tool and hook content in merged response: %q", merged.Content)
	}

	var metadata map[string]any
	if err := json.Unmarshal([]byte(merged.Metadata), &metadata); err != nil {
		t.Fatalf("metadata should be valid JSON: %v", err)
	}
	if metadata["tool"] == nil {
		t.Fatalf("expected original tool metadata to be preserved: %s", merged.Metadata)
	}
	hooks, ok := metadata["hooks"].([]any)
	if !ok || len(hooks) != 1 {
		t.Fatalf("expected one hook metadata entry: %s", merged.Metadata)
	}
}
