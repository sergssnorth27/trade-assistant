package steam

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/sergssnorth27/trade-assistant/internal/domain"
)

const (
	orderBookURL = "https://steamcommunity.com/market/orderbook?q=Load&qp="
)

type OrderBookResponse struct {
	Data struct {
		Success bool `json:"success"`
		Data    struct {
			AmtMaxBuyOrder      int64   `json:"amtMaxBuyOrder"`
			AmtMinSellOrder     int64   `json:"amtMinSellOrder"`
			ECurrency           int64   `json:"eCurrency"`
			CBuyOrders          int64   `json:"cBuyOrders"`
			CSellOrders         int64   `json:"cSellOrders"`
			RgCompactBuyOrders  []int64 `json:"rgCompactBuyOrders"`
			RgCompactSellOrders []int64 `json:"rgCompactSellOrders"`
		} `json:"data"`
	} `json:"data"`
}

type OrderBook struct {
	MaxBuyOrder domain.Price
}

type Order struct {
	Price domain.Price
	Count int64
}

func priceOrderToDomain(in int64) domain.Price {
	return domain.Price(in * 10)
}

func (c *Client) OrderBookPage(ctx context.Context, name string) ([]OrderBook, int, error) {
	qp, err := json.Marshal([]any{appCS2, name})
	if err != nil {
		return nil, 0, fmt.Errorf("marshal json: %w", err)
	}

	var resp OrderBookResponse
	if err := c.getJSON(ctx, c.searchLimiter, orderBookURL+url.QueryEscape(string(qp)), &resp); err != nil {
		return nil, 0, fmt.Errorf("do request, %w", err)
	}

	var buyOrders []Order
	var sellOrders []Order
	for i, v := range resp.Data.Data.RgCompactBuyOrders {
		if i == 0 || i%2 == 0 {
			countIdx := i + 1
			buyOrders = append(buyOrders, Order{
				Price: priceOrderToDomain(v),
				Count: resp.Data.Data.RgCompactBuyOrders[countIdx],
			})
		}
	}
	for i, v := range resp.Data.Data.RgCompactSellOrders {
		if i == 0 || i%2 == 0 {
			countIdx := i + 1
			sellOrders = append(sellOrders, Order{
				Price: priceOrderToDomain(v),
				Count: resp.Data.Data.RgCompactSellOrders[countIdx],
			})
		}
	}

	return nil, 0, nil
}
