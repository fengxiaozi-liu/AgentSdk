package session

import (
	"context"
	"testing"

	datadb "ferryman-agent/data/db"
	"ferryman-agent/data/repo"
)

func TestServiceUsesSessionRepo(t *testing.T) {
	ctx := context.Background()
	repos := repo.NewRepositories(datadb.NewSource())
	service := NewService(repos.Sessions)

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
