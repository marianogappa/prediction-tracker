package statemachine

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/marianogappa/predictions-tracker/domain"
)

var validTransitions = map[domain.PredictionState]map[domain.PredictionState]bool{
	domain.StateDraft: {
		domain.StateEnabled: true,
	},
	domain.StateEnabled: {
		domain.StateMonitoring: true,
		domain.StateDisabled:   true,
	},
	domain.StateMonitoring: {
		domain.StateFinalCorrect:    true,
		domain.StateFinalIncorrect:  true,
		domain.StateFinalUnresolved: true,
	},
	domain.StateDisabled: {
		domain.StateEnabled: true,
	},
}

var stateToEventType = map[domain.PredictionState]domain.EventType{
	domain.StateEnabled:         domain.EventPredictionEnabled,
	domain.StateMonitoring:      domain.EventMonitoringStarted,
	domain.StateDisabled:        domain.EventPredictionDisabled,
	domain.StateFinalCorrect:    domain.EventPredictionCorrect,
	domain.StateFinalIncorrect:  domain.EventPredictionIncorrect,
	domain.StateFinalUnresolved: domain.EventPredictionUnresolved,
	domain.StateErrored:         domain.EventPredictionErrored,
}

type TransitionPayload struct {
	FromState domain.PredictionState `json:"from_state"`
	ToState   domain.PredictionState `json:"to_state"`
	Reason    string                 `json:"reason,omitempty"`
}

// Transition validates and performs a state transition on a prediction.
// It returns the emitted event on success.
func Transition(pred *domain.Prediction, target domain.PredictionState, now time.Time, reason string) (domain.Event, error) {
	if pred.State.IsFinal() {
		return domain.Event{}, errors.New("prediction is in a terminal state")
	}

	if target == domain.StateErrored {
		return applyTransition(pred, target, now, reason)
	}

	allowed, ok := validTransitions[pred.State]
	if !ok || !allowed[target] {
		return domain.Event{}, fmt.Errorf("invalid transition from %s to %s", pred.State, target)
	}

	return applyTransition(pred, target, now, reason)
}

func applyTransition(pred *domain.Prediction, target domain.PredictionState, now time.Time, reason string) (domain.Event, error) {
	payload := TransitionPayload{
		FromState: pred.State,
		ToState:   target,
		Reason:    reason,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return domain.Event{}, fmt.Errorf("marshaling transition payload: %w", err)
	}

	eventType, ok := stateToEventType[target]
	if !ok {
		eventType = domain.EventPredictionFinalized
	}

	evt := domain.Event{
		ID:           uuid.New().String(),
		PredictionID: pred.ID,
		Type:         eventType,
		Timestamp:    now,
		Payload:      payloadJSON,
	}

	pred.State = target
	pred.UpdatedAt = now

	return evt, nil
}
