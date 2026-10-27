package snapshot

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/sergssnorth27/trade-assistant/internal/marketcsgo/converter"
)

const (
	url = "https://market.csgo.com/api/v2/prices/USD.json"
)

type List struct {
	Success  bool   `json:"success"`
	Time     int64  `json:"time"`
	Currency string `json:"currency"`
	Items    []Item `json:"items"`
}

type Item struct {
	MarketHashName string                `json:"market_hash_name"`
	Volume         int                   `json:"volume,string"`
	Price          converter.DollarPrice `json:"price"`
}

type InfoShort struct {
	Time      int64
	CountItem int
}

func RunShort(ctx context.Context, out chan InfoShort) error {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var lastTime int64
	for {
		info, changed, err := loadOnceShort(ctx, lastTime)
		if err != nil {
			log.Printf("short snapshot: %v", err)
		} else if changed {
			lastTime = info.Time
			select {
			case out <- info:
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func loadOnceShort(ctx context.Context, lastTime int64) (InfoShort, bool, error) {
	var l List
	if err := getJSON(ctx, url, &l); err != nil {
		return InfoShort{}, false, fmt.Errorf("list: %w", err)
	}

	if !l.Success || l.Time == 0 {
		return InfoShort{}, false, errors.New("invalid index response")
	}

	if l.Time == lastTime {
		return InfoShort{}, false, nil
	}

	return InfoShort{Time: l.Time, CountItem: len(l.Items)}, true, nil
}
