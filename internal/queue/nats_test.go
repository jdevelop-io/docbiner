package queue

import (
	"context"
	"sync"
	"testing"
	"time"
)

const testNATSURL = "nats://localhost:4222"

// cleanStream purges the stream and deletes the durable consumer so each test
// starts with a clean slate.
func cleanStream(t *testing.T, q *Queue) {
	t.Helper()
	ctx := context.Background()

	// Delete the durable consumer if it exists (ignore errors).
	_ = q.stream.DeleteConsumer(ctx, "worker")

	// Purge all messages from the stream.
	if err := q.stream.Purge(ctx); err != nil {
		t.Fatalf("failed to purge stream: %v", err)
	}
}

func TestPublishAndConsume(t *testing.T) {
	q, err := New(testNATSURL)
	if err != nil {
		t.Fatalf("failed to create queue: %v", err)
	}
	defer q.Close()

	cleanStream(t, q)

	msg := JobMessage{
		JobID: "job-001",
		Type:  "convert",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := q.Publish(ctx, msg); err != nil {
		t.Fatalf("failed to publish: %v", err)
	}

	var received JobMessage
	done := make(chan struct{})

	go func() {
		err := q.Subscribe(ctx, func(m JobMessage) error {
			received = m
			close(done)
			cancel() // stop subscribing after first message
			return nil
		})
		if err != nil && ctx.Err() == nil {
			t.Errorf("subscribe error: %v", err)
		}
	}()

	select {
	case <-done:
		if received.JobID != msg.JobID {
			t.Errorf("expected JobID %q, got %q", msg.JobID, received.JobID)
		}
		if received.Type != msg.Type {
			t.Errorf("expected Type %q, got %q", msg.Type, received.Type)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for message")
	}
}

func TestPublishMultiple(t *testing.T) {
	q, err := New(testNATSURL)
	if err != nil {
		t.Fatalf("failed to create queue: %v", err)
	}
	defer q.Close()

	cleanStream(t, q)

	messages := []JobMessage{
		{JobID: "job-101", Type: "convert"},
		{JobID: "job-102", Type: "merge"},
		{JobID: "job-103", Type: "convert"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for _, msg := range messages {
		if err := q.Publish(ctx, msg); err != nil {
			t.Fatalf("failed to publish %s: %v", msg.JobID, err)
		}
	}

	var mu sync.Mutex
	var received []JobMessage

	go func() {
		err := q.Subscribe(ctx, func(m JobMessage) error {
			mu.Lock()
			received = append(received, m)
			count := len(received)
			mu.Unlock()
			if count >= len(messages) {
				cancel() // stop after receiving all messages
			}
			return nil
		})
		if err != nil && ctx.Err() == nil {
			t.Errorf("subscribe error: %v", err)
		}
	}()

	<-ctx.Done()

	mu.Lock()
	defer mu.Unlock()

	if len(received) != len(messages) {
		t.Fatalf("expected %d messages, got %d", len(messages), len(received))
	}

	for i, msg := range messages {
		if received[i].JobID != msg.JobID {
			t.Errorf("message %d: expected JobID %q, got %q", i, msg.JobID, received[i].JobID)
		}
		if received[i].Type != msg.Type {
			t.Errorf("message %d: expected Type %q, got %q", i, msg.Type, received[i].Type)
		}
	}
}
