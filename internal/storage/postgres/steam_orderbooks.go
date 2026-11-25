package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sergssnorth27/trade-assistant/internal/steam"
)

const upsertQuery = `
	INSERT INTO steam_orderbooks
	(hash_name, fetched_at, max_buy_order, min_sell_order, buy_orders_total, sell_orders_total, buy_levels, sell_levels)
	VALUES ($1, $2, NULLIF($3, 0), NULLIF($4, 0), $5, $6, $7, $8)
	ON CONFLICT (hash_name) DO UPDATE SET
	    fetched_at        = EXCLUDED.fetched_at,
	    max_buy_order     = EXCLUDED.max_buy_order,
	    min_sell_order    = EXCLUDED.min_sell_order,
	    buy_orders_total  = EXCLUDED.buy_orders_total,
	    sell_orders_total = EXCLUDED.sell_orders_total,
	    buy_levels        = EXCLUDED.buy_levels,
	    sell_levels       = EXCLUDED.sell_levels`

type SteamOrderBooksRepo struct {
	pool *pgxpool.Pool
}

func NewSteamOrderBooksRepo(pool *pgxpool.Pool) *SteamOrderBooksRepo {
	return &SteamOrderBooksRepo{pool: pool}
}

func flatten(levels []steam.OrderLevel) []int64 {
	var result = make([]int64, 0, len(levels)*2)
	for _, v := range levels {
		result = append(result, int64(v.Price))
		result = append(result, v.Count)
	}
	return result
}

func (r *SteamOrderBooksRepo) Upsert(ctx context.Context, hashName string, book steam.OrderBook, fetchedAt time.Time) error {
	_, err := r.pool.Exec(ctx, upsertQuery,
		hashName,
		fetchedAt,
		book.MaxBuyOrder,
		book.MinSellOrder,
		book.QtyBuyOrders,
		book.QtySellOrders,
		flatten(book.BuyOrders),
		flatten(book.SellOrders),
	)
	if err != nil {
		return fmt.Errorf("upsert orderbook %s: %v err %w", hashName, book, err)
	}
	return nil
}
