package domain

import (
	"encoding/json"
	"time"
)

type EventType string

const (
	EventPredictionIngested   EventType = "prediction_ingested"
	EventPredictionEnabled    EventType = "prediction_enabled"
	EventPredictionDisabled   EventType = "prediction_disabled"
	EventValueFetched         EventType = "value_fetched"
	EventEvaluationPerformed  EventType = "evaluation_performed"
	EventPredictionFinalized  EventType = "prediction_finalized"
	EventPredictionCorrect    EventType = "prediction_correct"
	EventPredictionIncorrect  EventType = "prediction_incorrect"
	EventPredictionUnresolved EventType = "prediction_unresolved"
	EventPredictionErrored    EventType = "prediction_errored"
	EventMonitoringStarted    EventType = "monitoring_started"
)

type Event struct {
	ID           string          `json:"id"`
	PredictionID string          `json:"prediction_id"`
	Type         EventType       `json:"type"`
	Timestamp    time.Time       `json:"timestamp"`
	Payload      json.RawMessage `json:"payload,omitempty"`
}
