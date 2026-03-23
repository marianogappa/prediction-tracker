package sqlite

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/marianogappa/predictions-tracker/domain"
	"github.com/marianogappa/predictions-tracker/storage"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestPredictionRoundTrip(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	pred := domain.Prediction{
		ID:         "pred-1",
		Statement:  "BTC will reach 100k",
		Rule:       domain.Rule{Type: domain.RuleThreshold, Operator: ">=", PriceField: "close", Value: 100000, Deadline: now.Add(24 * time.Hour)},
		Asset:      "BINANCE:BTC/USDT",
		StartTime:  now,
		Deadline:   now.Add(24 * time.Hour),
		SourceURL:  "https://example.com",
		AuthorName: "Alice",
		AuthorURL:  "https://twitter.com/alice",
		State:      domain.StateDraft,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := s.InsertPrediction(ctx, pred); err != nil {
		t.Fatalf("insert: %v", err)
	}

	got, err := s.GetPrediction(ctx, "pred-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ID != pred.ID || got.Statement != pred.Statement || got.Asset != pred.Asset {
		t.Fatalf("round-trip mismatch: got %+v", got)
	}
	if got.State != domain.StateDraft {
		t.Fatalf("expected draft state, got %s", got.State)
	}
	if got.Rule.Type != domain.RuleThreshold || got.Rule.Value != 100000 {
		t.Fatalf("rule mismatch: %+v", got.Rule)
	}
}

func TestUpdatePrediction(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	pred := domain.Prediction{
		ID: "pred-1", Statement: "test", Asset: "BINANCE:BTC/USDT",
		Rule: domain.Rule{Type: domain.RuleThreshold, PriceField: "close", Deadline: now},
		StartTime: now, Deadline: now, State: domain.StateDraft,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := s.InsertPrediction(ctx, pred); err != nil {
		t.Fatalf("insert: %v", err)
	}

	pred.State = domain.StateEnabled
	pred.UpdatedAt = now.Add(time.Minute)
	if err := s.UpdatePrediction(ctx, pred); err != nil {
		t.Fatalf("update: %v", err)
	}

	got, err := s.GetPrediction(ctx, "pred-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.State != domain.StateEnabled {
		t.Fatalf("expected enabled, got %s", got.State)
	}
}

func TestListPredictionsFilter(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	for i, st := range []domain.PredictionState{domain.StateDraft, domain.StateEnabled, domain.StateMonitoring} {
		pred := domain.Prediction{
			ID: fmt.Sprintf("pred-%d", i), Statement: "test", Asset: "BINANCE:BTC/USDT",
			Rule: domain.Rule{Type: domain.RuleThreshold, PriceField: "close", Deadline: now},
			StartTime: now, Deadline: now, State: st,
			CreatedAt: now.Add(time.Duration(i) * time.Second), UpdatedAt: now,
		}
		if err := s.InsertPrediction(ctx, pred); err != nil {
			t.Fatalf("insert %d: %v", i, err)
		}
	}

	results, err := s.ListPredictions(ctx, storage.PredictionFilter{States: []domain.PredictionState{domain.StateDraft, domain.StateEnabled}})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestEventAppendAndList(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	pred := domain.Prediction{
		ID: "pred-1", Statement: "test", Asset: "X",
		Rule: domain.Rule{Type: domain.RuleThreshold, PriceField: "close", Deadline: now},
		StartTime: now, Deadline: now, State: domain.StateDraft,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := s.InsertPrediction(ctx, pred); err != nil {
		t.Fatal(err)
	}

	e1 := domain.Event{ID: "e1", PredictionID: "pred-1", Type: domain.EventPredictionIngested, Timestamp: now}
	e2 := domain.Event{ID: "e2", PredictionID: "pred-1", Type: domain.EventPredictionEnabled, Timestamp: now.Add(time.Second)}
	for _, e := range []domain.Event{e1, e2} {
		if err := s.InsertEvent(ctx, e); err != nil {
			t.Fatalf("insert event: %v", err)
		}
	}

	events, err := s.ListEvents(ctx, "pred-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].Type != domain.EventPredictionIngested {
		t.Fatalf("expected first event to be ingested, got %s", events[0].Type)
	}

	all, err := s.ListAllEvents(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 total events, got %d", len(all))
	}
}

func TestCandleValuesInsertAndGet(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	pred := domain.Prediction{
		ID: "pred-1", Statement: "test", Asset: "X",
		Rule: domain.Rule{Type: domain.RuleThreshold, PriceField: "close", Deadline: now},
		StartTime: now, Deadline: now, State: domain.StateDraft,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := s.InsertPrediction(ctx, pred); err != nil {
		t.Fatal(err)
	}

	vals := []domain.CandleValue{
		{Timestamp: 1000, Open: 1, High: 2, Low: 0.5, Close: 1.5, Source: "BINANCE"},
		{Timestamp: 1060, Open: 1.5, High: 3, Low: 1, Close: 2.5, Source: "BINANCE"},
		{Timestamp: 1120, Open: 2.5, High: 4, Low: 2, Close: 3.5, Source: "BINANCE"},
	}
	if err := s.InsertValues(ctx, "pred-1", vals); err != nil {
		t.Fatal(err)
	}

	got, err := s.GetValues(ctx, "pred-1", time.Unix(1000, 0), time.Unix(1060, 0))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 values, got %d", len(got))
	}

	lastTs, err := s.GetLastValueTimestamp(ctx, "pred-1")
	if err != nil {
		t.Fatal(err)
	}
	if lastTs != 1120 {
		t.Fatalf("expected last timestamp 1120, got %d", lastTs)
	}
}

func TestDuplicateValueInsertIgnored(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	pred := domain.Prediction{
		ID: "pred-1", Statement: "test", Asset: "X",
		Rule: domain.Rule{Type: domain.RuleThreshold, PriceField: "close", Deadline: now},
		StartTime: now, Deadline: now, State: domain.StateDraft,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := s.InsertPrediction(ctx, pred); err != nil {
		t.Fatal(err)
	}

	val := domain.CandleValue{Timestamp: 1000, Open: 1, High: 2, Low: 0.5, Close: 1.5, Source: "BINANCE"}
	if err := s.InsertValues(ctx, "pred-1", []domain.CandleValue{val}); err != nil {
		t.Fatal(err)
	}
	if err := s.InsertValues(ctx, "pred-1", []domain.CandleValue{val}); err != nil {
		t.Fatal(err)
	}

	got, err := s.GetValues(ctx, "pred-1", time.Unix(0, 0), time.Unix(9999, 0))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 value after duplicate insert, got %d", len(got))
	}
}
