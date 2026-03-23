package event

import (
	"testing"
	"time"

	"github.com/marianogappa/predictions-tracker/domain"
)

func TestBusPublishesToAllSubscribers(t *testing.T) {
	bus := NewBus()

	var received1, received2 []domain.Event
	bus.Subscribe(func(e domain.Event) { received1 = append(received1, e) })
	bus.Subscribe(func(e domain.Event) { received2 = append(received2, e) })

	evt := domain.Event{
		ID:           "e1",
		PredictionID: "p1",
		Type:         domain.EventPredictionEnabled,
		Timestamp:    time.Now(),
	}
	bus.Publish(evt)

	if len(received1) != 1 || received1[0].ID != "e1" {
		t.Fatalf("subscriber 1: expected 1 event with ID e1, got %v", received1)
	}
	if len(received2) != 1 || received2[0].ID != "e1" {
		t.Fatalf("subscriber 2: expected 1 event with ID e1, got %v", received2)
	}
}

func TestBusPreservesOrdering(t *testing.T) {
	bus := NewBus()

	var order []string
	bus.Subscribe(func(e domain.Event) { order = append(order, "first:"+e.ID) })
	bus.Subscribe(func(e domain.Event) { order = append(order, "second:"+e.ID) })

	bus.Publish(domain.Event{ID: "a"})
	bus.Publish(domain.Event{ID: "b"})

	expected := []string{"first:a", "second:a", "first:b", "second:b"}
	if len(order) != len(expected) {
		t.Fatalf("expected %d calls, got %d", len(expected), len(order))
	}
	for i, v := range expected {
		if order[i] != v {
			t.Fatalf("at index %d: expected %s, got %s", i, v, order[i])
		}
	}
}

func TestBusNoSubscribers(t *testing.T) {
	bus := NewBus()
	bus.Publish(domain.Event{ID: "lonely"})
}
