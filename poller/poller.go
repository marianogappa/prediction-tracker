package poller

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/marianogappa/predictions-tracker/domain"
	"github.com/marianogappa/predictions-tracker/eval"
	"github.com/marianogappa/predictions-tracker/event"
	"github.com/marianogappa/predictions-tracker/source"
	"github.com/marianogappa/predictions-tracker/statemachine"
	"github.com/marianogappa/predictions-tracker/storage"
)

type Poller struct {
	store    storage.Storage
	source   source.SourceOfTruth
	engine   *eval.Engine
	bus      *event.Bus
	interval time.Duration
}

func New(store storage.Storage, src source.SourceOfTruth, engine *eval.Engine, bus *event.Bus, interval time.Duration) *Poller {
	return &Poller{
		store:    store,
		source:   src,
		engine:   engine,
		bus:      bus,
		interval: interval,
	}
}

func (p *Poller) Run(ctx context.Context) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	p.tick(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.tick(ctx)
		}
	}
}

func (p *Poller) tick(ctx context.Context) {
	p.promoteEnabled(ctx)
	p.pollMonitoring(ctx)
}

func (p *Poller) promoteEnabled(ctx context.Context) {
	preds, err := p.store.ListPredictions(ctx, storage.PredictionFilter{
		States: []domain.PredictionState{domain.StateEnabled},
	})
	if err != nil {
		log.Printf("poller: listing enabled predictions: %v", err)
		return
	}
	now := time.Now().UTC()
	for _, pred := range preds {
		evt, err := statemachine.Transition(&pred, domain.StateMonitoring, now, "auto-promote by poller")
		if err != nil {
			log.Printf("poller: promoting %s to monitoring: %v", pred.ID, err)
			continue
		}
		if err := p.store.UpdatePrediction(ctx, pred); err != nil {
			log.Printf("poller: updating prediction %s: %v", pred.ID, err)
			continue
		}
		if err := p.store.InsertEvent(ctx, evt); err != nil {
			log.Printf("poller: inserting event for %s: %v", pred.ID, err)
			continue
		}
		p.bus.Publish(evt)
	}
}

func (p *Poller) pollMonitoring(ctx context.Context) {
	preds, err := p.store.ListPredictions(ctx, storage.PredictionFilter{
		States: []domain.PredictionState{domain.StateMonitoring},
	})
	if err != nil {
		log.Printf("poller: listing monitoring predictions: %v", err)
		return
	}

	now := time.Now().UTC()
	for _, pred := range preds {
		if err := p.processPrediction(ctx, &pred, now); err != nil {
			log.Printf("poller: processing %s: %v", pred.ID, err)
			p.transitionToErrored(ctx, &pred, now, err.Error())
		}
	}
}

func (p *Poller) processPrediction(ctx context.Context, pred *domain.Prediction, now time.Time) error {
	lastTs, err := p.store.GetLastValueTimestamp(ctx, pred.ID)
	if err != nil {
		return err
	}

	from := pred.StartTime
	if lastTs > 0 {
		from = time.Unix(lastTs+1, 0).UTC()
	}

	candles, err := p.source.FetchCandles(ctx, pred.Asset, from, time.Minute)
	if err != nil {
		return err
	}

	if len(candles) > 0 {
		if err := p.store.InsertValues(ctx, pred.ID, candles); err != nil {
			return err
		}
		p.emitValueFetchedEvent(ctx, pred.ID, len(candles), now)
	}

	allValues, err := p.store.GetValues(ctx, pred.ID, pred.StartTime, pred.Deadline)
	if err != nil {
		return err
	}

	result := p.engine.Evaluate(pred.Rule, allValues, now)
	p.emitEvaluationEvent(ctx, pred.ID, result, now)

	if !result.Decided {
		return nil
	}

	var targetState domain.PredictionState
	if result.Correct {
		targetState = domain.StateFinalCorrect
	} else {
		targetState = domain.StateFinalIncorrect
	}

	if now.After(pred.Deadline) && !result.Correct {
		targetState = domain.StateFinalUnresolved
	}

	evt, err := statemachine.Transition(pred, targetState, now, result.Reason)
	if err != nil {
		return err
	}
	if err := p.store.UpdatePrediction(ctx, *pred); err != nil {
		return err
	}
	if err := p.store.InsertEvent(ctx, evt); err != nil {
		return err
	}
	p.bus.Publish(evt)
	return nil
}

func (p *Poller) transitionToErrored(ctx context.Context, pred *domain.Prediction, now time.Time, reason string) {
	evt, err := statemachine.Transition(pred, domain.StateErrored, now, reason)
	if err != nil {
		log.Printf("poller: transitioning %s to errored: %v", pred.ID, err)
		return
	}
	if err := p.store.UpdatePrediction(ctx, *pred); err != nil {
		log.Printf("poller: updating errored prediction %s: %v", pred.ID, err)
		return
	}
	if err := p.store.InsertEvent(ctx, evt); err != nil {
		log.Printf("poller: inserting errored event for %s: %v", pred.ID, err)
		return
	}
	p.bus.Publish(evt)
}

func (p *Poller) emitValueFetchedEvent(ctx context.Context, predictionID string, count int, now time.Time) {
	payload, _ := json.Marshal(map[string]int{"candles_fetched": count})
	evt := domain.Event{
		ID:           uuid.New().String(),
		PredictionID: predictionID,
		Type:         domain.EventValueFetched,
		Timestamp:    now,
		Payload:      payload,
	}
	_ = p.store.InsertEvent(ctx, evt)
	p.bus.Publish(evt)
}

func (p *Poller) emitEvaluationEvent(ctx context.Context, predictionID string, result eval.Result, now time.Time) {
	payload, _ := json.Marshal(map[string]any{
		"decided": result.Decided,
		"correct": result.Correct,
		"reason":  result.Reason,
	})
	evt := domain.Event{
		ID:           uuid.New().String(),
		PredictionID: predictionID,
		Type:         domain.EventEvaluationPerformed,
		Timestamp:    now,
		Payload:      payload,
	}
	_ = p.store.InsertEvent(ctx, evt)
	p.bus.Publish(evt)
}
