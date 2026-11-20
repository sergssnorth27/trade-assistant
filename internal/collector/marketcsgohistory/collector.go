// Package marketcsgohistory собирает историю продаж market.csgo (full-history)
// по кандидатам для заноса и пишет статистику в market_variant_stats.
package marketcsgohistory

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/sergssnorth27/trade-assistant/internal/marketcsgo/history"
	"github.com/sergssnorth27/trade-assistant/internal/marketstats"
)

const (
	requestPause  = 500 * time.Millisecond // 2 запроса в секунду
	batchSize     = 100
	progressEvery = 100
	max429InRow   = 10
)

var ErrRateLimited = errors.New("marketcsgo history: rate limited, pass stopped")

type steamStatsStore interface {
	LoadInflowCandidates(ctx context.Context, minSales7d int) ([]string, error)
}

type marketStatsStore interface {
	Upsert(ctx context.Context, stats []marketstats.MarketStats) error
}

type Options struct {
	MinSteamSales7d int // кандидат должен продаваться в Steam не меньше N раз за 7 дней
	Limit           int // сколько предметов обработать за проход; 0 — все (удобно для первого запуска)
}

type Collector struct {
	client      *http.Client
	steamStats  steamStatsStore
	marketStats marketStatsStore
	opts        Options
}

func New(client *http.Client, steamStats steamStatsStore, marketStats marketStatsStore, opts Options) *Collector {
	return &Collector{client: client, steamStats: steamStats, marketStats: marketStats, opts: opts}
}

type candidate struct {
	name string
	id   int64
}

// Loop запускает проход, потом ждёт pause, и так до отмены ctx.
// Ошибка прохода (в том числе 429) не останавливает цикл: следующий проход — после паузы.
func (c *Collector) Loop(ctx context.Context, pause time.Duration) error {
	for {
		if err := c.Run(ctx); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			log.Printf("marketcsgo history: pass failed: %v", err)
		}

		log.Printf("marketcsgo history: next pass in %s", pause)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pause):
		}
	}
}

// Run — один проход по всем кандидатам.
func (c *Collector) Run(ctx context.Context) error {
	start := time.Now()

	candidates, err := c.candidates(ctx)
	if err != nil {
		return err
	}
	if c.opts.Limit > 0 && len(candidates) > c.opts.Limit {
		candidates = candidates[:c.opts.Limit]
	}

	ticker := time.NewTicker(requestPause)
	defer ticker.Stop()

	batch := make([]marketstats.MarketStats, 0, batchSize)
	var done, failed, rateLimitedInRow int

	for _, cand := range candidates {
		select {
		case <-ctx.Done():
			c.flush(context.Background(), batch) // не теряем уже посчитанное
			return ctx.Err()
		case <-ticker.C:
		}

		stats, err := c.processItem(ctx, cand)
		switch {
		case errors.Is(err, history.ErrRateLimited):
			rateLimitedInRow++
			failed++
			log.Printf("marketcsgo history: %s: rate limited (%d in a row)", cand.name, rateLimitedInRow)
			if rateLimitedInRow >= max429InRow {
				c.flush(ctx, batch)
				return fmt.Errorf("after %d/%d items: %w", done, len(candidates), ErrRateLimited)
			}
			continue
		case err != nil:
			failed++
			log.Printf("marketcsgo history: %s: %v", cand.name, err)
			continue
		}

		rateLimitedInRow = 0
		done++
		batch = append(batch, stats)

		if len(batch) == batchSize {
			if err := c.marketStats.Upsert(ctx, batch); err != nil {
				return fmt.Errorf("upsert market stats: %w", err)
			}
			batch = batch[:0]
		}

		if (done+failed)%progressEvery == 0 {
			log.Printf("marketcsgo history: progress %d/%d, ok %d, failed %d, elapsed %s",
				done+failed, len(candidates), done, failed, time.Since(start).Round(time.Second))
		}
	}

	if err := c.marketStats.Upsert(ctx, batch); err != nil {
		return fmt.Errorf("upsert market stats: %w", err)
	}

	log.Printf("marketcsgo history: pass done: %d ok, %d failed, %s",
		done, failed, time.Since(start).Round(time.Second))
	return nil
}

// candidates — кандидаты из Steam, у которых есть id в словаре full-history.
func (c *Collector) candidates(ctx context.Context) ([]candidate, error) {
	ids, err := history.LoadIDs(ctx, c.client)
	if err != nil {
		return nil, fmt.Errorf("load full-history ids: %w", err)
	}
	names, err := c.steamStats.LoadInflowCandidates(ctx, c.opts.MinSteamSales7d)
	if err != nil {
		return nil, fmt.Errorf("load inflow candidates: %w", err)
	}

	out := make([]candidate, 0, len(names))
	for _, name := range names {
		if id, ok := ids[name]; ok {
			out = append(out, candidate{name: name, id: id})
		}
	}

	log.Printf("marketcsgo history: steam candidates %d, in dictionary %d, missing %d",
		len(names), len(out), len(names)-len(out))
	return out, nil
}

func (c *Collector) processItem(ctx context.Context, cand candidate) (marketstats.MarketStats, error) {
	sales, err := history.Fetch(ctx, c.client, cand.id)
	if err != nil {
		return marketstats.MarketStats{}, fmt.Errorf("fetch: %w", err)
	}
	return marketstats.Compute(cand.name, cand.id, sales, time.Now()), nil
}

// flush пишет остаток пачки при досрочном выходе; ошибку только логирует.
func (c *Collector) flush(ctx context.Context, batch []marketstats.MarketStats) {
	if len(batch) == 0 {
		return
	}
	if err := c.marketStats.Upsert(ctx, batch); err != nil {
		log.Printf("marketcsgo history: flush %d stats: %v", len(batch), err)
	}
}
