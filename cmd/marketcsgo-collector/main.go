package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
	"github.com/sergssnorth27/trade-assistant/internal/httpapi"
	"github.com/sergssnorth27/trade-assistant/internal/marketcsgo/names"
	"github.com/sergssnorth27/trade-assistant/internal/marketcsgo/snapshot"
	"github.com/sergssnorth27/trade-assistant/internal/marketcsgo/state"
	"github.com/sergssnorth27/trade-assistant/internal/marketcsgo/ws"
)

// setupLog направляет лог одновременно в терминал и в новый файл logs/collector-<время запуска>.log.
func setupLog() (*os.File, error) {
	if err := os.MkdirAll("logs", 0o755); err != nil {
		return nil, fmt.Errorf("create logs dir: %w", err)
	}

	name := filepath.Join("logs", "collector-"+time.Now().Format("2006-01-02_15-04-05")+".log")
	f, err := os.Create(name)
	if err != nil {
		return nil, fmt.Errorf("create log file: %w", err)
	}

	log.SetOutput(io.MultiWriter(os.Stderr, f))
	log.SetFlags(log.LstdFlags | log.Lmicroseconds) // время с микросекундами
	log.Printf("log file: %s", name)
	return f, nil
}

func main() {
	logFile, err := setupLog()
	if err != nil {
		log.Fatalf("setup log: %v", err)
	}
	defer logFile.Close()

	if err := godotenv.Load(); err != nil {
		log.Printf("load .env: %v", err)
	}
	apiKey := os.Getenv("MARKET_CSGO_API")
	if apiKey == "" {
		log.Fatal("MARKET_CSGO_API is not set")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	snapshotChan := make(chan snapshot.Snapshot, 1)
	eventChan := make(chan ws.Event, 10000)

	var dict names.Dict
	if err := dict.Load(ctx); err != nil {
		log.Fatalf("names: %v", err)
	}
	go dict.Run(ctx)
	go ws.Run(ctx, apiKey, eventChan)
	go snapshot.Run(ctx, apiKey, snapshotChan)

	owner := state.New(&dict, "M4A1-S | Printstream (Battle-Scarred)")

	go func() {
		if err := httpapi.Run(ctx, "localhost:8080", httpapi.NewHandler(owner)); err != nil {
			log.Printf("http: %v", err)
		}
	}()
	log.Println("http: listening on http://localhost:8080")

	if err := owner.Run(ctx, snapshotChan, eventChan); err != nil && !errors.Is(err, context.Canceled) {
		log.Printf("owner: %v", err)
	}
}
