package session

import (
	"context"
	"testing"

	datadb "ferryman-agent/internal/data/db"
	"ferryman-agent/internal/data/repo"
)

func TestServiceUsesSessionRepo(t *testing.T) {
	ctx := context.Background()
	client, err := datadb.NewDbClient(datadb.DatabaseConfig{
		Type: datadb.DatabaseSQLite,
		Path: ":memory:",
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := client.AutoMigrate(&repo.SessionRecord{}, &repo.MessageRecord{}, &repo.HistoryRecord{}); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	service := NewService(repo.NewSessionRepo(client))

	session, err := service.Create(ctx, "work")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	session.Title = "updated"
	session.SummaryMessageID = "summary-1"
	session.PromptTokens = 10
	session.CompletionTokens = 20
	updated, err := service.Save(ctx, session)
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if updated.Title != "updated" || updated.SummaryMessageID != "summary-1" {
		t.Fatalf("unexpected updated session: %+v", updated)
	}
	listed, err := service.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("expected one root session, got %d", len(listed))
	}
}
