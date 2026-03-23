package storage

import (
	"context"
	"time"

	"github.com/marianogappa/predictions-tracker/domain"
)

type PredictionFilter struct {
	States []domain.PredictionState
	Asset  string
	Limit  int
	Offset int
}

type Storage interface {
	InsertPrediction(ctx context.Context, p domain.Prediction) error
	UpdatePrediction(ctx context.Context, p domain.Prediction) error
	GetPrediction(ctx context.Context, id string) (domain.Prediction, error)
	ListPredictions(ctx context.Context, filter PredictionFilter) ([]domain.Prediction, error)

	InsertEvent(ctx context.Context, e domain.Event) error
	ListEvents(ctx context.Context, predictionID string) ([]domain.Event, error)
	ListAllEvents(ctx context.Context) ([]domain.Event, error)

	InsertValues(ctx context.Context, predictionID string, vals []domain.CandleValue) error
	GetValues(ctx context.Context, predictionID string, from, to time.Time) ([]domain.CandleValue, error)
	GetLastValueTimestamp(ctx context.Context, predictionID string) (int64, error)

	Close() error
}
