package domain

import "time"

type PredictionState string

const (
	StateDraft           PredictionState = "draft"
	StateEnabled         PredictionState = "enabled"
	StateMonitoring      PredictionState = "monitoring"
	StateDisabled        PredictionState = "disabled"
	StateErrored         PredictionState = "errored"
	StateFinalCorrect    PredictionState = "final_correct"
	StateFinalIncorrect  PredictionState = "final_incorrect"
	StateFinalUnresolved PredictionState = "final_unresolved"
)

func (s PredictionState) IsFinal() bool {
	switch s {
	case StateFinalCorrect, StateFinalIncorrect, StateFinalUnresolved:
		return true
	default:
		return false
	}
}

type Prediction struct {
	ID         string          `json:"id"`
	Statement  string          `json:"statement"`
	Rule       Rule            `json:"rule"`
	Asset      string          `json:"asset"`
	StartTime  time.Time       `json:"start_time"`
	Deadline   time.Time       `json:"deadline"`
	SourceURL  string          `json:"source_url,omitempty"`
	AuthorName string          `json:"author_name,omitempty"`
	AuthorURL  string          `json:"author_url,omitempty"`
	State      PredictionState `json:"state"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}
