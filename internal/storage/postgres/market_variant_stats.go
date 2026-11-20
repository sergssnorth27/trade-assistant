package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sergssnorth27/trade-assistant/internal/marketstats"
)

const upsertMarketVariantStats = `
INSERT INTO market_variant_stats (
    -- служебное
    hash_name,
    market_id,
    computed_at,
    -- price_
    price_median_24h,
    price_median_7d,
    price_median_30d,
    price_median_90d,
    price_p25_24h,
    price_p25_7d,
    price_p25_30d,
    price_p25_90d,
    -- liq_
    liq_sales_24h,
    liq_sales_7d,
    liq_sales_30d,
    liq_sales_90d,
    -- q_
    q_oldest_sale_at,
    q_last_sale_at
) VALUES (
    -- служебное
    @hash_name,
    @market_id,
    @computed_at,
    -- price_
    @price_median_24h,
    @price_median_7d,
    @price_median_30d,
    @price_median_90d,
    @price_p25_24h,
    @price_p25_7d,
    @price_p25_30d,
    @price_p25_90d,
    -- liq_
    @liq_sales_24h,
    @liq_sales_7d,
    @liq_sales_30d,
    @liq_sales_90d,
    -- q_
    @q_oldest_sale_at,
    @q_last_sale_at
)
ON CONFLICT (hash_name) DO UPDATE SET
    -- служебное
    market_id        = EXCLUDED.market_id,
    computed_at      = EXCLUDED.computed_at,
    -- price_
    price_median_24h = EXCLUDED.price_median_24h,
    price_median_7d  = EXCLUDED.price_median_7d,
    price_median_30d = EXCLUDED.price_median_30d,
    price_median_90d = EXCLUDED.price_median_90d,
    price_p25_24h    = EXCLUDED.price_p25_24h,
    price_p25_7d     = EXCLUDED.price_p25_7d,
    price_p25_30d    = EXCLUDED.price_p25_30d,
    price_p25_90d    = EXCLUDED.price_p25_90d,
    -- liq_
    liq_sales_24h    = EXCLUDED.liq_sales_24h,
    liq_sales_7d     = EXCLUDED.liq_sales_7d,
    liq_sales_30d    = EXCLUDED.liq_sales_30d,
    liq_sales_90d    = EXCLUDED.liq_sales_90d,
    -- q_
    q_oldest_sale_at = EXCLUDED.q_oldest_sale_at,
    q_last_sale_at   = EXCLUDED.q_last_sale_at`

type MarketVariantStatsRepo struct {
	pool *pgxpool.Pool
}

func NewMarketVariantStatsRepo(pool *pgxpool.Pool) *MarketVariantStatsRepo {
	return &MarketVariantStatsRepo{pool: pool}
}

// Upsert записывает статистику пачкой: один pgx.Batch, всё или ничего.
func (r *MarketVariantStatsRepo) Upsert(ctx context.Context, stats []marketstats.MarketStats) error {
	if len(stats) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	for _, s := range stats {
		batch.Queue(upsertMarketVariantStats, marketStatsArgs(s))
	}

	if err := r.pool.SendBatch(ctx, batch).Close(); err != nil {
		return fmt.Errorf("upsert market variant stats (%d rows): %w", len(stats), err)
	}
	return nil
}

// marketStatsArgs раскладывает MarketStats по именованным параметрам upsertMarketVariantStats.
func marketStatsArgs(s marketstats.MarketStats) pgx.NamedArgs {
	return pgx.NamedArgs{
		// служебное
		"hash_name":   s.HashName,
		"market_id":   s.MarketID,
		"computed_at": s.ComputedAt,

		// price_ (priceArg — из steam_variant_stats.go, *domain.Price → *int64)
		"price_median_24h": priceArg(s.W24h.Median),
		"price_median_7d":  priceArg(s.W7d.Median),
		"price_median_30d": priceArg(s.W30d.Median),
		"price_median_90d": priceArg(s.W90d.Median),
		"price_p25_24h":    priceArg(s.W24h.P25),
		"price_p25_7d":     priceArg(s.W7d.P25),
		"price_p25_30d":    priceArg(s.W30d.P25),
		"price_p25_90d":    priceArg(s.W90d.P25),

		// liq_
		"liq_sales_24h": s.W24h.Sales,
		"liq_sales_7d":  s.W7d.Sales,
		"liq_sales_30d": s.W30d.Sales,
		"liq_sales_90d": s.W90d.Sales,

		// q_ (*time.Time: nil → NULL)
		"q_oldest_sale_at": s.OldestSaleAt,
		"q_last_sale_at":   s.LastSaleAt,
	}
}
