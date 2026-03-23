package eval

import (
	"time"

	"github.com/marianogappa/predictions-tracker/domain"
)

type Result struct {
	Decided   bool
	Correct   bool
	Reason    string
	Timestamp time.Time
}

type Evaluator interface {
	Evaluate(rule domain.Rule, values []domain.CandleValue, now time.Time) Result
}

func NewEngine() *Engine {
	return &Engine{}
}

type Engine struct{}

func (e *Engine) Evaluate(rule domain.Rule, values []domain.CandleValue, now time.Time) Result {
	switch rule.Type {
	case domain.RuleThreshold:
		return evaluateThreshold(rule, values, now)
	case domain.RuleCrossing:
		return evaluateCrossing(rule, values, now)
	case domain.RuleSustainedDuration:
		return evaluateSustained(rule, values, now)
	default:
		return Result{Decided: true, Correct: false, Reason: "unknown rule type", Timestamp: now}
	}
}
