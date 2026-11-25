package steam

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	"github.com/sergssnorth27/trade-assistant/internal/domain"
)

const (
	orderBookURL = "https://steamcommunity.com/market/orderbook?q=Load&qp="
)

var ErrNotUSD = errors.New("currency not usd")

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
	MaxBuyOrder   domain.Price
	MinSellOrder  domain.Price
	Ecurrency     int64
	QtyBuyOrders  int64
	QtySellOrders int64
	BuyOrders     []OrderLevel
	SellOrders    []OrderLevel
}

type OrderLevel struct {
	Price domain.Price
	Count int64
}

func priceOrderToDomain(in int64) domain.Price {
	return domain.Price(in * 10)
}

func (c *Client) OrderBook(ctx context.Context, name string) (OrderBook, error) {
	qp, err := json.Marshal([]any{appCS2, name})
	if err != nil {
		return OrderBook{}, fmt.Errorf("marshal json: %w", err)
	}

	var resp OrderBookResponse
	if err := c.getJSON(ctx, c.searchLimiter, orderBookURL+url.QueryEscape(string(qp)), &resp); err != nil {
		return OrderBook{}, fmt.Errorf("do request, %w", err)
	}

	if !resp.Data.Success {
		return OrderBook{}, fmt.Errorf("data succes: %v", resp.Data.Success)
	}
	if resp.Data.Data.ECurrency != currencyUSD {
		return OrderBook{}, fmt.Errorf("currency: %w", ErrNotUSD)
	}

	listBuyOrders := resp.Data.Data.RgCompactBuyOrders
	listSellOrders := resp.Data.Data.RgCompactSellOrders

	if len(listBuyOrders)%2 != 0 {
		return OrderBook{}, fmt.Errorf("no correct len listBuyOrders: %v", len(listBuyOrders))
	}
	if len(listSellOrders)%2 != 0 {
		return OrderBook{}, fmt.Errorf("no correct len listSellOrders: %v", len(listSellOrders))
	}

	buyOrders := make([]OrderLevel, 0, len(listBuyOrders)/2)
	sellOrders := make([]OrderLevel, 0, len(listSellOrders)/2)

	for i := 0; i+1 < len(listBuyOrders); i += 2 {
		buyOrders = append(buyOrders, OrderLevel{
			Price: priceOrderToDomain(listBuyOrders[i]),
			Count: listBuyOrders[i+1],
		})
	}
	for i := 0; i+1 < len(listSellOrders); i += 2 {
		sellOrders = append(sellOrders, OrderLevel{
			Price: priceOrderToDomain(listSellOrders[i]),
			Count: listSellOrders[i+1],
		})
	}

	orderbook := OrderBook{
		MaxBuyOrder:   priceOrderToDomain(resp.Data.Data.AmtMaxBuyOrder),
		MinSellOrder:  priceOrderToDomain(resp.Data.Data.AmtMinSellOrder),
		Ecurrency:     resp.Data.Data.ECurrency,
		QtyBuyOrders:  resp.Data.Data.CBuyOrders,
		QtySellOrders: resp.Data.Data.CSellOrders,
		BuyOrders:     buyOrders,
		SellOrders:    sellOrders,
	}

	return orderbook, nil
}

func (b *OrderBook) Trim(maxDepth float64, maxSellLevels int) {
	minPrice := domain.Price(float64(b.MaxBuyOrder) * (1.0 - maxDepth))
	for i, order := range b.BuyOrders {
		if order.Price < minPrice {
			b.BuyOrders = b.BuyOrders[:i]
			break
		}
	}
	if len(b.SellOrders) >= maxSellLevels {
		b.SellOrders = b.SellOrders[:maxSellLevels]
	}
}
