package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/marianogappa/predictions-tracker/plugin/llm"
)

func main() {
	apiKey := flag.String("api-key", os.Getenv("OPENAI_API_KEY"), "OpenAI API key")
	apiURL := flag.String("api-url", os.Getenv("LLM_API_URL"), "LLM API URL (OpenAI-compatible)")
	model := flag.String("model", os.Getenv("LLM_MODEL"), "LLM model name")
	serverURL := flag.String("server", os.Getenv("PREDICTIONS_SERVER_URL"), "Predictions server URL")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Usage: ingest [flags] \"<prediction text>\"")
		os.Exit(1)
	}

	text := flag.Arg(0)
	ing := llm.NewIngester(*apiKey, *apiURL, *model, *serverURL)

	ctx := context.Background()
	extracted, err := ing.Extract(ctx, text)
	if err != nil {
		log.Fatalf("extraction failed: %v", err)
	}

	fmt.Printf("Extracted prediction:\n")
	fmt.Printf("  Statement:   %s\n", extracted.Statement)
	fmt.Printf("  Asset:       %s\n", extracted.Asset)
	fmt.Printf("  Condition:   %s %s %.2f\n", extracted.PriceField, extracted.Operator, extracted.Value)
	fmt.Printf("  Deadline:    %s\n", extracted.Deadline)
	fmt.Printf("  Author:      %s\n", extracted.AuthorName)
	fmt.Printf("  Source:      %s\n", extracted.SourceURL)

	id, err := ing.Ingest(ctx, extracted)
	if err != nil {
		log.Fatalf("ingestion failed: %v", err)
	}
	fmt.Printf("\nPrediction created and enabled: %s\n", id)
}
