package statemachine

import (
	"testing"
	"time"

	"github.com/marianogappa/predictions-tracker/domain"
)

func TestValidTransitions(t *testing.T) {
	now := time.Now()
	cases := []struct {
		from   domain.PredictionState
		to     domain.PredictionState
		expect domain.EventType
	}{
		{domain.StateDraft, domain.StateEnabled, domain.EventPredictionEnabled},
		{domain.StateEnabled, domain.StateMonitoring, domain.EventMonitoringStarted},
		{domain.StateEnabled, domain.StateDisabled, domain.EventPredictionDisabled},
		{domain.StateDisabled, domain.StateEnabled, domain.EventPredictionEnabled},
		{domain.StateMonitoring, domain.StateFinalCorrect, domain.EventPredictionCorrect},
		{domain.StateMonitoring, domain.StateFinalIncorrect, domain.EventPredictionIncorrect},
		{domain.StateMonitoring, domain.StateFinalUnresolved, domain.EventPredictionUnresolved},
	}

	for _, tc := range cases {
		t.Run(string(tc.from)+"->"+string(tc.to), func(t *testing.T) {
			pred := &domain.Prediction{ID: "test", State: tc.from}
			evt, err := Transition(pred, tc.to, now, "")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if pred.State != tc.to {
				t.Fatalf("expected state %s, got %s", tc.to, pred.State)
			}
			if evt.Type != tc.expect {
				t.Fatalf("expected event type %s, got %s", tc.expect, evt.Type)
			}
			if evt.PredictionID != "test" {
				t.Fatalf("expected prediction ID 'test', got %s", evt.PredictionID)
			}
		})
	}
}

func TestInvalidTransitions(t *testing.T) {
	now := time.Now()
	cases := []struct {
		from domain.PredictionState
		to   domain.PredictionState
	}{
		{domain.StateDraft, domain.StateMonitoring},
		{domain.StateDraft, domain.StateFinalCorrect},
		{domain.StateDraft, domain.StateDisabled},
		{domain.StateEnabled, domain.StateFinalCorrect},
		{domain.StateMonitoring, domain.StateEnabled},
		{domain.StateMonitoring, domain.StateDisabled},
		{domain.StateDisabled, domain.StateMonitoring},
		{domain.StateDisabled, domain.StateFinalCorrect},
	}

	for _, tc := range cases {
		t.Run(string(tc.from)+"->"+string(tc.to), func(t *testing.T) {
			pred := &domain.Prediction{ID: "test", State: tc.from}
			_, err := Transition(pred, tc.to, now, "")
			if err == nil {
				t.Fatalf("expected error for transition %s -> %s", tc.from, tc.to)
			}
			if pred.State != tc.from {
				t.Fatalf("state should not have changed, expected %s, got %s", tc.from, pred.State)
			}
		})
	}
}

func TestErroredTransitionFromAnyNonFinal(t *testing.T) {
	now := time.Now()
	nonFinal := []domain.PredictionState{
		domain.StateDraft,
		domain.StateEnabled,
		domain.StateMonitoring,
		domain.StateDisabled,
	}

	for _, from := range nonFinal {
		t.Run(string(from)+"->errored", func(t *testing.T) {
			pred := &domain.Prediction{ID: "test", State: from}
			evt, err := Transition(pred, domain.StateErrored, now, "something broke")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if pred.State != domain.StateErrored {
				t.Fatalf("expected errored state, got %s", pred.State)
			}
			if evt.Type != domain.EventPredictionErrored {
				t.Fatalf("expected event type %s, got %s", domain.EventPredictionErrored, evt.Type)
			}
		})
	}
}

func TestFinalStateRejectsAllTransitions(t *testing.T) {
	now := time.Now()
	finals := []domain.PredictionState{
		domain.StateFinalCorrect,
		domain.StateFinalIncorrect,
		domain.StateFinalUnresolved,
	}
	targets := []domain.PredictionState{
		domain.StateDraft, domain.StateEnabled, domain.StateMonitoring,
		domain.StateErrored, domain.StateFinalCorrect,
	}

	for _, from := range finals {
		for _, to := range targets {
			t.Run(string(from)+"->"+string(to), func(t *testing.T) {
				pred := &domain.Prediction{ID: "test", State: from}
				_, err := Transition(pred, to, now, "")
				if err == nil {
					t.Fatalf("expected error transitioning from final state %s", from)
				}
			})
		}
	}
}
