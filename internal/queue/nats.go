package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

const (
	StreamName  = "DOCBINER"
	SubjectJobs = "docbiner.jobs"
)

// JobMessage represents a job to be processed by a worker.
type JobMessage struct {
	JobID string `json:"job_id"`
	Type  string `json:"type"` // "convert", "merge"
}

// Queue wraps a NATS JetStream connection for publishing and consuming job messages.
type Queue struct {
	nc     *nats.Conn
	js     jetstream.JetStream
	stream jetstream.Stream
}

// New connects to NATS, creates a JetStream context, and ensures the DOCBINER
// stream exists. It returns a ready-to-use Queue.
func New(url string) (*Queue, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("jetstream new: %w", err)
	}

	stream, err := js.CreateOrUpdateStream(context.Background(), jetstream.StreamConfig{
		Name:     StreamName,
		Subjects: []string{"docbiner.>"},
	})
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("create stream: %w", err)
	}

	return &Queue{
		nc:     nc,
		js:     js,
		stream: stream,
	}, nil
}

// Close drains and closes the NATS connection.
func (q *Queue) Close() {
	if q.nc != nil {
		q.nc.Close()
	}
}

// Publish serialises msg as JSON and publishes it to the jobs subject.
func (q *Queue) Publish(ctx context.Context, msg JobMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	if _, err := q.js.Publish(ctx, SubjectJobs, data); err != nil {
		return fmt.Errorf("publish: %w", err)
	}

	return nil
}

// Subscribe creates a durable pull consumer named "worker" and loops over
// incoming messages, calling handler for each one. Messages are Ack'd when
// the handler returns nil and Nak'd otherwise. The loop stops when ctx is
// cancelled.
func (q *Queue) Subscribe(ctx context.Context, handler func(JobMessage) error) error {
	consumer, err := q.stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:   "worker",
		AckPolicy: jetstream.AckExplicitPolicy,
	})
	if err != nil {
		return fmt.Errorf("create consumer: %w", err)
	}

	iter, err := consumer.Messages()
	if err != nil {
		return fmt.Errorf("messages iterator: %w", err)
	}
	defer iter.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		msg, err := iter.Next()
		if err != nil {
			// If context was cancelled, exit gracefully.
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("next message: %w", err)
		}

		var job JobMessage
		if err := json.Unmarshal(msg.Data(), &job); err != nil {
			_ = msg.Nak()
			continue
		}

		if err := handler(job); err != nil {
			_ = msg.Nak()
		} else {
			_ = msg.Ack()
		}
	}
}
