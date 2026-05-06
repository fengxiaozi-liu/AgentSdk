package base

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"ferryman-agent/history"
	"ferryman-agent/permission"
	"ferryman-agent/pubsub"
	toolcore "ferryman-agent/tools/core"
)

type recordingFileHook struct {
	events []toolcore.FileEvent
}

func (h *recordingFileHook) OnFileEvent(ctx context.Context, event toolcore.FileEvent) (*toolcore.HookResult, error) {
	h.events = append(h.events, event)
	return &toolcore.HookResult{
		Content: "hook:" + string(event.Type),
		Metadata: map[string]any{
			"type": string(event.Type),
			"path": event.Path,
		},
	}, nil
}

type fakeHistoryService struct {
	*pubsub.Broker[history.File]
	files map[string]history.File
}

func newFakeHistoryService() *fakeHistoryService {
	return &fakeHistoryService{
		Broker: pubsub.NewBroker[history.File](),
		files:  map[string]history.File{},
	}
}

func (s *fakeHistoryService) Create(ctx context.Context, sessionID, path, content string) (history.File, error) {
	file := history.File{ID: path, SessionID: sessionID, Path: path, Content: content, Version: history.InitialVersion}
	s.files[path] = file
	return file, nil
}

func (s *fakeHistoryService) CreateVersion(ctx context.Context, sessionID, path, content string) (history.File, error) {
	file := history.File{ID: path, SessionID: sessionID, Path: path, Content: content, Version: "v1"}
	s.files[path] = file
	return file, nil
}

func (s *fakeHistoryService) Get(ctx context.Context, id string) (history.File, error) {
	file, ok := s.files[id]
	if !ok {
		return history.File{}, sql.ErrNoRows
	}
	return file, nil
}

func (s *fakeHistoryService) GetByPathAndSession(ctx context.Context, path, sessionID string) (history.File, error) {
	file, ok := s.files[path]
	if !ok || file.SessionID != sessionID {
		return history.File{}, sql.ErrNoRows
	}
	return file, nil
}

func (s *fakeHistoryService) ListBySession(ctx context.Context, sessionID string) ([]history.File, error) {
	files := []history.File{}
	for _, file := range s.files {
		if file.SessionID == sessionID {
			files = append(files, file)
		}
	}
	return files, nil
}

func (s *fakeHistoryService) ListLatestSessionFiles(ctx context.Context, sessionID string) ([]history.File, error) {
	return s.ListBySession(ctx, sessionID)
}

func (s *fakeHistoryService) Update(ctx context.Context, file history.File) (history.File, error) {
	s.files[file.Path] = file
	return file, nil
}

func (s *fakeHistoryService) Delete(ctx context.Context, id string) error {
	delete(s.files, id)
	return nil
}

func (s *fakeHistoryService) DeleteSessionFiles(ctx context.Context, sessionID string) error {
	for path, file := range s.files {
		if file.SessionID == sessionID {
			delete(s.files, path)
		}
	}
	return nil
}

func hookTestContext() context.Context {
	ctx := context.WithValue(context.Background(), toolcore.SessionIDContextKey, "session-1")
	return context.WithValue(ctx, toolcore.MessageIDContextKey, "message-1")
}

func hookTestPermission() permission.Service {
	permissions := permission.NewService()
	permissions.AutoApproveSession("session-1")
	return permissions
}

func assertHookResponse(t *testing.T, resp toolcore.ToolResponse, eventType toolcore.FileEventType) {
	t.Helper()
	if resp.IsError {
		t.Fatalf("expected success response, got error: %s", resp.Content)
	}
	if !strings.Contains(resp.Content, "hook:"+string(eventType)) {
		t.Fatalf("expected hook content in response: %s", resp.Content)
	}
	var metadata map[string]any
	if err := json.Unmarshal([]byte(resp.Metadata), &metadata); err != nil {
		t.Fatalf("metadata should be valid JSON: %v", err)
	}
	if metadata["tool"] == nil || metadata["hooks"] == nil {
		t.Fatalf("expected tool and hooks metadata: %s", resp.Metadata)
	}
}

func TestFileToolHooksPublishEvents(t *testing.T) {
	workingDir := ensureBaseConfig(t)
	ctx := hookTestContext()
	permissions := hookTestPermission()
	files := newFakeHistoryService()
	hook := &recordingFileHook{}

	viewPath := filepath.Join(workingDir, "hook_view.txt")
	if err := writeTestFile(viewPath, "alpha\nbeta\n"); err != nil {
		t.Fatalf("write view file: %v", err)
	}
	viewResp, err := NewViewTool(hook).Run(ctx, toolcore.ToolCall{
		ID:    "view-call",
		Name:  ViewToolName,
		Input: `{"file_path":"hook_view.txt"}`,
	})
	if err != nil {
		t.Fatalf("view run: %v", err)
	}
	assertHookResponse(t, viewResp, toolcore.FileViewed)

	writeResp, err := NewWriteTool(permissions, files, hook).Run(ctx, toolcore.ToolCall{
		ID:    "write-call",
		Name:  WriteToolName,
		Input: `{"file_path":"hook_write.txt","content":"created\n"}`,
	})
	if err != nil {
		t.Fatalf("write run: %v", err)
	}
	assertHookResponse(t, writeResp, toolcore.FileWritten)

	editPath := filepath.Join(workingDir, "hook_edit.txt")
	if err := writeTestFile(editPath, "old\nkeep\n"); err != nil {
		t.Fatalf("write edit file: %v", err)
	}
	if _, err := NewViewTool().Run(ctx, toolcore.ToolCall{ID: "read-edit", Name: ViewToolName, Input: `{"file_path":"hook_edit.txt"}`}); err != nil {
		t.Fatalf("read edit file: %v", err)
	}
	editResp, err := NewEditTool(permissions, files, hook).Run(ctx, toolcore.ToolCall{
		ID:    "edit-call",
		Name:  EditToolName,
		Input: `{"file_path":"hook_edit.txt","old_string":"old","new_string":"new"}`,
	})
	if err != nil {
		t.Fatalf("edit run: %v", err)
	}
	assertHookResponse(t, editResp, toolcore.FileEdited)

	patchPath := filepath.Join(workingDir, "hook_patch.txt")
	if err := writeTestFile(patchPath, "hello\nold\nbye"); err != nil {
		t.Fatalf("write patch file: %v", err)
	}
	if _, err := NewViewTool().Run(ctx, toolcore.ToolCall{ID: "read-patch", Name: ViewToolName, Input: `{"file_path":"hook_patch.txt"}`}); err != nil {
		t.Fatalf("read patch file: %v", err)
	}
	patchResp, err := NewPatchTool(permissions, files, hook).Run(ctx, toolcore.ToolCall{
		ID:    "patch-call",
		Name:  PatchToolName,
		Input: `{"patch_text":"*** Begin Patch\n*** Update File: hook_patch.txt\n@@\n hello\n-old\n+new\n bye\n*** End Patch"}`,
	})
	if err != nil {
		t.Fatalf("patch run: %v", err)
	}
	assertHookResponse(t, patchResp, toolcore.FilePatched)

	eventTypes := []toolcore.FileEventType{}
	for _, event := range hook.events {
		eventTypes = append(eventTypes, event.Type)
		if event.SessionID != "session-1" || event.MessageID != "message-1" {
			t.Fatalf("expected session/message IDs on event: %+v", event)
		}
		if event.ToolCallID == "" {
			t.Fatalf("expected tool call ID on event: %+v", event)
		}
		if event.Path == "" {
			t.Fatalf("expected path on event: %+v", event)
		}
	}
	for _, eventType := range []toolcore.FileEventType{toolcore.FileViewed, toolcore.FileWritten, toolcore.FileEdited, toolcore.FilePatched} {
		if !slices.Contains(eventTypes, eventType) {
			t.Fatalf("expected event type %s in %v", eventType, eventTypes)
		}
	}
}

func writeTestFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}
