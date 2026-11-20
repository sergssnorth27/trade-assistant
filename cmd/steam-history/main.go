// steam-history — по кругу собирает историю продаж Steam по группам из БД
// и пишет статистику вариантов в steam_variant_stats.
//
// caffeinate -i go run ./cmd/steam-history
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/sergssnorth27/trade-assistant/internal/collector/steamhistory"
	"github.com/sergssnorth27/trade-assistant/internal/steam"
	"github.com/sergssnorth27/trade-assistant/internal/steam/auth"
	"github.com/sergssnorth27/trade-assistant/internal/steamstats"
	"github.com/sergssnorth27/trade-assistant/internal/storage/postgres"
)

func main() {
	// Ctrl+C и docker stop — штатная остановка, в том числе посреди 6-часовой паузы.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := godotenv.Load(); err != nil {
		log.Printf("load .env: %v", err)
	}

	dsn := os.Getenv("GOOSE_DBSTRING")
	if dsn == "" {
		log.Fatal("GOOSE_DBSTRING is not set")
	}

	login := os.Getenv("STEAM_LOGIN")
	password := os.Getenv("STEAM_PASSWORD")
	sharedSecret := os.Getenv("STEAM_SHARED_SECRET")
	if login == "" || password == "" || sharedSecret == "" {
		log.Fatal("STEAM_LOGIN, STEAM_PASSWORD and STEAM_SHARED_SECRET must be set")
	}

	// Авторизация: без неё Steam отдаёт историю в рублях (по IP), и Compute вернёт ErrNotUSD.
	authHTTP := &http.Client{Timeout: 30 * time.Second}
	steamLoginSecure, err := auth.Login(ctx, authHTTP, login, password, sharedSecret)
	if err != nil {
		log.Fatalf("steam auth: %v", err) // саму cookie никогда не печатаем
	}
	client, err := steam.NewAuthorizedClient(steamLoginSecure)
	if err != nil {
		log.Fatalf("steam client: %v", err)
	}

	pool, err := postgres.NewPool(ctx, dsn)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer pool.Close()

	collector := steamhistory.NewCollector(
		postgres.NewSteamGroupsRepo(pool),
		client,
		postgres.NewSteamVariantStatsRepo(pool),
		steamhistory.Options{
			MaxPrice:    100_000, // $100: группы, где даже самый дешёвый вариант дороже, не берём
			MinPrice:    200,     // $0.20: только для наклеек, нашивок и других предметов без вариантов
			MinListings: 100,     // минимум лотов у группы на всех вариантах вместе
			Budget:      1500,    // групп за проход: меньше порога, после которого Steam банил (~2000)
		},
		steamstats.Config{HorizonDays: 7},
	)

	const (
		cyclePause = 6 * time.Hour // пауза после нормального прохода
		banPause   = 6 * time.Hour // пауза после бана Steam (3 раза 429 подряд)
	)
	err = collector.Loop(ctx, cyclePause, banPause)
	if err != nil && !errors.Is(err, context.Canceled) {
		log.Printf("steam history collector: %v", err)
	}
	log.Println("steam history collector stopped")
}
