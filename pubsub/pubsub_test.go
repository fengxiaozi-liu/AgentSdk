package pubsub

import (
	"context"
	"testing"
	"time"
)

func TestBrokerPublishDeliversEvent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	broker := NewBroker[string]()
	events := broker.Subscribe(ctx)
	broker.Publish(CreatedEvent, "hello")

	select {
	case event := <-events:
		if event.Type != CreatedEvent {
			t.Fatalf("expected event type %q, got %q", CreatedEvent, event.Type)
		}
		if event.Payload != "hello" {
			t.Fatalf("expected payload hello, got %q", event.Payload)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for event")
	}
}
