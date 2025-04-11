package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// EventType represents the type of workflow event.
type EventType string

const (
	EventWorkflowCreated    EventType = "WorkflowCreated"
	EventWorkflowUpdated    EventType = "WorkflowUpdated"
	EventWorkflowCompleted  EventType = "WorkflowCompleted"
	EventWorkflowDispatched EventType = "WorkflowDispatched"
	EventNodeUpdated        EventType = "NodeUpdated"
	EventNodeCompleted      EventType = "NodeCompleted"
	EventNodeDispatched     EventType = "NodeDispatched"
)

// Event represents a workflow or node event.
type Event struct {
	Type      EventType `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	ProtoType string    `json:"proto_type"` // e.g., "Node", "UpdateWorkflowRequest"
	Payload   []byte    `json:"payload"`    // serialized protobuf message
}

// EventLogger defines the interface for event logging.
type EventLogger interface {
	LogEvent(event Event)
}

// StdoutEventLogger logs events to stdout.
type StdoutEventLogger struct{}

// LogEvent logs the event as JSON to stdout.
func (l *StdoutEventLogger) LogEvent(event Event) {
	data, err := json.Marshal(event)
	if err != nil {
		fmt.Printf("Error marshaling event: %v\n", err)
		return
	}
	fmt.Println(string(data))
}

// PostgresEventLogger logs events into a Postgres table.
type PostgresEventLogger struct {
	DB *sql.DB
}

// LogEvent inserts the event into the events table.
func (l *PostgresEventLogger) LogEvent(event Event) {
	_, err := l.DB.Exec(
		`INSERT INTO events (type, timestamp, proto_type, payload) VALUES ($1, $2, $3, $4)`,
		string(event.Type),
		event.Timestamp,
		event.ProtoType,
		event.Payload,
	)
	if err != nil {
		log.Printf("Failed to insert event: %v", err)
	}
}
