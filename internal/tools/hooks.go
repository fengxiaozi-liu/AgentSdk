package tools

import (
	"context"
	"encoding/json"
	"fmt"
)

type FileEventType string

const (
	FileViewed  FileEventType = "file.viewed"
	FileEdited  FileEventType = "file.edited"
	FileWritten FileEventType = "file.written"
	FilePatched FileEventType = "file.patched"
	FileDeleted FileEventType = "file.deleted"
)

type FileEvent struct {
	Type       FileEventType  `json:"type"`
	ToolName   string         `json:"tool_name,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
	SessionID  string         `json:"session_id,omitempty"`
	MessageID  string         `json:"message_id,omitempty"`
	Path       string         `json:"path,omitempty"`
	Paths      []string       `json:"paths,omitempty"`
	OldContent string         `json:"old_content,omitempty"`
	NewContent string         `json:"new_content,omitempty"`
	Content    string         `json:"content,omitempty"`
	Diff       string         `json:"diff,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

type HookResult struct {
	Content  string         `json:"content,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
	Error    string         `json:"error,omitempty"`
}

type FileHook interface {
	OnFileEvent(context.Context, FileEvent) (*HookResult, error)
}

type FileHookDispatcher struct {
	hooks []FileHook
}

func NewFileHookDispatcher(hooks ...FileHook) *FileHookDispatcher {
	registered := make([]FileHook, 0, len(hooks))
	for _, hook := range hooks {
		if hook != nil {
			registered = append(registered, hook)
		}
	}
	return &FileHookDispatcher{hooks: registered}
}

func (d *FileHookDispatcher) Dispatch(ctx context.Context, event FileEvent) []HookResult {
	if d == nil || len(d.hooks) == 0 {
		return nil
	}
	results := make([]HookResult, 0, len(d.hooks))
	for _, hook := range d.hooks {
		result, err := hook.OnFileEvent(ctx, event)
		if err != nil {
			results = append(results, HookResult{Error: err.Error()})
			continue
		}
		if result != nil {
			results = append(results, *result)
		}
	}
	return results
}

func WithHookResults(response ToolResponse, results []HookResult) ToolResponse {
	if len(results) == 0 {
		return response
	}

	for _, result := range results {
		if result.Content != "" {
			if response.Content != "" {
				response.Content += "\n"
			}
			response.Content += result.Content
		}
	}

	metadata := map[string]any{
		"hooks": results,
	}
	if response.Metadata != "" {
		var raw json.RawMessage
		if err := json.Unmarshal([]byte(response.Metadata), &raw); err == nil {
			metadata["tool"] = raw
		} else {
			metadata["tool"] = response.Metadata
		}
	}
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		response.Metadata = fmt.Sprintf(`{"hooks":[{"error":%q}]}`, err.Error())
		return response
	}
	response.Metadata = string(metadataBytes)
	return response
}
