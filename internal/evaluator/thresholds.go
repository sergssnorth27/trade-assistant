package evaluator

import (
	"context"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/sergssnorth27/trade-assistant/internal/steamstats"
)

// statsSource — откуда брать статистику (SteamVariantStatsRepo).
type statsSource interface {
	LoadForThresholds(ctx context.Context) ([]steamstats.VariantStats, error)
}

type Thresholds struct {
	current  atomic.Pointer[map[string]Threshold]
	source   statsSource
	cfg      Config
	interval time.Duration
}

func NewThresholds(source statsSource, cfg Config, interval time.Duration) *Thresholds {
	return &Thresholds{
		source:   source,
		cfg:      cfg,
		interval: interval,
	}
}

func (t *Thresholds) Get(name string) (Threshold, bool) {
	m := t.current.Load()
	if m == nil {
		return Threshold{}, false
	}
	th, ok := (*m)[name]
	return th, ok
}

func (t *Thresholds) Load(ctx context.Context) error {
	stats, err := t.source.LoadForThresholds(ctx)
	if err != nil {
		return fmt.Errorf("thresholds: load stats: %w", err)
	}
	thresholds, rejects := BuildThresholds(stats, time.Now(), t.cfg)
	log.Printf("loaded %v, good items: %v, rejects L %v", len(stats), len(thresholds), rejects)
	t.current.Store(&thresholds)
	return nil
}

func (t *Thresholds) Run(ctx context.Context) error {
	if err := t.Load(ctx); err != nil {
		return err
	}
	ticker := time.NewTicker(t.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := t.Load(ctx); err != nil {
				log.Printf("load thresholds err: %v", err)
			}
		}
	}
}
