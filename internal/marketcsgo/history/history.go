// Package history загружает историю продаж предмета с market.csgo (full-history).
package history

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"time"

	"github.com/sergssnorth27/trade-assistant/internal/domain"
)

const baseURL = "https://market.csgo.com/api/v2/full-history/"

var ErrRateLimited = errors.New("market: rate limited")

type response struct {
	Time int64       `json:"time"`
	Data historyData `json:"data"`
}

type historyData struct {
	ID      int64 `json:"id"`
	Sales7d struct {
		USD int `json:"USD"`
	} `json:"sales7d"`
	Sales30d struct {
		USD int `json:"USD"`
	} `json:"sales30d"`
	HistorySales [][]json.Number `json:"history"` // [время, RUB, USD, EUR]
}

type allResp struct {
	History map[string]json.Number `json:"history"` // "имя предмета" → id
}

// Sale — одна продажа на маркете.
type Sale struct {
	Time  time.Time
	Price domain.Price
}

// Parse разбирает ответ full-history. Плохие записи пропускаются,
// их число пишется в лог одной строкой.
func Parse(r io.Reader) ([]Sale, error) {
	var resp response
	if err := json.NewDecoder(r).Decode(&resp); err != nil {
		return nil, fmt.Errorf("decode history: %w", err)
	}

	records := resp.Data.HistorySales
	sales := make([]Sale, 0, len(records))
	skipped := 0
	for _, e := range records {
		if len(e) < 3 {
			skipped++
			continue
		}
		ts, err := e[0].Int64()
		if err != nil {
			skipped++
			continue
		}
		usd, err := e[2].Float64()
		if err != nil {
			skipped++
			continue
		}
		price := domain.Price(math.Round(usd * 1000))
		if price <= 0 {
			skipped++
			continue
		}
		sales = append(sales, Sale{Time: time.Unix(ts, 0), Price: price})
	}

	if skipped > 0 {
		log.Printf("history id %d: skipped %d of %d records", resp.Data.ID, skipped, len(records))
	}
	return sales, nil
}

// Fetch загружает историю продаж предмета по id из full-history.
// На 429 возвращает ошибку, обёрнутую в ErrRateLimited.
func Fetch(ctx context.Context, client *http.Client, id int64) ([]Sale, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s%d.json", baseURL, id), nil)
	if err != nil {
		return nil, fmt.Errorf("history id %d: create request: %w", id, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("history id %d: do request: %w", id, err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusTooManyRequests:
		return nil, fmt.Errorf("history id %d: %w", id, ErrRateLimited)
	case resp.StatusCode != http.StatusOK:
		return nil, fmt.Errorf("history id %d: unexpected status %d", id, resp.StatusCode)
	}

	sales, err := Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("history id %d: %w", id, err)
	}
	return sales, nil
}

func LoadIDs(ctx context.Context, client *http.Client) (map[string]int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://market.csgo.com/api/v2/full-history/all.json", nil)
	if err != nil {
		return nil, fmt.Errorf("full history create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do get req load IDs err: %w", err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusTooManyRequests:
		return nil, fmt.Errorf("get full-history: %w", ErrRateLimited)
	case resp.StatusCode != http.StatusOK:
		return nil, fmt.Errorf("get full-history unexpected status %d", resp.StatusCode)
	}

	var allResp allResp
	if err := json.NewDecoder(resp.Body).Decode(&allResp); err != nil {
		return nil, fmt.Errorf("decode full-history err: %w", err)
	}

	skipped := 0
	var result = make(map[string]int64, len(allResp.History))
	for name, id := range allResp.History {
		idInt, err := id.Int64()
		if err != nil {
			skipped++
			continue
		}
		result[name] = idInt
	}
	if skipped > 0 {
		log.Printf("conver id to int count: %v", skipped)
	}

	return result, nil
}
