package api

import (
	"embed"
	"html/template"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/marianogappa/predictions-tracker/eval"
	"github.com/marianogappa/predictions-tracker/event"
	"github.com/marianogappa/predictions-tracker/storage"
)

//go:embed templates/*.html
var templateFS embed.FS

type Server struct {
	store  storage.Storage
	bus    *event.Bus
	engine *eval.Engine
	tmpl   *template.Template
	router chi.Router
}

func NewServer(store storage.Storage, bus *event.Bus, engine *eval.Engine) (*Server, error) {
	funcMap := template.FuncMap{
		"stateClass": stateClass,
	}
	tmpl, err := template.New("").Funcs(funcMap).ParseFS(templateFS, "templates/*.html")
	if err != nil {
		return nil, err
	}
	s := &Server{
		store:  store,
		bus:    bus,
		engine: engine,
		tmpl:   tmpl,
	}
	s.router = s.routes()
	return s, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *Server) routes() chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/api/predictions", s.handleListPredictions)
	r.Get("/api/predictions/{id}", s.handleGetPrediction)
	r.Post("/api/predictions", s.handleCreatePrediction)
	r.Post("/api/predictions/{id}/enable", s.handleEnablePrediction)
	r.Post("/api/predictions/{id}/disable", s.handleDisablePrediction)
	r.Get("/api/predictions/{id}/events", s.handleListEvents)
	r.Get("/api/predictions/{id}/values", s.handleListValues)
	r.Get("/api/predictions/{id}/export", s.handleExportPrediction)

	r.Get("/", s.handleUIList)
	r.Get("/predictions/{id}", s.handleUIDetail)

	return r
}

func stateClass(state string) string {
	switch state {
	case "draft":
		return "badge-draft"
	case "enabled":
		return "badge-enabled"
	case "monitoring":
		return "badge-monitoring"
	case "disabled":
		return "badge-disabled"
	case "errored":
		return "badge-errored"
	case "final_correct":
		return "badge-correct"
	case "final_incorrect":
		return "badge-incorrect"
	case "final_unresolved":
		return "badge-unresolved"
	default:
		return ""
	}
}
