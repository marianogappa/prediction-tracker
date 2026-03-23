package eval

import (
	"fmt"
	"time"

	"github.com/marianogappa/predictions-tracker/domain"
)

func evaluateCrossing(rule domain.Rule, values []domain.CandleValue, now time.Time) Result {
	deadline := rule.Deadline.Unix()

	if len(values) < 2 {
		if now.Unix() >= deadline {
			return Result{Decided: true, Correct: false, Reason: "deadline passed with insufficient data for crossing detection", Timestamp: now}
		}
		return Result{Decided: false}
	}

	for i := 1; i < len(values); i++ {
		if values[i].Timestamp > deadline {
			break
		}
		prev := values[i-1].PriceByField(rule.PriceField)
		curr := values[i].PriceByField(rule.PriceField)

		crossed := false
		switch rule.Operator {
		case ">=", ">":
			crossed = prev < rule.Value && curr >= rule.Value
		case "<=", "<":
			crossed = prev > rule.Value && curr <= rule.Value
		}

		if crossed {
			return Result{
				Decided:   true,
				Correct:   true,
				Reason:    fmt.Sprintf("crossing detected at timestamp %d: %s went from %.2f to %.2f (target %.2f)", values[i].Timestamp, rule.PriceField, prev, curr, rule.Value),
				Timestamp: time.Unix(values[i].Timestamp, 0).UTC(),
			}
		}
	}

	if now.Unix() >= deadline {
		return Result{Decided: true, Correct: false, Reason: "deadline passed without crossing", Timestamp: now}
	}
	return Result{Decided: false}
}
