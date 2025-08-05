package logging

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"mix/internal/pubsub"

	"github.com/go-logfmt/logfmt"
)

// Removed persist constants for embedded binary

type LogData struct {
	messages []LogMessage
	*pubsub.Broker[LogMessage]
	lock sync.Mutex
}

func (l *LogData) Add(ctx context.Context, msg LogMessage) error {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.messages = append(l.messages, msg)
	return l.Publish(ctx, pubsub.CreatedEvent, msg)
}

func (l *LogData) List() []LogMessage {
	l.lock.Lock()
	defer l.lock.Unlock()
	return l.messages
}

var defaultLogData = &LogData{
	messages: make([]LogMessage, 0),
	Broker:   pubsub.NewBroker[LogMessage](),
}

type writer struct{}

func (w *writer) Write(p []byte) (int, error) {
	// First, write to stdout so it gets captured by shoreman
	if _, err := os.Stdout.Write(p); err != nil {
		return 0, fmt.Errorf("writing to stdout: %w", err)
	}

	// Then parse and store the log message for internal use
	d := logfmt.NewDecoder(bytes.NewReader(p))

	for d.ScanRecord() {
		msg := LogMessage{
			ID:   fmt.Sprintf("%d", time.Now().UnixNano()),
			Time: time.Now(),
		}
		for d.ScanKeyval() {
			switch string(d.Key()) {
			case "time":
				parsed, err := time.Parse(time.RFC3339, string(d.Value()))
				if err != nil {
					return 0, fmt.Errorf("parsing time: %w", err)
				}
				msg.Time = parsed
			case "level":
				msg.Level = strings.ToLower(string(d.Value()))
			case "msg":
				msg.Message = string(d.Value())
			default:
				msg.Attributes = append(msg.Attributes, Attr{
					Key:   string(d.Key()),
					Value: string(d.Value()),
				})
			}
		}
		if err := defaultLogData.Add(context.Background(), msg); err != nil {
			// Log publish error but don't fail the write operation
			fmt.Fprintf(os.Stderr, "failed to publish log message: %v\n", err)
		}
	}
	if d.Err() != nil {
		return 0, d.Err()
	}
	return len(p), nil
}

func NewWriter() *writer {
	w := &writer{}
	return w
}

func Subscribe(ctx context.Context) <-chan pubsub.Event[LogMessage] {
	return defaultLogData.Subscribe(ctx)
}

func List() []LogMessage {
	return defaultLogData.List()
}
