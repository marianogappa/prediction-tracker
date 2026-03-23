package source

import (
	"context"
	"time"

	"github.com/marianogappa/predictions-tracker/domain"
)

type SourceOfTruth interface {
	FetchCandles(ctx context.Context, asset string, from time.Time, interval time.Duration) ([]domain.CandleValue, error)
}
