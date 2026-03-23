package eval

import (
	"fmt"
	"time"

	"github.com/marianogappa/predictions-tracker/domain"
)

func evaluateSustained(rule domain.Rule, values []domain.CandleValue, now time.Time) Result {
	deadline := rule.Deadline.Unix()
	requiredDuration := rule.DurationAsDuration()

	if requiredDuration <= 0 {
		return Result{Decided: true, Correct: false, Reason: "sustained_duration rule requires a positive duration", Timestamp: now}
	}

	var streakStart int64

	for _, v := range values {
		if v.Timestamp > deadline {
			break
		}
		price := v.PriceByField(rule.PriceField)
		if compareOp(price, rule.Operator, rule.Value) {
			if streakStart == 0 {
				streakStart = v.Timestamp
			}
			elapsed := time.Duration(v.Timestamp-streakStart) * time.Second
			if elapsed >= requiredDuration {
				return Result{
					Decided:   true,
					Correct:   true,
					Reason:    fmt.Sprintf("condition sustained for %v starting at %d", requiredDuration, streakStart),
					Timestamp: time.Unix(v.Timestamp, 0).UTC(),
				}
			}
		} else {
			streakStart = 0
		}
	}

	if now.Unix() >= deadline {
		return Result{Decided: true, Correct: false, Reason: "deadline passed without sustained condition", Timestamp: now}
	}
	return Result{Decided: false}
}
