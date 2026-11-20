package steamhistory

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/sergssnorth27/trade-assistant/internal/domain"
	"github.com/sergssnorth27/trade-assistant/internal/steam"
	"github.com/sergssnorth27/trade-assistant/internal/steamstats"
)

// max429InRow — сколько ответов 429 подряд считаем баном: Run прекращает проход.
const max429InRow = 3

type groupSource interface {
	GroupIDs(ctx context.Context, maxPrice domain.Price, minPrice domain.Price, sellListings int) ([]string, error) // из steam_groups
}

type pageFetcher interface {
	GroupPage(ctx context.Context, groupID string) ([]byte, error) // steam.Client
}

type statsStore interface {
	Upsert(ctx context.Context, stats []steamstats.VariantStats) error
}

// Options — какие группы обходить и сколько за один проход.
type Options struct {
	MaxPrice    domain.Price // группы, где даже самый дешёвый вариант дороже, пропускаем ($100 = 100_000)
	MinPrice    domain.Price // нижняя граница — только для наклеек и других предметов без вариантов ($0.20 = 200)
	MinListings int          // минимум лотов у группы
	Budget      int          // сколько групп обработать за проход; 0 — без ограничения
}

type Collector struct {
	groups groupSource
	pages  pageFetcher
	stats  statsStore

	cfg           steamstats.Config
	opts          Options
	rateLimitWait time.Duration
}

func NewCollector(groups groupSource, pages pageFetcher, stats statsStore, opts Options, cfg steamstats.Config) *Collector {
	return &Collector{
		groups:        groups,
		pages:         pages,
		stats:         stats,
		opts:          opts,
		cfg:           cfg,
		rateLimitWait: time.Minute,
	}
}

func (c *Collector) Loop(ctx context.Context, cyclePause, banPause time.Duration) error {
	for {
		err := c.Run(ctx)
		var pause time.Duration

		switch {
		case ctx.Err() != nil:
			return ctx.Err() // нас остановили

		case err == nil:
			pause = cyclePause
			log.Printf("steamhistory: cycle done, next in %s", pause)

		case errors.Is(err, steam.ErrRateLimited):
			pause = banPause
			log.Printf("steamhistory: %v; next try in %s", err, pause)

		default:
			return err // ErrNotUSD, ошибка БД и т. п. — сами не исправятся
		}

		if err := sleep(ctx, pause); err != nil {
			return err
		}
	}
}

// Run — один проход: не больше Budget групп, которые давно не обновлялись.
// Ошибка одной группы не останавливает проход. После max429InRow ответов 429 подряд
// возвращает ошибку, оборачивающую steam.ErrRateLimited.
func (c *Collector) Run(ctx context.Context) error {
	ids, err := c.groups.GroupIDs(ctx, c.opts.MaxPrice, c.opts.MinPrice, c.opts.MinListings)
	if err != nil {
		return fmt.Errorf("steamhistory: list groups: %w", err)
	}
	if c.opts.Budget > 0 && len(ids) > c.opts.Budget {
		ids = ids[:c.opts.Budget] // остальные возьмёт следующий проход
	}
	log.Printf("steamhistory: %d groups to process", len(ids))

	failed, rateLimitedInRow := 0, 0
	for i := 0; i < len(ids); {
		if err := ctx.Err(); err != nil {
			return err
		}

		id := ids[i]
		err := c.processGroup(ctx, id)

		switch {
		case err == nil:
			rateLimitedInRow = 0

		case errors.Is(err, steam.ErrRateLimited):
			rateLimitedInRow++
			if rateLimitedInRow >= max429InRow {
				return fmt.Errorf("steamhistory: 429 %d times in a row, stopped at %d/%d: %w",
					rateLimitedInRow, i, len(ids), err)
			}
			log.Printf("steamhistory: rate limited on %s (%d/%d), waiting %s",
				id, rateLimitedInRow, max429InRow, c.rateLimitWait)
			if err := sleep(ctx, c.rateLimitWait); err != nil {
				return err
			}
			continue // та же группа

		case ctx.Err() != nil:
			return ctx.Err()

		case errors.Is(err, steamstats.ErrNotUSD):
			return err

		default:
			rateLimitedInRow = 0
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
		return fmt.Errorf("group %s: fetch: %w", groupID, err)
	}
	histories, err := steam.ParseGroupHistory(html)
	if err != nil {
		return fmt.Errorf("group %s: parse: %w", groupID, err)
	}

	var all []steamstats.VariantStats
	for name, variant := range histories {
		s, err := steamstats.Compute(name, groupID, variant, fetchedAt, c.cfg)
		if err != nil {
			return fmt.Errorf("group %s: compute %s: %w", groupID, name, err)
		}
		all = append(all, s)
	}
	if err := c.stats.Upsert(ctx, all); err != nil {
		return fmt.Errorf("group %s: upsert: %w", groupID, err)
	}
	return nil
}
