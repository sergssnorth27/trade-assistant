// Команда marketcsgo-history раз в сутки собирает историю продаж market.csgo
// по кандидатам для заноса и пишет статистику в market_variant_stats.
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/sergssnorth27/trade-assistant/internal/collector/marketcsgohistory"
	"github.com/sergssnorth27/trade-assistant/internal/storage/postgres"
)

const (
	minSteamSales7d = 35            // кандидат должен продаваться в Steam не меньше 35 раз за 7 дней
	passPause       = 3 * time.Hour // пауза между проходами
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("no .env file: %v", err)
	}
	logFile, err := setupLog()
	if err != nil {
		log.Fatalf("setup log: %v", err)
	}
	defer logFile.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dsn := os.Getenv("GOOSE_DBSTRING")
	if dsn == "" {
		log.Fatal("GOOSE_DBSTRING is not set")
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("ping db: %v", err)
	}

	collector := marketcsgohistory.New(
		&http.Client{Timeout: 30 * time.Second},
		postgres.NewSteamVariantStatsRepo(pool),
		postgres.NewMarketVariantStatsRepo(pool),
		marketcsgohistory.Options{MinSteamSales7d: minSteamSales7d},
	)

	if err := collector.Loop(ctx, passPause); err != nil && ctx.Err() == nil {
		log.Printf("loop stopped: %v", err)
	}
	log.Print("marketcsgo-history: stopped")
}

// setupLog пишет лог и в консоль, и в logs/marketcsgo-history-<время>.log.
func setupLog() (*os.File, error) {
	if err := os.MkdirAll("logs", 0o755); err != nil {
		return nil, err
	}
	name := fmt.Sprintf("logs/marketcsgo-history-%s.log", time.Now().Format("2006-01-02_15-04-05"))
	f, err := os.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}
	log.SetOutput(io.MultiWriter(os.Stdout, f))
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Printf("log file: %s", name)
	return f, nil
}
