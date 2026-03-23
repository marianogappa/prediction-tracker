package domain

import "time"

type RuleType string

const (
	RuleThreshold         RuleType = "threshold"
	RuleCrossing          RuleType = "crossing"
	RuleSustainedDuration RuleType = "sustained_duration"
)

type Rule struct {
	Type       RuleType  `json:"type"`
	Operator   string    `json:"operator,omitempty"`
	PriceField string    `json:"price_field"`
	Value      float64   `json:"value,omitempty"`
	DurationMs *int64    `json:"duration_ms,omitempty"`
	Deadline   time.Time `json:"deadline"`
}

func (r Rule) DurationAsDuration() time.Duration {
	if r.DurationMs == nil {
		return 0
	}
	return time.Duration(*r.DurationMs) * time.Millisecond
}
