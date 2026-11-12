package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sergssnorth27/trade-assistant/internal/domain"
	"github.com/sergssnorth27/trade-assistant/internal/steamstats"
)

const upsertVariantStats = `
INSERT INTO steam_variant_stats (
    -- служебное
    hash_name,
    group_id,
    computed_at,
    last_point_at,
    source_currency,
    usd_rate,
    points_count,
    -- price_: окна
    price_median_24h,
    price_p25_24h,
    price_p75_24h,
    price_median_3d,
    price_p25_3d,
    price_p75_3d,
    price_median_7d,
    price_p25_7d,
    price_p75_7d,
    price_median_30d,
    price_p25_30d,
    price_p75_30d,
    price_median_60d,
    price_p25_60d,
    price_p75_60d,
    price_median_90d,
    price_p25_90d,
    price_p75_90d,
    -- price_: коридор дневных медиан
    price_min_daily_7d,
    price_min_daily_30d,
    price_min_daily_60d,
    price_min_daily_90d,
    price_max_daily_7d,
    price_max_daily_30d,
    price_max_daily_60d,
    price_max_daily_90d,
    -- price_: отношения, тренд, ATH/ATL, лот
    price_ratio_24h_7d,
    price_ratio_3d_7d,
    price_ratio_7d_30d,
    price_ratio_30d_90d,
    price_trend_7d,
    price_trend_30d,
    price_trend_r2_7d,
    price_trend_r2_30d,
    price_ath,
    price_ath_at,
    price_atl,
    price_atl_at,
    price_min_listing,
    -- liq_
    liq_sales_24h,
    liq_sales_3d,
    liq_sales_7d,
    liq_sales_30d,
    liq_sales_60d,
    liq_sales_90d,
    liq_sales_per_day_7d,
    liq_sales_per_day_30d,
    liq_days_with_sales_7d,
    liq_days_with_sales_30d,
    liq_top_day_share_7d,
    liq_volume_trend_7d,
    -- risk_
    risk_fwd_p10,
    risk_fwd_p50,
    risk_fwd_days,
    risk_fwd_samples,
    risk_volatility_7d,
    risk_volatility_30d,
    risk_max_drawdown_90d,
    -- q_
    q_first_sale_at,
    q_age_days,
    q_last_sale_at,
    q_last_sale_price,
    q_mixed_phases
) VALUES (
    -- служебное
    @hash_name,
    @group_id,
    @computed_at,
    @last_point_at,
    @source_currency,
    @usd_rate,
    @points_count,
    -- price_: окна
    @price_median_24h,
    @price_p25_24h,
    @price_p75_24h,
    @price_median_3d,
    @price_p25_3d,
    @price_p75_3d,
    @price_median_7d,
    @price_p25_7d,
    @price_p75_7d,
    @price_median_30d,
    @price_p25_30d,
    @price_p75_30d,
    @price_median_60d,
    @price_p25_60d,
    @price_p75_60d,
    @price_median_90d,
    @price_p25_90d,
    @price_p75_90d,
    -- price_: коридор дневных медиан
    @price_min_daily_7d,
    @price_min_daily_30d,
    @price_min_daily_60d,
    @price_min_daily_90d,
    @price_max_daily_7d,
    @price_max_daily_30d,
    @price_max_daily_60d,
    @price_max_daily_90d,
    -- price_: отношения, тренд, ATH/ATL, лот
    @price_ratio_24h_7d,
    @price_ratio_3d_7d,
    @price_ratio_7d_30d,
    @price_ratio_30d_90d,
    @price_trend_7d,
    @price_trend_30d,
    @price_trend_r2_7d,
    @price_trend_r2_30d,
    @price_ath,
    @price_ath_at,
    @price_atl,
    @price_atl_at,
    @price_min_listing,
    -- liq_
    @liq_sales_24h,
    @liq_sales_3d,
    @liq_sales_7d,
    @liq_sales_30d,
    @liq_sales_60d,
    @liq_sales_90d,
    @liq_sales_per_day_7d,
    @liq_sales_per_day_30d,
    @liq_days_with_sales_7d,
    @liq_days_with_sales_30d,
    @liq_top_day_share_7d,
    @liq_volume_trend_7d,
    -- risk_
    @risk_fwd_p10,
    @risk_fwd_p50,
    @risk_fwd_days,
    @risk_fwd_samples,
    @risk_volatility_7d,
    @risk_volatility_30d,
    @risk_max_drawdown_90d,
    -- q_
    @q_first_sale_at,
    @q_age_days,
    @q_last_sale_at,
    @q_last_sale_price,
    @q_mixed_phases
)
ON CONFLICT (hash_name) DO UPDATE SET
    -- служебное
    group_id                = EXCLUDED.group_id,
    computed_at             = EXCLUDED.computed_at,
    last_point_at           = EXCLUDED.last_point_at,
    source_currency         = EXCLUDED.source_currency,
    usd_rate                = EXCLUDED.usd_rate,
    points_count            = EXCLUDED.points_count,
    -- price_: окна
    price_median_24h        = EXCLUDED.price_median_24h,
    price_p25_24h           = EXCLUDED.price_p25_24h,
    price_p75_24h           = EXCLUDED.price_p75_24h,
    price_median_3d         = EXCLUDED.price_median_3d,
    price_p25_3d            = EXCLUDED.price_p25_3d,
    price_p75_3d            = EXCLUDED.price_p75_3d,
    price_median_7d         = EXCLUDED.price_median_7d,
    price_p25_7d            = EXCLUDED.price_p25_7d,
    price_p75_7d            = EXCLUDED.price_p75_7d,
    price_median_30d        = EXCLUDED.price_median_30d,
    price_p25_30d           = EXCLUDED.price_p25_30d,
    price_p75_30d           = EXCLUDED.price_p75_30d,
    price_median_60d        = EXCLUDED.price_median_60d,
    price_p25_60d           = EXCLUDED.price_p25_60d,
    price_p75_60d           = EXCLUDED.price_p75_60d,
    price_median_90d        = EXCLUDED.price_median_90d,
    price_p25_90d           = EXCLUDED.price_p25_90d,
    price_p75_90d           = EXCLUDED.price_p75_90d,
    -- price_: коридор дневных медиан
    price_min_daily_7d      = EXCLUDED.price_min_daily_7d,
    price_min_daily_30d     = EXCLUDED.price_min_daily_30d,
    price_min_daily_60d     = EXCLUDED.price_min_daily_60d,
    price_min_daily_90d     = EXCLUDED.price_min_daily_90d,
    price_max_daily_7d      = EXCLUDED.price_max_daily_7d,
    price_max_daily_30d     = EXCLUDED.price_max_daily_30d,
    price_max_daily_60d     = EXCLUDED.price_max_daily_60d,
    price_max_daily_90d     = EXCLUDED.price_max_daily_90d,
    -- price_: отношения, тренд, ATH/ATL, лот
    price_ratio_24h_7d      = EXCLUDED.price_ratio_24h_7d,
    price_ratio_3d_7d       = EXCLUDED.price_ratio_3d_7d,
    price_ratio_7d_30d      = EXCLUDED.price_ratio_7d_30d,
    price_ratio_30d_90d     = EXCLUDED.price_ratio_30d_90d,
    price_trend_7d          = EXCLUDED.price_trend_7d,
    price_trend_30d         = EXCLUDED.price_trend_30d,
    price_trend_r2_7d       = EXCLUDED.price_trend_r2_7d,
    price_trend_r2_30d      = EXCLUDED.price_trend_r2_30d,
    price_ath               = EXCLUDED.price_ath,
    price_ath_at            = EXCLUDED.price_ath_at,
    price_atl               = EXCLUDED.price_atl,
    price_atl_at            = EXCLUDED.price_atl_at,
    price_min_listing       = EXCLUDED.price_min_listing,
    -- liq_
    liq_sales_24h           = EXCLUDED.liq_sales_24h,
    liq_sales_3d            = EXCLUDED.liq_sales_3d,
    liq_sales_7d            = EXCLUDED.liq_sales_7d,
    liq_sales_30d           = EXCLUDED.liq_sales_30d,
    liq_sales_60d           = EXCLUDED.liq_sales_60d,
    liq_sales_90d           = EXCLUDED.liq_sales_90d,
    liq_sales_per_day_7d    = EXCLUDED.liq_sales_per_day_7d,
    liq_sales_per_day_30d   = EXCLUDED.liq_sales_per_day_30d,
    liq_days_with_sales_7d  = EXCLUDED.liq_days_with_sales_7d,
    liq_days_with_sales_30d = EXCLUDED.liq_days_with_sales_30d,
    liq_top_day_share_7d    = EXCLUDED.liq_top_day_share_7d,
    liq_volume_trend_7d     = EXCLUDED.liq_volume_trend_7d,
    -- risk_
    risk_fwd_p10            = EXCLUDED.risk_fwd_p10,
    risk_fwd_p50            = EXCLUDED.risk_fwd_p50,
    risk_fwd_days           = EXCLUDED.risk_fwd_days,
    risk_fwd_samples        = EXCLUDED.risk_fwd_samples,
    risk_volatility_7d      = EXCLUDED.risk_volatility_7d,
    risk_volatility_30d     = EXCLUDED.risk_volatility_30d,
    risk_max_drawdown_90d   = EXCLUDED.risk_max_drawdown_90d,
    -- q_
    q_first_sale_at         = EXCLUDED.q_first_sale_at,
    q_age_days              = EXCLUDED.q_age_days,
    q_last_sale_at          = EXCLUDED.q_last_sale_at,
    q_last_sale_price       = EXCLUDED.q_last_sale_price,
    q_mixed_phases          = EXCLUDED.q_mixed_phases`

type SteamVariantStatsRepo struct {
	pool *pgxpool.Pool
}

func NewSteamVariantStatsRepo(pool *pgxpool.Pool) *SteamVariantStatsRepo {
	return &SteamVariantStatsRepo{pool: pool}
}

func priceArg(p *domain.Price) *int64 {
	if p == nil {
		return nil
	}
	v := int64(*p)
	return &v
}

// variantStatsArgs раскладывает VariantStats по именованным параметрам запроса upsertVariantStats.
func variantStatsArgs(s steamstats.VariantStats) pgx.NamedArgs {
	return pgx.NamedArgs{
		// служебное
		"hash_name":       s.HashName,
		"group_id":        s.GroupID,
		"computed_at":     s.ComputedAt,
		"last_point_at":   s.LastPointAt,
		"source_currency": s.SourceCurrency,
		"usd_rate":        s.UsdRate,
		"points_count":    s.PointsCount,

		// price_: окна
		"price_median_24h": priceArg(s.Price.W24h.Median),
		"price_p25_24h":    priceArg(s.Price.W24h.P25),
		"price_p75_24h":    priceArg(s.Price.W24h.P75),
		"price_median_3d":  priceArg(s.Price.W3d.Median),
		"price_p25_3d":     priceArg(s.Price.W3d.P25),
		"price_p75_3d":     priceArg(s.Price.W3d.P75),
		"price_median_7d":  priceArg(s.Price.W7d.Median),
		"price_p25_7d":     priceArg(s.Price.W7d.P25),
		"price_p75_7d":     priceArg(s.Price.W7d.P75),
		"price_median_30d": priceArg(s.Price.W30d.Median),
		"price_p25_30d":    priceArg(s.Price.W30d.P25),
		"price_p75_30d":    priceArg(s.Price.W30d.P75),
		"price_median_60d": priceArg(s.Price.W60d.Median),
		"price_p25_60d":    priceArg(s.Price.W60d.P25),
		"price_p75_60d":    priceArg(s.Price.W60d.P75),
		"price_median_90d": priceArg(s.Price.W90d.Median),
		"price_p25_90d":    priceArg(s.Price.W90d.P25),
		"price_p75_90d":    priceArg(s.Price.W90d.P75),

		// price_: коридор дневных медиан
		"price_min_daily_7d":  priceArg(s.Price.MinDaily7d),
		"price_min_daily_30d": priceArg(s.Price.MinDaily30d),
		"price_min_daily_60d": priceArg(s.Price.MinDaily60d),
		"price_min_daily_90d": priceArg(s.Price.MinDaily90d),
		"price_max_daily_7d":  priceArg(s.Price.MaxDaily7d),
		"price_max_daily_30d": priceArg(s.Price.MaxDaily30d),
		"price_max_daily_60d": priceArg(s.Price.MaxDaily60d),
		"price_max_daily_90d": priceArg(s.Price.MaxDaily90d),

		// price_: отношения, тренд, ATH/ATL, лот
		"price_ratio_24h_7d":  s.Price.Ratio24h7d,
		"price_ratio_3d_7d":   s.Price.Ratio3d7d,
		"price_ratio_7d_30d":  s.Price.Ratio7d30d,
		"price_ratio_30d_90d": s.Price.Ratio30d90d,
		"price_trend_7d":      s.Price.Trend7d,
		"price_trend_30d":     s.Price.Trend30d,
		"price_trend_r2_7d":   s.Price.TrendR2_7d,
		"price_trend_r2_30d":  s.Price.TrendR2_30d,
		"price_ath":           priceArg(s.Price.ATH),
		"price_ath_at":        s.Price.ATHAt,
		"price_atl":           priceArg(s.Price.ATL),
		"price_atl_at":        s.Price.ATLAt,
		"price_min_listing":   priceArg(s.Price.MinListing),

		// liq_
		"liq_sales_24h":           s.Liquidity.Sales24h,
		"liq_sales_3d":            s.Liquidity.Sales3d,
		"liq_sales_7d":            s.Liquidity.Sales7d,
		"liq_sales_30d":           s.Liquidity.Sales30d,
		"liq_sales_60d":           s.Liquidity.Sales60d,
		"liq_sales_90d":           s.Liquidity.Sales90d,
		"liq_sales_per_day_7d":    s.Liquidity.SalesPerDay7d,
		"liq_sales_per_day_30d":   s.Liquidity.SalesPerDay30d,
		"liq_days_with_sales_7d":  s.Liquidity.DaysWithSales7d,
		"liq_days_with_sales_30d": s.Liquidity.DaysWithSales30d,
		"liq_top_day_share_7d":    s.Liquidity.TopDayShare7d,
		"liq_volume_trend_7d":     s.Liquidity.VolumeTrend7d,

		// risk_
		"risk_fwd_p10":          s.Risk.FwdP10,
		"risk_fwd_p50":          s.Risk.FwdP50,
		"risk_fwd_days":         s.Risk.FwdDays,
		"risk_fwd_samples":      s.Risk.FwdSamples,
		"risk_volatility_7d":    s.Risk.Volatility7d,
		"risk_volatility_30d":   s.Risk.Volatility30d,
		"risk_max_drawdown_90d": s.Risk.MaxDrawdown90d,

		// q_
		"q_first_sale_at":   s.Quality.FirstSaleAt,
		"q_age_days":        s.Quality.AgeDays,
		"q_last_sale_at":    s.Quality.LastSaleAt,
		"q_last_sale_price": priceArg(s.Quality.LastSalePrice),
		"q_mixed_phases":    s.Quality.MixedPhases,
	}
}

// Upsert записывает статистику всех вариантов одной группы одним батчем.
func (r *SteamVariantStatsRepo) Upsert(ctx context.Context, stats []steamstats.VariantStats) error {
	batch := &pgx.Batch{}
	for _, s := range stats {
		batch.Queue(upsertVariantStats, variantStatsArgs(s))
	}

	return r.pool.SendBatch(ctx, batch).Close()
}
