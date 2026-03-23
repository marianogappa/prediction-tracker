package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

type ExtractedPrediction struct {
	Statement  string `json:"statement"`
	Asset      string `json:"asset"`
	Operator   string `json:"operator"`
	PriceField string `json:"price_field"`
	Value      float64 `json:"value"`
	Deadline   string `json:"deadline"`
	AuthorName string `json:"author_name,omitempty"`
	SourceURL  string `json:"source_url,omitempty"`
}

type Ingester struct {
	apiKey    string
	apiURL    string
	model     string
	serverURL string
}

func NewIngester(apiKey, apiURL, model, serverURL string) *Ingester {
	if apiURL == "" {
		apiURL = "https://api.openai.com/v1/chat/completions"
	}
	if model == "" {
		model = "gpt-4o-mini"
	}
	if serverURL == "" {
		serverURL = "http://localhost:8080"
	}
	return &Ingester{apiKey: apiKey, apiURL: apiURL, model: model, serverURL: serverURL}
}

const systemPrompt = `You are a prediction extraction engine. Given a natural language prediction about cryptocurrency prices, extract the following structured JSON:

{
  "statement": "the original prediction text",
  "asset": "EXCHANGE:BASE/QUOTE (e.g. BINANCE:BTC/USDT)",
  "operator": "one of: >=, >, <=, <",
  "price_field": "close",
  "value": 50000,
  "deadline": "2026-06-01T00:00:00Z",
  "author_name": "name if mentioned",
  "source_url": "url if mentioned"
}

Rules:
- Default exchange to BINANCE if not specified
- Default quote asset to USDT if not specified
- Default price_field to "close"
- Deadline must be RFC3339 format
- If no deadline is given, use 30 days from now
- Respond ONLY with the JSON, no other text`

func (ing *Ingester) Extract(ctx context.Context, text string) (*ExtractedPrediction, error) {
	if ing.apiKey == "" {
		return nil, errors.New("LLM API key is not configured")
	}

	body := map[string]any{
		"model": ing.model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": text},
		},
		"temperature": 0,
	}
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ing.apiURL, bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ing.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling LLM API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("LLM API returned %d: %s", resp.StatusCode, string(b))
	}

	var llmResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&llmResp); err != nil {
		return nil, fmt.Errorf("decoding LLM response: %w", err)
	}
	if len(llmResp.Choices) == 0 {
		return nil, errors.New("LLM returned no choices")
	}

	var extracted ExtractedPrediction
	if err := json.Unmarshal([]byte(llmResp.Choices[0].Message.Content), &extracted); err != nil {
		return nil, fmt.Errorf("parsing LLM output as JSON: %w (raw: %s)", err, llmResp.Choices[0].Message.Content)
	}
	return &extracted, nil
}

type createRequest struct {
	Statement  string          `json:"statement"`
	Rule       json.RawMessage `json:"rule"`
	Asset      string          `json:"asset"`
	Deadline   time.Time       `json:"deadline"`
	SourceURL  string          `json:"source_url,omitempty"`
	AuthorName string          `json:"author_name,omitempty"`
}

type createResponse struct {
	ID string `json:"id"`
}

func (ing *Ingester) Ingest(ctx context.Context, ep *ExtractedPrediction) (string, error) {
	deadline, err := time.Parse(time.RFC3339, ep.Deadline)
	if err != nil {
		return "", fmt.Errorf("parsing deadline: %w", err)
	}

	rule := map[string]any{
		"type":        "threshold",
		"operator":    ep.Operator,
		"price_field": ep.PriceField,
		"value":       ep.Value,
		"deadline":    ep.Deadline,
	}
	ruleJSON, _ := json.Marshal(rule)

	req := createRequest{
		Statement:  ep.Statement,
		Rule:       ruleJSON,
		Asset:      ep.Asset,
		Deadline:   deadline,
		SourceURL:  ep.SourceURL,
		AuthorName: ep.AuthorName,
	}
	reqJSON, _ := json.Marshal(req)

	resp, err := http.Post(ing.serverURL+"/api/predictions", "application/json", bytes.NewReader(reqJSON))
	if err != nil {
		return "", fmt.Errorf("posting to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("server returned %d: %s", resp.StatusCode, string(b))
	}

	var created createResponse
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		return "", fmt.Errorf("decoding response: %w", err)
	}

	enableReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, ing.serverURL+"/api/predictions/"+created.ID+"/enable", nil)
	enableReq.Header.Set("Accept", "application/json")
	enableResp, err := http.DefaultClient.Do(enableReq)
	if err != nil {
		return created.ID, fmt.Errorf("enabling prediction: %w", err)
	}
	defer enableResp.Body.Close()

	return created.ID, nil
}
