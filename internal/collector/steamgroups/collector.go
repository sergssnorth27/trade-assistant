package steamgroups

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/sergssnorth27/trade-assistant/internal/steam"
)

const (
	pageSize        = 30          // столько групп отдаёт одна страница поиска
	maxFails        = 3           // ошибок подряд на одной странице, после которых сдаёмся
	rateLimitPause  = time.Minute // пауза после 429
	logPageInterval = 10          // писать прогресс раз в N страниц
)

type searcher interface {
	SearchPage(ctx context.Context, start int) ([]steam.Group, int, error)
}

type store interface {
	UpsertGroups(ctx context.Context, ids, names []string) error
}

type Collector struct {
	search searcher
	store  store
}

func NewCollector(search searcher, store store) *Collector {
	return &Collector{search: search, store: store}
}

// Run проходит все страницы поиска и записывает группы в базу страница за страницей.
// Останавливается при отмене ctx или после maxFails ошибок подряд на одной странице.
func (c *Collector) Run(ctx context.Context) error {
	began := time.Now()

	total := 1 // настоящее значение придёт с первой страницей; 1 — чтобы войти в цикл
	start, page, saved, fails := 0, 0, 0, 0

	for start < total {
		groups, count, err := c.search.SearchPage(ctx, start)

		switch {
		case ctx.Err() != nil:
			return ctx.Err() // нас остановили

		case errors.Is(err, steam.ErrRateLimited):
			log.Printf("steamgroups: 429 на start=%d, ждём %v", start, rateLimitPause)
			if err := sleep(ctx, rateLimitPause); err != nil {
				return err
			}
			continue // та же страница

		case err != nil:
			fails++
			if fails >= maxFails {
				return fmt.Errorf("steamgroups: start=%d: %d ошибок подряд: %w", start, fails, err)
			}
			log.Printf("steamgroups: start=%d: %v (попытка %d из %d)", start, err, fails, maxFails)
			continue // та же страница
		}

		fails = 0
		total = count

		ids := make([]string, 0, len(groups))
		names := make([]string, 0, len(groups))
		for _, g := range groups {
			ids = append(ids, g.ID)
			names = append(names, g.Name)
		}
		if err := c.store.UpsertGroups(ctx, ids, names); err != nil {
			return fmt.Errorf("steamgroups: save start=%d: %w", start, err)
		}

		saved += len(groups)
		start += pageSize // двигаемся только после успешной записи
		page++

		if page%logPageInterval == 0 {
			log.Printf("steamgroups: страниц %d, групп записано %d из %d, прошло %v",
				page, saved, total, time.Since(began).Round(time.Second))
		}
	}

	log.Printf("steamgroups: готово, страниц %d, групп записано %d из %d за %v",
		page, saved, total, time.Since(began).Round(time.Second))
	return nil
}

// sleep ждёт d или отмены контекста.
func sleep(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}
