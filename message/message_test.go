package message

import (
	"context"
	"testing"

	"ferryman-agent/llm/models"
	"ferryman-agent/session"
	"ferryman-agent/testutil"
)

func TestServiceCreateAndList(t *testing.T) {
	ctx := context.Background()
	harness := testutil.NewDBHarness(t)

	sessions := session.NewService(harness.Queries)
	msgs := NewService(harness.Queries)

	sess, err := sessions.Create(ctx, "chat")
	if err != nil {
		t.Fatalf("Create session: %v", err)
	}

	created, err := msgs.Create(ctx, sess.ID, CreateMessageParams{
		Role:  User,
		Model: models.GPT41,
		Parts: []ContentPart{TextContent{Text: "hello"}},
	})
	if err != nil {
		t.Fatalf("Create message: %v", err)
	}

	listed, err := msgs.List(ctx, sess.ID)
	if err != nil {
		t.Fatalf("List messages: %v", err)
	}

	if len(listed) != 1 {
		t.Fatalf("expected 1 message, got %d", len(listed))
	}
	if listed[0].ID != created.ID {
		t.Fatalf("expected message id %q, got %q", created.ID, listed[0].ID)
	}
	if listed[0].Role != User {
		t.Fatalf("expected role %q, got %q", User, listed[0].Role)
	}
	if finish := listed[0].FinishPart(); finish == nil || finish.Reason != "stop" {
		t.Fatalf("expected stop finish part, got %+v", finish)
	}
}
