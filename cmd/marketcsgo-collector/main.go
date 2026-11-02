package main

import (
	"context"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/sergssnorth27/trade-assistant/internal/marketcsgo/names"
	"github.com/sergssnorth27/trade-assistant/internal/marketcsgo/snapshot"
	"github.com/sergssnorth27/trade-assistant/internal/marketcsgo/state"
	"github.com/sergssnorth27/trade-assistant/internal/marketcsgo/ws"
)

type Price int64

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("err load env: %v", err)
	}
	marketCSGOAPIKEY := os.Getenv("MARKET_CSGO_API")

	if marketCSGOAPIKEY == "" {
		log.Fatal("MARKET_CSGO_API is not set")
	}
	ctx := context.Background()

	snapshotChan := make(chan snapshot.Snapshot, 1)
	eventChan := make(chan ws.Event, 10000)

	var dict names.Dict
	if err := dict.Load(ctx); err != nil { // main ждёт словарь
		log.Fatalf("names: %v", err)
	}

	go dict.Run(ctx) // дальше раз в час в фоне

	go func() {
		ws.Run(ctx, marketCSGOAPIKEY, eventChan)
	}()

	go func() {
		snapshot.Run(ctx, marketCSGOAPIKEY, snapshotChan)
	}()

	owner := state.New(&dict, "M4A1-S | Printstream (Battle-Scarred)")
	owner.Run(ctx, snapshotChan, eventChan)
}
