package steam

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
	"unicode"

	"github.com/sergssnorth27/trade-assistant/internal/domain"
)

const (
	appCS2      = 730
	currencyUSD = 1
	searchURL   = "https://steamcommunity.com/market/actions?q=Search&l=english&qp="
)

type searchResponse struct {
	Data struct {
		Results []struct {
			AssetDescription struct {
				GroupID       string `json:"market_bucket_group_id"`
				GroupHashName string `json:"market_bucket_group_name"`
			} `json:"asset_description"`
			SellListings    int    `json:"cSellOrders"`
			MinSellSubtotal string `json:"strMinSellSubtotal"`
			Currency        int    `json:"eCurrency"`
		} `json:"results"`
		TotalCount int `json:"total_count"`
	} `json:"data"`
}

type Group struct {
	ID           string
	Name         string
	MinPrice     *domain.Price
	SellListings int
}

// parseSteamPrice переводит цену из поиска Steam в тысячные доли доллара:
// "$4.56", "4,56 $", "$1,234.56", "1 234,56 $" → 4560, 4560, 1234560, 1234560.
// Steam всегда показывает два знака после разделителя, поэтому все цифры подряд — это центы.
func parseSteamPrice(s string) (domain.Price, error) {
	var digits strings.Builder
	lastGroupLen := 0 // длина последней группы цифр: должна быть 2 (копейки/центы)
	inGroup := false

	for _, r := range s {
		if unicode.IsDigit(r) {
			digits.WriteRune(r)
			if !inGroup {
				lastGroupLen = 0
				inGroup = true
			}
			lastGroupLen++
		} else {
			inGroup = false
		}
	}

	if digits.Len() == 0 {
		return 0, fmt.Errorf("steam price %q: no digits", s)
	}
	if lastGroupLen != 2 {
		return 0, fmt.Errorf("steam price %q: expected 2 decimal digits", s)
	}

	cents, err := strconv.ParseInt(digits.String(), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("steam price %q: %w", s, err)
	}
	return domain.Price(cents * 10), nil
}

func (c *Client) SearchPage(ctx context.Context, start int) ([]Group, int, error) {
	qp, err := json.Marshal([]any{map[string]any{
		"appid":   appCS2,
		"filters": map[string]any{},
		"price": map[string]any{
			"eCurrency": currencyUSD,
		},
		"accessoryFilters": map[string]any{},
		"sort":             1,
		"direction":        1,
		"start":            start,
	}})
	if err != nil {
		return nil, 0, fmt.Errorf("marshal json: %w", err)
	}

	var resp searchResponse
	if err := c.getJSON(ctx, c.searchLimiter, searchURL+url.QueryEscape(string(qp)), &resp); err != nil {
		return nil, 0, fmt.Errorf("do request, %w", err)
	}

	groups := make([]Group, 0, len(resp.Data.Results))
	for _, v := range resp.Data.Results {
		if v.AssetDescription.GroupID == "" {
			continue
		}
		g := Group{
			ID:           v.AssetDescription.GroupID,
			Name:         v.AssetDescription.GroupHashName,
			SellListings: v.SellListings,
		}
		price, err := parseSteamPrice(v.MinSellSubtotal)
		switch {
		case err != nil:
			log.Printf("steam group: %s: %v", g.ID, err)
		case v.Currency != currencyUSD:
			log.Printf("steam search: %s: currency %d, want USD", g.ID, v.Currency)
		default:
			g.MinPrice = &price
		}

		groups = append(groups, g)
	}

	return groups, resp.Data.TotalCount, nil
}
