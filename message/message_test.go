package message

import (
	"context"
	"testing"

	datadb "ferryman-agent/data/db"
	"ferryman-agent/data/repo"
	"ferryman-agent/llm/models"
)

func TestServiceUsesMessageRepoAndRoundTripsParts(t *testing.T) {
	ctx := context.Background()
	client, err := datadb.Open(datadb.DatabaseConfig{
		Type: datadb.DatabaseSQLite,
		Path: ":memory:",
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := client.AutoMigrate(&repo.SessionRecord{}, &repo.MessageRecord{}, &repo.HistoryRecord{}); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	sessions := repo.NewSessionRepo(client)
	messages := repo.NewMessageRepo(client)
	_, err = sessions.Create(ctx, repo.SessionRecord{ID: "s1", Title: "work"})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	service := NewService(messages)

	msg, err := service.Create(ctx, "s1", CreateMessageParams{
		Role:  User,
		Parts: []ContentPart{TextContent{Text: "hello"}},
		Model: models.ModelID("custom-model"),
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if msg.Model != "custom-model" || msg.Content().Text != "hello" {
		t.Fatalf("unexpected created message: %+v", msg)
	}
	msg.AddFinish(FinishReasonEndTurn)
	if err := service.Update(ctx, msg); err != nil {
		t.Fatalf("update: %v", err)
	}
	listed, err := service.List(ctx, "s1")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(listed) != 1 || listed[0].FinishReason() != FinishReasonEndTurn {
		t.Fatalf("unexpected listed messages: %+v", listed)
	}
	session, err := sessions.Get(ctx, "s1")
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if session.MessageCount != 1 {
		t.Fatalf("expected message count 1, got %d", session.MessageCount)
	}
}
