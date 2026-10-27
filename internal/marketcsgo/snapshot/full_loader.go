package snapshot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"
)

const (
	urlIndex     = "https://market.csgo.com/api/full-export/USD.json"
	urlChunkBase = "https://market.csgo.com/api/full-export/"
)

type exportIndex struct {
	Success  bool     `json:"success"`
	Time     int64    `json:"time"`
	Currency string   `json:"currency"`
	Format   []string `json:"format"`
	Items    []string `json:"items"`
}

// type Item struct {
// 	Price          int64  `json:"price"`
// 	ID             int    `json:"id"`
// 	MarketHashName string `json:"market_hash_name"`
// 	ClassID        string `json:"classid"`
// 	InstanceID     int    `json:"instanceid"`
// }

type Info struct {
	Time      int64
	CountItem int
}

// Run раз в 30 секунд проверяет снапшот и отправляет Info, когда он обновился.
func Run(ctx context.Context, apiKey string, out chan<- Info) error {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var lastTime int64

	for {
		info, changed, err := loadOnce(ctx, lastTime)
		if err != nil {
			log.Printf("snapshot: %v", err) // логируем и ждём следующего тика
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

func loadOnce(ctx context.Context, lastTIme int64) (Info, bool, error) {
	var idx exportIndex
	if err := getJSON(ctx, urlIndex, &idx); err != nil {
		return Info{}, false, fmt.Errorf("index: %w", err)
	}

	if !idx.Success || idx.Time == 0 {
		return Info{}, false, errors.New("invalid index response")
	}

	if idx.Time == lastTIme {
		return Info{}, false, nil
	}

	count := 0
	for _, name := range idx.Items {
		var rows [][]json.RawMessage
		if err := getJSON(ctx, urlChunkBase+name, &rows); err != nil {
			return Info{}, false, fmt.Errorf("chunk %s: %w", name, err)
		}
		count += len(rows)
	}
	return Info{Time: idx.Time, CountItem: count}, true, nil
}

// getJSON делает GET-запрос и декодирует ответ в dst.
func getJSON(ctx context.Context, url string, dst any) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return fmt.Errorf("decode: %w", err)
	}
	return nil

}
