package steam

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
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
		} `json:"results"`
		TotalCount int `json:"total_count"`
	} `json:"data"`
}

type Group struct {
	ID   string
	Name string
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
		groups = append(groups, Group{ID: v.AssetDescription.GroupID, Name: v.AssetDescription.GroupHashName})
	}

	return groups, resp.Data.TotalCount, nil
}
