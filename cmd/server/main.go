package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/marianogappa/predictions-tracker/api"
	"github.com/marianogappa/predictions-tracker/eval"
	"github.com/marianogappa/predictions-tracker/event"
	"github.com/marianogappa/predictions-tracker/poller"
	"github.com/marianogappa/predictions-tracker/source/crypto"
	"github.com/marianogappa/predictions-tracker/storage/sqlite"
)

func main() {
	dbPath := flag.String("db", envOr("PREDICTIONS_DB_PATH", "./predictions.db"), "SQLite database path")
	pollInterval := flag.Duration("poll-interval", envDuration("PREDICTIONS_POLL_INTERVAL", 60*time.Second), "Polling interval")
	listenAddr := flag.String("listen", envOr("PREDICTIONS_LISTEN_ADDR", ":8080"), "HTTP listen address")
	flag.Parse()

	store, err := sqlite.New(*dbPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer store.Close()

	bus := event.NewBus()
	engine := eval.NewEngine()
	source := crypto.NewAdapter()

	p := poller.New(store, source, engine, bus, *pollInterval)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go p.Run(ctx)

	srv, err := api.NewServer(store, bus, engine)
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}

	httpServer := &http.Server{
		Addr:    *listenAddr,
		Handler: srv,
	}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("shutting down...")
		cancel()
		shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutCancel()
		httpServer.Shutdown(shutCtx)
	}()

	log.Printf("starting server on %s (db=%s, poll=%s)", *listenAddr, *dbPath, *pollInterval)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		d, err := time.ParseDuration(v)
		if err == nil {
			return d
		}
	}
	return fallback
}
