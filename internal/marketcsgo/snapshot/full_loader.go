package snapshot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

const (
	urlIndex     = "https://market.csgo.com/api/full-export/USD.json"
	urlChunkBase = "https://market.csgo.com/api/full-export/"
	pollInterval = 15 * time.Second
	chunkWorkers = 8
	slowChunk    = 2 * time.Second // чанки дольше этого пишем в лог
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

var httpClient = newHttpClient()

func newHttpClient() *http.Client {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.IdleConnTimeout = 15 * time.Second
	t.MaxIdleConnsPerHost = chunkWorkers
	return &http.Client{Transport: t}
}

// Run раз в 30 секунд проверяет снапшот и отправляет Info, когда он обновился.
func Run(ctx context.Context, apiKey string, out chan<- Info) error {
	ticker := time.NewTicker(pollInterval)
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

var logCompression sync.Once

func loadOnce(ctx context.Context, lastTIme int64) (Info, bool, error) {
	start := time.Now()
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

	counts := make([]int, len(idx.Items))
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(chunkWorkers)

	for i, name := range idx.Items {
		g.Go(func() error {
			chunkStart := time.Now()

			var rows [][]json.RawMessage
			if err := getJSON(gctx, urlChunkBase+name, &rows); err != nil {
				return fmt.Errorf("chunk %s: %w", name, err)
			}
			if d := time.Since(chunkStart); d > slowChunk {
				log.Printf("snapshot: slow chunk %s: %v", name, d.Round(time.Millisecond))
			}
			if len(rows) != 10000 {
				log.Printf("snapshot: chunk %d/%d %s: %d rows", i+1, len(idx.Items), name, len(rows))
			}

			counts[i] = len(rows)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return Info{}, false, err // хоть один чанк упал — весь снапшот не годится
	}

	total := 0
	for _, c := range counts {
		total += c
	}

	log.Printf("snapshot: %d lots, %d chunks, %v (workers=%d)",
		total, len(idx.Items), time.Since(start).Round(time.Millisecond), chunkWorkers)

	return Info{Time: idx.Time, CountItem: total}, true, nil
}

// getJSON делает GET-запрос и декодирует ответ в dst.
func getJSON(ctx context.Context, url string, dst any) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	logCompression.Do(func() {
		log.Printf("snapshot: gzip=%v", resp.Uncompressed)
	})

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return fmt.Errorf("decode: %w", err)
	}
	return nil

}
