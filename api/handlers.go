package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/marianogappa/predictions-tracker/domain"
	exportpkg "github.com/marianogappa/predictions-tracker/export"
	"github.com/marianogappa/predictions-tracker/statemachine"
	"github.com/marianogappa/predictions-tracker/storage"
)

type createPredictionRequest struct {
	Statement  string      `json:"statement"`
	Rule       domain.Rule `json:"rule"`
	Asset      string      `json:"asset"`
	StartTime  *time.Time  `json:"start_time,omitempty"`
	Deadline   time.Time   `json:"deadline"`
	SourceURL  string      `json:"source_url,omitempty"`
	AuthorName string      `json:"author_name,omitempty"`
	AuthorURL  string      `json:"author_url,omitempty"`
}

func (s *Server) handleCreatePrediction(w http.ResponseWriter, r *http.Request) {
	var req createPredictionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	now := time.Now().UTC()
	startTime := now
	if req.StartTime != nil {
		startTime = *req.StartTime
	}

	pred := domain.Prediction{
		ID:         uuid.New().String(),
		Statement:  req.Statement,
		Rule:       req.Rule,
		Asset:      req.Asset,
		StartTime:  startTime,
		Deadline:   req.Deadline,
		SourceURL:  req.SourceURL,
		AuthorName: req.AuthorName,
		AuthorURL:  req.AuthorURL,
		State:      domain.StateDraft,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := s.store.InsertPrediction(r.Context(), pred); err != nil {
		http.Error(w, "failed to insert prediction", http.StatusInternalServerError)
		return
	}

	ingestEvt := domain.Event{
		ID:           uuid.New().String(),
		PredictionID: pred.ID,
		Type:         domain.EventPredictionIngested,
		Timestamp:    now,
	}
	_ = s.store.InsertEvent(r.Context(), ingestEvt)
	s.bus.Publish(ingestEvt)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(pred)
}

func (s *Server) handleListPredictions(w http.ResponseWriter, r *http.Request) {
	filter := storage.PredictionFilter{}
	if states := r.URL.Query()["state"]; len(states) > 0 {
		for _, st := range states {
			filter.States = append(filter.States, domain.PredictionState(st))
		}
	}

	preds, err := s.store.ListPredictions(r.Context(), filter)
	if err != nil {
		http.Error(w, "failed to list predictions", http.StatusInternalServerError)
		return
	}
	if preds == nil {
		preds = []domain.Prediction{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(preds)
}

func (s *Server) handleGetPrediction(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	pred, err := s.store.GetPrediction(r.Context(), id)
	if err != nil {
		http.Error(w, "prediction not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pred)
}

func (s *Server) handleEnablePrediction(w http.ResponseWriter, r *http.Request) {
	s.transitionPrediction(w, r, domain.StateEnabled)
}

func (s *Server) handleDisablePrediction(w http.ResponseWriter, r *http.Request) {
	s.transitionPrediction(w, r, domain.StateDisabled)
}

func (s *Server) transitionPrediction(w http.ResponseWriter, r *http.Request, target domain.PredictionState) {
	id := chi.URLParam(r, "id")
	pred, err := s.store.GetPrediction(r.Context(), id)
	if err != nil {
		http.Error(w, "prediction not found", http.StatusNotFound)
		return
	}

	now := time.Now().UTC()
	evt, err := statemachine.Transition(&pred, target, now, "manual via API")
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	if err := s.store.UpdatePrediction(r.Context(), pred); err != nil {
		http.Error(w, "failed to update prediction", http.StatusInternalServerError)
		return
	}
	if err := s.store.InsertEvent(r.Context(), evt); err != nil {
		http.Error(w, "failed to insert event", http.StatusInternalServerError)
		return
	}
	s.bus.Publish(evt)

	if r.Header.Get("Accept") == "" || r.Header.Get("Referer") != "" {
		http.Redirect(w, r, "/predictions/"+id, http.StatusSeeOther)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pred)
}

func (s *Server) handleListEvents(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	events, err := s.store.ListEvents(r.Context(), id)
	if err != nil {
		http.Error(w, "failed to list events", http.StatusInternalServerError)
		return
	}
	if events == nil {
		events = []domain.Event{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}

func (s *Server) handleListValues(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	pred, err := s.store.GetPrediction(r.Context(), id)
	if err != nil {
		http.Error(w, "prediction not found", http.StatusNotFound)
		return
	}
	values, err := s.store.GetValues(r.Context(), pred.ID, pred.StartTime, pred.Deadline)
	if err != nil {
		http.Error(w, "failed to get values", http.StatusInternalServerError)
		return
	}
	if values == nil {
		values = []domain.CandleValue{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(values)
}

func (s *Server) handleExportPrediction(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	pred, err := s.store.GetPrediction(r.Context(), id)
	if err != nil {
		http.Error(w, "prediction not found", http.StatusNotFound)
		return
	}

	events, err := s.store.ListEvents(r.Context(), id)
	if err != nil {
		http.Error(w, "failed to list events", http.StatusInternalServerError)
		return
	}

	values, err := s.store.GetValues(r.Context(), id, pred.StartTime, pred.Deadline)
	if err != nil {
		http.Error(w, "failed to get values", http.StatusInternalServerError)
		return
	}

	var chartPNG []byte
	if len(values) > 0 {
		chartPNG, _ = exportpkg.RenderChart(values, pred.Rule)
	}

	md, err := exportpkg.RenderMarkdown(exportpkg.ExportData{
		Prediction: pred,
		Events:     events,
		Values:     values,
		ChartPNG:   chartPNG,
	})
	if err != nil {
		http.Error(w, "failed to render export", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\"prediction-"+id+".md\"")
	w.Write([]byte(md))
}
