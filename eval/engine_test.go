package eval

import (
	"testing"
	"time"

	"github.com/marianogappa/predictions-tracker/domain"
)

func ts(sec int64) time.Time { return time.Unix(sec, 0).UTC() }

func TestThresholdPassBeforeDeadline(t *testing.T) {
	e := NewEngine()
	rule := domain.Rule{Type: domain.RuleThreshold, Operator: ">=", PriceField: "close", Value: 100, Deadline: ts(2000)}
	values := []domain.CandleValue{
		{Timestamp: 1000, Close: 90},
		{Timestamp: 1060, Close: 100},
	}
	r := e.Evaluate(rule, values, ts(1060))
	if !r.Decided || !r.Correct {
		t.Fatalf("expected decided+correct, got %+v", r)
	}
}

func TestThresholdFailAtDeadline(t *testing.T) {
	e := NewEngine()
	rule := domain.Rule{Type: domain.RuleThreshold, Operator: ">=", PriceField: "close", Value: 100, Deadline: ts(2000)}
	values := []domain.CandleValue{
		{Timestamp: 1000, Close: 90},
		{Timestamp: 1060, Close: 95},
	}
	r := e.Evaluate(rule, values, ts(2000))
	if !r.Decided || r.Correct {
		t.Fatalf("expected decided+incorrect, got %+v", r)
	}
}

func TestThresholdUndecidedBeforeDeadline(t *testing.T) {
	e := NewEngine()
	rule := domain.Rule{Type: domain.RuleThreshold, Operator: ">=", PriceField: "close", Value: 100, Deadline: ts(2000)}
	values := []domain.CandleValue{
		{Timestamp: 1000, Close: 90},
	}
	r := e.Evaluate(rule, values, ts(1500))
	if r.Decided {
		t.Fatalf("expected undecided, got %+v", r)
	}
}

func TestThresholdOperators(t *testing.T) {
	e := NewEngine()
	cases := []struct {
		op      string
		price   float64
		target  float64
		correct bool
	}{
		{">=", 100, 100, true},
		{">", 100, 100, false},
		{">", 101, 100, true},
		{"<=", 100, 100, true},
		{"<", 100, 100, false},
		{"<", 99, 100, true},
	}
	for _, tc := range cases {
		rule := domain.Rule{Type: domain.RuleThreshold, Operator: tc.op, PriceField: "close", Value: tc.target, Deadline: ts(2000)}
		values := []domain.CandleValue{{Timestamp: 1000, Close: tc.price}}
		r := e.Evaluate(rule, values, ts(1000))
		if r.Correct != tc.correct {
			t.Errorf("op=%s price=%.0f target=%.0f: expected correct=%v, got %v", tc.op, tc.price, tc.target, tc.correct, r.Correct)
		}
	}
}

func TestThresholdUsesCorrectPriceField(t *testing.T) {
	e := NewEngine()
	rule := domain.Rule{Type: domain.RuleThreshold, Operator: ">=", PriceField: "high", Value: 200, Deadline: ts(2000)}
	values := []domain.CandleValue{
		{Timestamp: 1000, Open: 150, High: 210, Low: 140, Close: 160},
	}
	r := e.Evaluate(rule, values, ts(1000))
	if !r.Decided || !r.Correct {
		t.Fatalf("expected correct via high price, got %+v", r)
	}
}

func TestCrossingDetected(t *testing.T) {
	e := NewEngine()
	rule := domain.Rule{Type: domain.RuleCrossing, Operator: ">=", PriceField: "close", Value: 100, Deadline: ts(2000)}
	values := []domain.CandleValue{
		{Timestamp: 1000, Close: 90},
		{Timestamp: 1060, Close: 105},
	}
	r := e.Evaluate(rule, values, ts(1060))
	if !r.Decided || !r.Correct {
		t.Fatalf("expected crossing detected, got %+v", r)
	}
}

func TestCrossingNotDetected(t *testing.T) {
	e := NewEngine()
	rule := domain.Rule{Type: domain.RuleCrossing, Operator: ">=", PriceField: "close", Value: 100, Deadline: ts(2000)}
	values := []domain.CandleValue{
		{Timestamp: 1000, Close: 110},
		{Timestamp: 1060, Close: 120},
	}
	r := e.Evaluate(rule, values, ts(2000))
	if !r.Decided || r.Correct {
		t.Fatalf("expected no crossing (already above), got %+v", r)
	}
}

func TestCrossingDownward(t *testing.T) {
	e := NewEngine()
	rule := domain.Rule{Type: domain.RuleCrossing, Operator: "<=", PriceField: "close", Value: 100, Deadline: ts(2000)}
	values := []domain.CandleValue{
		{Timestamp: 1000, Close: 110},
		{Timestamp: 1060, Close: 95},
	}
	r := e.Evaluate(rule, values, ts(1060))
	if !r.Decided || !r.Correct {
		t.Fatalf("expected downward crossing, got %+v", r)
	}
}

func TestSustainedDurationCorrect(t *testing.T) {
	e := NewEngine()
	dur := int64(120000) // 120s in ms
	rule := domain.Rule{Type: domain.RuleSustainedDuration, Operator: ">=", PriceField: "close", Value: 100, DurationMs: &dur, Deadline: ts(2000)}
	values := []domain.CandleValue{
		{Timestamp: 1000, Close: 105},
		{Timestamp: 1060, Close: 110},
		{Timestamp: 1120, Close: 115},
	}
	r := e.Evaluate(rule, values, ts(1120))
	if !r.Decided || !r.Correct {
		t.Fatalf("expected sustained correct, got %+v", r)
	}
}

func TestSustainedDurationBroken(t *testing.T) {
	e := NewEngine()
	dur := int64(180000) // 180s
	rule := domain.Rule{Type: domain.RuleSustainedDuration, Operator: ">=", PriceField: "close", Value: 100, DurationMs: &dur, Deadline: ts(2000)}
	values := []domain.CandleValue{
		{Timestamp: 1000, Close: 105},
		{Timestamp: 1060, Close: 90}, // breaks
		{Timestamp: 1120, Close: 110},
		{Timestamp: 1180, Close: 115},
	}
	r := e.Evaluate(rule, values, ts(2000))
	if !r.Decided || r.Correct {
		t.Fatalf("expected sustained broken+incorrect at deadline, got %+v", r)
	}
}

func TestSustainedDurationUndecided(t *testing.T) {
	e := NewEngine()
	dur := int64(300000) // 300s
	rule := domain.Rule{Type: domain.RuleSustainedDuration, Operator: ">=", PriceField: "close", Value: 100, DurationMs: &dur, Deadline: ts(5000)}
	values := []domain.CandleValue{
		{Timestamp: 1000, Close: 105},
		{Timestamp: 1060, Close: 110},
	}
	r := e.Evaluate(rule, values, ts(1500))
	if r.Decided {
		t.Fatalf("expected undecided, got %+v", r)
	}
}

func TestDeterminism(t *testing.T) {
	e := NewEngine()
	rule := domain.Rule{Type: domain.RuleThreshold, Operator: ">=", PriceField: "close", Value: 100, Deadline: ts(2000)}
	values := []domain.CandleValue{
		{Timestamp: 1000, Close: 90},
		{Timestamp: 1060, Close: 105},
	}
	now := ts(1060)

	r1 := e.Evaluate(rule, values, now)
	r2 := e.Evaluate(rule, values, now)
	if r1 != r2 {
		t.Fatalf("non-deterministic: %+v vs %+v", r1, r2)
	}
}
