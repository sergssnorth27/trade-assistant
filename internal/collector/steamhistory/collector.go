package steamhistory

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/sergssnorth27/trade-assistant/internal/steam"
	"github.com/sergssnorth27/trade-assistant/internal/steamstats"
)

type groupSource interface {
	GroupIDs(ctx context.Context) ([]string, error) // из steam_groups
}

type pageFetcher interface {
	GroupPage(ctx context.Context, groupID string) ([]byte, error) // steam.Client
}

type statsStore interface {
	Upsert(ctx context.Context, stats []steamstats.VariantStats) error
}

type Collector struct {
	groups groupSource
	pages  pageFetcher
	stats  statsStore

	cfg           steamstats.Config
	rateLimitWait time.Duration
}

func NewCollector(groups groupSource, pages pageFetcher, stats statsStore, cfg steamstats.Config) *Collector {
	return &Collector{
		groups:        groups,
		pages:         pages,
		stats:         stats,
		cfg:           cfg,
		rateLimitWait: time.Minute,
	}
}

// Run обходит все группы из БД: скачивает страницу, считает статистику вариантов и сохраняет её.
// Ошибка одной группы не останавливает обход — она пишется в лог, и коллектор идёт дальше.
func (c *Collector) Run(ctx context.Context) error {
	ids, err := c.groups.GroupIDs(ctx)
	if err != nil {
		return fmt.Errorf("steamhistory: list groups: %w", err)
	}
	log.Printf("steamhistory: %d groups to process", len(ids))

	failed := 0
	for i := 0; i < len(ids); {
		if err := ctx.Err(); err != nil {
			return err
		}

		id := ids[i]
		err := c.processGroup(ctx, id)

		switch {
		case err == nil:
			// ок

		case errors.Is(err, steam.ErrRateLimited):
			log.Printf("steamhistory: rate limited on %s, waiting %s", id, c.rateLimitWait)
			if err := sleep(ctx, c.rateLimitWait); err != nil {
				return err
			}
			continue // i не увеличиваем — повторяем эту же группу

		case ctx.Err() != nil:
			return ctx.Err() // отменили во время запроса — выходим, а не считаем группу сломанной

		case errors.Is(err, steamstats.ErrNotUSD):
			return err // клиент не авторизован — все остальные группы придут в рублях, продолжать бессмысленно

		default:
			failed++
			log.Printf("steamhistory: %v", err)
		}

		i++
		if i%50 == 0 || i == len(ids) {
			log.Printf("steamhistory: progress %d/%d, failed %d", i, len(ids), failed)
		}
	}
	return nil
}

// sleep ждёт d или отмены контекста — что наступит раньше.
func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func (c *Collector) processGroup(ctx context.Context, groupID string) error {
	fetchedAt := time.Now()
	html, err := c.pages.GroupPage(ctx, groupID)
	if err != nil {
		return err
	}
	histories, err := steam.ParseGroupHistory(html)

	var all []steamstats.VariantStats
	for name, variant := range histories {
		s, err := steamstats.Compute(name, groupID, variant, fetchedAt, c.cfg)
		if err != nil {
			return fmt.Errorf("group %s: compute %s: %w", groupID, name, err)
		}
		all = append(all, s)
	}
	errUpsert := c.stats.Upsert(ctx, all)
	if err != nil {
		return errUpsert
	}
	return nil
}
