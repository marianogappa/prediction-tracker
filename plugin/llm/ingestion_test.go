package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExtractCallsLLMAndParsesResponse(t *testing.T) {
	extracted := ExtractedPrediction{
		Statement:  "BTC will hit 100k by end of 2026",
		Asset:      "BINANCE:BTC/USDT",
		Operator:   ">=",
		PriceField: "close",
		Value:      100000,
		Deadline:   "2026-12-31T00:00:00Z",
	}
	extractedJSON, _ := json.Marshal(extracted)

	llmServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"content": string(extractedJSON)}},
			},
		})
	}))
	defer llmServer.Close()

	ing := NewIngester("test-key", llmServer.URL, "test-model", "")
	got, err := ing.Extract(context.Background(), "I think BTC will hit 100k by end of 2026")
	if err != nil {
		t.Fatal(err)
	}
	if got.Asset != "BINANCE:BTC/USDT" || got.Value != 100000 {
		t.Fatalf("unexpected extraction: %+v", got)
	}
}

func TestIngestCreatesPredictionOnServer(t *testing.T) {
	var gotCreate, gotEnable bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/predictions":
			gotCreate = true
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"id": "pred-123"})
		case r.Method == http.MethodPost && r.URL.Path == "/api/predictions/pred-123/enable":
			gotEnable = true
			json.NewEncoder(w).Encode(map[string]string{"id": "pred-123", "state": "enabled"})
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer server.Close()

	ing := NewIngester("", "", "", server.URL)
	ep := &ExtractedPrediction{
		Statement:  "BTC to 100k",
		Asset:      "BINANCE:BTC/USDT",
		Operator:   ">=",
		PriceField: "close",
		Value:      100000,
		Deadline:   "2026-12-31T00:00:00Z",
	}

	id, err := ing.Ingest(context.Background(), ep)
	if err != nil {
		t.Fatal(err)
	}
	if id != "pred-123" {
		t.Fatalf("expected id pred-123, got %s", id)
	}
	if !gotCreate || !gotEnable {
		t.Fatalf("expected both create and enable calls, got create=%v enable=%v", gotCreate, gotEnable)
	}
}
