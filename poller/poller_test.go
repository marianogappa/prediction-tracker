package poller

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/marianogappa/predictions-tracker/domain"
	"github.com/marianogappa/predictions-tracker/eval"
	"github.com/marianogappa/predictions-tracker/event"
	"github.com/marianogappa/predictions-tracker/storage/sqlite"
)

type mockSource struct {
	candles []domain.CandleValue
	err     error
}

func (m *mockSource) FetchCandles(_ context.Context, _ string, _ time.Time, _ time.Duration) ([]domain.CandleValue, error) {
	return m.candles, m.err
}

func TestPollerPromotesEnabledToMonitoring(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := sqlite.New(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	pred := domain.Prediction{
		ID: "p1", Statement: "test", Asset: "BINANCE:BTC/USDT",
		Rule:      domain.Rule{Type: domain.RuleThreshold, Operator: ">=", PriceField: "close", Value: 100000, Deadline: now.Add(24 * time.Hour)},
		StartTime: now, Deadline: now.Add(24 * time.Hour),
		State: domain.StateEnabled, CreatedAt: now, UpdatedAt: now,
	}
	if err := store.InsertPrediction(ctx, pred); err != nil {
		t.Fatal(err)
	}

	src := &mockSource{}
	bus := event.NewBus()
	engine := eval.NewEngine()
	p := New(store, src, engine, bus, time.Hour)

	p.promoteEnabled(ctx)

	got, err := store.GetPrediction(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	if got.State != domain.StateMonitoring {
		t.Fatalf("expected monitoring, got %s", got.State)
	}
}

func TestPollerFinalizesCorrectPrediction(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := sqlite.New(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	pred := domain.Prediction{
		ID: "p1", Statement: "BTC above 50k", Asset: "BINANCE:BTC/USDT",
		Rule:      domain.Rule{Type: domain.RuleThreshold, Operator: ">=", PriceField: "close", Value: 50000, Deadline: now.Add(24 * time.Hour)},
		StartTime: now.Add(-time.Hour), Deadline: now.Add(24 * time.Hour),
		State: domain.StateMonitoring, CreatedAt: now, UpdatedAt: now,
	}
	if err := store.InsertPrediction(ctx, pred); err != nil {
		t.Fatal(err)
	}

	src := &mockSource{
		candles: []domain.CandleValue{
			{Timestamp: now.Unix(), Open: 49000, High: 51000, Low: 48000, Close: 50500, Source: "BINANCE"},
		},
	}

	var published []domain.Event
	bus := event.NewBus()
	bus.Subscribe(func(e domain.Event) { published = append(published, e) })

	engine := eval.NewEngine()
	p := New(store, src, engine, bus, time.Hour)
	p.pollMonitoring(ctx)

	got, err := store.GetPrediction(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	if got.State != domain.StateFinalCorrect {
		t.Fatalf("expected final_correct, got %s", got.State)
	}
	if len(published) == 0 {
		t.Fatal("expected events to be published")
	}
}
