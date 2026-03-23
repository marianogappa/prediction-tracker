package eval

import (
	"fmt"
	"time"

	"github.com/marianogappa/predictions-tracker/domain"
)

func evaluateThreshold(rule domain.Rule, values []domain.CandleValue, now time.Time) Result {
	deadline := rule.Deadline.Unix()

	for _, v := range values {
		if v.Timestamp > deadline {
			break
		}
		price := v.PriceByField(rule.PriceField)
		if compareOp(price, rule.Operator, rule.Value) {
			return Result{
				Decided:   true,
				Correct:   true,
				Reason:    fmt.Sprintf("%s %s %.2f satisfied at timestamp %d (price=%.2f)", rule.PriceField, rule.Operator, rule.Value, v.Timestamp, price),
				Timestamp: time.Unix(v.Timestamp, 0).UTC(),
			}
		}
	}

	if now.Unix() >= deadline {
		return Result{
			Decided:   true,
			Correct:   false,
			Reason:    fmt.Sprintf("deadline passed without %s %s %.2f", rule.PriceField, rule.Operator, rule.Value),
			Timestamp: now,
		}
	}

	return Result{Decided: false}
}

func compareOp(price float64, op string, target float64) bool {
	switch op {
	case ">=":
		return price >= target
	case ">":
		return price > target
	case "<=":
		return price <= target
	case "<":
		return price < target
	default:
		return false
	}
}
