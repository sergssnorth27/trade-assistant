package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/joho/godotenv"
	"github.com/sergssnorth27/trade-assistant/internal/collector/steamgroups"
	"github.com/sergssnorth27/trade-assistant/internal/steam"
	"github.com/sergssnorth27/trade-assistant/internal/storage/postgres"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := godotenv.Load(); err != nil {
		log.Printf("load .env: %v", err)
	}
	dsn := os.Getenv("GOOSE_DBSTRING")
	if dsn == "" {
		log.Fatal("dsn is not set")
	}

	client := steam.NewClient()

	pool, err := postgres.NewPool(ctx, dsn)
	if err != nil {
		log.Fatalf("create new pool, err: %v", err)
	}
	defer pool.Close()

	searchRepo := postgres.NewSteamGroupsRepo(pool)
	searchGroupCollector := steamgroups.NewCollector(client, searchRepo)
	if err := searchGroupCollector.Run(ctx); err != nil {
		log.Printf("search group collector run: %v", err)
	}
}
