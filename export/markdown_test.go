package export

import (
	"strings"
	"testing"
	"time"

	"github.com/marianogappa/predictions-tracker/domain"
)

func TestRenderMarkdownContainsEssentialSections(t *testing.T) {
	now := time.Now().UTC()
	data := ExportData{
		Prediction: domain.Prediction{
			ID:         "p1",
			Statement:  "BTC above 100k",
			Asset:      "BINANCE:BTC/USDT",
			State:      domain.StateFinalCorrect,
			Rule:       domain.Rule{Type: domain.RuleThreshold, Operator: ">=", PriceField: "close", Value: 100000, Deadline: now},
			AuthorName: "Alice",
			SourceURL:  "https://example.com",
			CreatedAt:  now.Add(-time.Hour),
			UpdatedAt:  now,
			Deadline:   now,
		},
		Events: []domain.Event{
			{ID: "e1", PredictionID: "p1", Type: domain.EventPredictionIngested, Timestamp: now.Add(-time.Hour)},
			{ID: "e2", PredictionID: "p1", Type: domain.EventPredictionCorrect, Timestamp: now},
		},
		Values: []domain.CandleValue{
			{Timestamp: now.Unix() - 60, Open: 99000, High: 101000, Low: 98000, Close: 100500},
		},
	}

	md, err := RenderMarkdown(data)
	if err != nil {
		t.Fatal(err)
	}

	required := []string{
		"# Prediction: BTC above 100k",
		"**Asset:** BINANCE:BTC/USDT",
		"**State:** final_correct",
		"**Author:** Alice",
		"**Source:** https://example.com",
		"## Rule",
		"\"threshold\"",
		"## Evaluation Result",
		"## Event Log",
		"prediction_ingested",
		"prediction_correct",
		"## Price Data",
		"100500.00",
		"## Integrity",
		"**Event Log SHA256:**",
	}
	for _, s := range required {
		if !strings.Contains(md, s) {
			t.Errorf("markdown missing %q", s)
		}
	}
}

func TestRenderChartProducesOutput(t *testing.T) {
	rule := domain.Rule{Type: domain.RuleThreshold, PriceField: "close", Value: 100}
	values := []domain.CandleValue{
		{Timestamp: 1000, Close: 90},
		{Timestamp: 1060, Close: 95},
		{Timestamp: 1120, Close: 105},
	}
	png, err := RenderChart(values, rule)
	if err != nil {
		t.Fatal(err)
	}
	if len(png) < 100 {
		t.Fatalf("chart output too small: %d bytes", len(png))
	}
}
