package session

import (
	"context"
	"testing"

	"ferryman-agent/testutil"
)

func TestServiceCreateSaveAndList(t *testing.T) {
	ctx := context.Background()
	harness := testutil.NewDBHarness(t)
	service := NewService(harness.Queries)

	created, err := service.Create(ctx, "chat")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	created.Title = "updated"
	created.PromptTokens = 12
	created.CompletionTokens = 34

	saved, err := service.Save(ctx, created)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	if saved.Title != "updated" {
		t.Fatalf("expected updated title, got %q", saved.Title)
	}

	listed, err := service.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(listed) != 1 {
		t.Fatalf("expected 1 session, got %d", len(listed))
	}
	if listed[0].ID != created.ID {
		t.Fatalf("expected session id %q, got %q", created.ID, listed[0].ID)
	}
	if listed[0].PromptTokens != 12 || listed[0].CompletionTokens != 34 {
		t.Fatalf("unexpected token counts: %+v", listed[0])
	}
}
