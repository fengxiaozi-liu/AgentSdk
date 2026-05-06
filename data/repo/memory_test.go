package repo

import (
	"context"
	"testing"

	"ferryman-agent/config"
	datadb "ferryman-agent/data/db"
)

func TestRepositoriesCoverSessionMessageAndHistory(t *testing.T) {
	ctx := context.Background()
	database, err := datadb.Open(config.DatabaseConfig{
		Type:        config.DatabaseSQLite,
		Path:        ":memory:",
		AutoMigrate: true,
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	sessions := NewSessionRepo(database)
	messages := NewMessageRepo(database)
	history := NewHistoryRepo(database)

	session, err := sessions.Create(ctx, CreateSessionParams{ID: "s1", Title: "root"})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if session.Title != "root" {
		t.Fatalf("unexpected session title: %q", session.Title)
	}

	message, err := messages.Create(ctx, CreateMessageParams{
		ID:        "m1",
		SessionID: "s1",
		Role:      "user",
		Parts:     `[{"type":"text","data":{"text":"hello"}}]`,
		Model:     "test-model",
	})
	if err != nil {
		t.Fatalf("create message: %v", err)
	}
	if message.Model != "test-model" {
		t.Fatalf("unexpected message model: %q", message.Model)
	}
	session, err = sessions.Get(ctx, "s1")
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if session.MessageCount != 1 {
		t.Fatalf("expected message count 1, got %d", session.MessageCount)
	}

	file, err := history.Create(ctx, CreateFileParams{
		ID:        "f1",
		SessionID: "s1",
		Path:      "file.txt",
		Content:   "one",
		Version:   "initial",
	})
	if err != nil {
		t.Fatalf("create file: %v", err)
	}
	if file.Version != "initial" {
		t.Fatalf("unexpected file version: %q", file.Version)
	}
	_, err = history.Create(ctx, CreateFileParams{
		ID:        "f2",
		SessionID: "s1",
		Path:      "file.txt",
		Content:   "two",
		Version:   "v1",
	})
	if err != nil {
		t.Fatalf("create file version: %v", err)
	}
	latest, err := history.GetLatestByPathAndSession(ctx, "file.txt", "s1")
	if err != nil {
		t.Fatalf("get latest file: %v", err)
	}
	if latest.Content != "two" {
		t.Fatalf("expected latest content two, got %q", latest.Content)
	}
}
