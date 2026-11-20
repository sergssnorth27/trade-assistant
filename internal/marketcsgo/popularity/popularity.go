// Package popularity загружает число продаж market.csgo за 7 дней по всем
// предметам одним запросом (prices/class_instance, поле popularity_7d).
// Цены из этого ответа не используются: актуальные цены — в стаканах Owner.
package popularity

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

const endpoint = "https://market.csgo.com/api/v2/prices/class_instance/USD.json"

type response struct {
	Success bool               `json:"success"`
	Items   map[string]itemDTO `json:"items"` // ключ — "classid_instanceid"
}

type itemDTO struct {
	Popularity7d   flexInt `json:"popularity_7d"`
	MarketHashName string  `json:"market_hash_name"`
}

// flexInt принимает и число (47), и строку ("47"); пусто и null → 0.
// market.csgo присылает одно и то же поле в разных форматах.
type flexInt int

func (f *flexInt) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "" || s == "null" {
		*f = 0
		return nil
	}
	n, err := strconv.ParseFloat(s, 64) // на случай "47.0"
	if err != nil {
		return fmt.Errorf("flexInt %s: %w", b, err)
	}
	*f = flexInt(n)
	return nil
}

// Load возвращает продажи за 7 дней по market_hash_name.
func Load(ctx context.Context, client *http.Client) (map[string]int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("popularity: do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("popularity: unexpected status %d", resp.StatusCode)
	}
	return Parse(resp.Body)
}

// Parse суммирует popularity_7d по всем class_instance одного имени.
func Parse(r io.Reader) (map[string]int, error) {
	var resp response
	if err := json.NewDecoder(r).Decode(&resp); err != nil {
		return nil, fmt.Errorf("popularity: decode: %w", err)
	}
	if !resp.Success {
		return nil, fmt.Errorf("popularity: success=false")
	}

	out := make(map[string]int, len(resp.Items))
	for _, it := range resp.Items {
		if it.MarketHashName == "" {
			continue
		}
		out[it.MarketHashName] += int(it.Popularity7d)
	}
	return out, nil
}
