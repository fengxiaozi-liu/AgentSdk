package permission

import (
	"context"
	"testing"
	"time"
)

func TestRequestPublishesAndGrantResolves(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	service := NewService()
	events := service.Subscribe(ctx)

	done := make(chan bool, 1)
	go func() {
		done <- service.Request(CreatePermissionRequest{
			SessionID:   "s1",
			ToolName:    "write",
			Action:      "write",
			Description: "update file",
			Path:        "/tmp/project/file.txt",
		})
	}()

	select {
	case event := <-events:
		service.Grant(event.Payload)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for permission event")
	}

	select {
	case granted := <-done:
		if !granted {
			t.Fatal("expected permission request to be granted")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for permission response")
	}
}
