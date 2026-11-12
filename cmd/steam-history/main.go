package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/joho/godotenv"
	"github.com/sergssnorth27/trade-assistant/internal/collector/steamhistory"
	"github.com/sergssnorth27/trade-assistant/internal/steam"
	"github.com/sergssnorth27/trade-assistant/internal/steam/auth"
	"github.com/sergssnorth27/trade-assistant/internal/steamstats"
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

	login := os.Getenv("STEAM_LOGIN")
	password := os.Getenv("STEAM_PASSWORD")
	sharedSecret := os.Getenv("STEAM_SHARED_SECRET")

	steamLoginSecure, err := auth.Login(ctx, &http.Client{}, login, password, sharedSecret)
	if err != nil {
		log.Fatalf("steam auth: %v", err)
	}
	client, err := steam.NewAuthorizedClient(steamLoginSecure)
	if err != nil {
		log.Fatalf("fail create auth user steam: %v", err)
	}

	pool, err := postgres.NewPool(ctx, dsn)
	if err != nil {
		log.Fatalf("create new pool, err: %v", err)
	}
	defer pool.Close()

	groupsRepo := postgres.NewSteamGroupsRepo(pool)
	statsRepo := postgres.NewSteamVariantStatsRepo(pool)
	steamHistoryCollector := steamhistory.NewCollector(groupsRepo, client, statsRepo, steamstats.Config{HorizonDays: 7})
	if err := steamHistoryCollector.Run(ctx); err != nil {
		log.Printf("search group collector run: %v", err)
	}
}
