package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/marianogappa/predictions-tracker/domain"
	"github.com/marianogappa/predictions-tracker/storage"
)

type listPageData struct {
	Predictions []domain.Prediction
	StateFilter string
}

type detailPageData struct {
	Prediction domain.Prediction
	Events     []domain.Event
	Values     []domain.CandleValue
}

func (s *Server) handleUIList(w http.ResponseWriter, r *http.Request) {
	filter := storage.PredictionFilter{}
	stateFilter := r.URL.Query().Get("state")
	if stateFilter != "" {
		filter.States = []domain.PredictionState{domain.PredictionState(stateFilter)}
	}

	preds, err := s.store.ListPredictions(r.Context(), filter)
	if err != nil {
		http.Error(w, "failed to list predictions", http.StatusInternalServerError)
		return
	}
	if preds == nil {
		preds = []domain.Prediction{}
	}

	data := listPageData{Predictions: preds, StateFilter: stateFilter}
	if err := s.tmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleUIDetail(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	pred, err := s.store.GetPrediction(r.Context(), id)
	if err != nil {
		http.Error(w, "prediction not found", http.StatusNotFound)
		return
	}

	events, err := s.store.ListEvents(r.Context(), id)
	if err != nil {
		events = []domain.Event{}
	}

	values, err := s.store.GetValues(r.Context(), id, pred.StartTime, pred.Deadline)
	if err != nil {
		values = []domain.CandleValue{}
	}

	data := detailPageData{Prediction: pred, Events: events, Values: values}
	if err := s.tmpl.ExecuteTemplate(w, "detail.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
