package steamstats

import (
	"time"

	"github.com/sergssnorth27/trade-assistant/internal/domain"
)

// Config — параметры расчёта статистики.
type Config struct {
	HorizonDays int // через сколько дней планируем продать (H), сейчас 7
}

// VariantStats — вся статистика Steam по одному варианту предмета.
// Одна структура = одна строка таблицы steam_variant_stats.
// Указатель nil = NULL в БД = «нет данных».
type VariantStats struct {
	HashName       string     // hash_name
	GroupID        string     // group_id
	ComputedAt     time.Time  // computed_at
	LastPointAt    *time.Time // last_point_at
	SourceCurrency int        // source_currency: 1 — USD, 3 — EUR, 5 — RUB
	PointsCount    int        // points_count

	Price     PriceStats
	Liquidity LiquidityStats
	Risk      RiskStats
	Quality   QualityStats
}

// PriceWindow — цены продаж за одно окно.
type PriceWindow struct {
	Median *domain.Price // price_median_{окно}
	P25    *domain.Price // price_p25_{окно}
	P75    *domain.Price // price_p75_{окно}
}

// PriceStats — за сколько продам (колонки price_*).
type PriceStats struct {
	W24h PriceWindow
	W3d  PriceWindow
	W7d  PriceWindow
	W30d PriceWindow
	W60d PriceWindow
	W90d PriceWindow

	MinDaily7d  *domain.Price // price_min_daily_7d
	MinDaily30d *domain.Price // price_min_daily_30d
	MinDaily60d *domain.Price // price_min_daily_60d
	MinDaily90d *domain.Price // price_min_daily_90d
	MaxDaily7d  *domain.Price // price_max_daily_7d
	MaxDaily30d *domain.Price // price_max_daily_30d
	MaxDaily60d *domain.Price // price_max_daily_60d
	MaxDaily90d *domain.Price // price_max_daily_90d

	Ratio24h7d  *float64 // price_ratio_24h_7d
	Ratio3d7d   *float64 // price_ratio_3d_7d
	Ratio7d30d  *float64 // price_ratio_7d_30d
	Ratio30d90d *float64 // price_ratio_30d_90d

	Trend7d     *float64 // price_trend_7d: изменение в день, −0.01 = −1%
	Trend30d    *float64 // price_trend_30d
	TrendR2_7d  *float64 // price_trend_r2_7d
	TrendR2_30d *float64 // price_trend_r2_30d

	ATH   *domain.Price // price_ath
	ATHAt *time.Time    // price_ath_at
	ATL   *domain.Price // price_atl
	ATLAt *time.Time    // price_atl_at

	MinListing *domain.Price // price_min_listing (справочно)
}

// LiquidityStats — как быстро продам (колонки liq_*).
// Sales без указателя: 0 продаж — это факт, а не «нет данных».
type LiquidityStats struct {
	Sales24h int // liq_sales_24h
	Sales3d  int // liq_sales_3d
	Sales7d  int // liq_sales_7d
	Sales30d int // liq_sales_30d
	Sales60d int // liq_sales_60d
	Sales90d int // liq_sales_90d

	SalesPerDay7d  float64 // liq_sales_per_day_7d
	SalesPerDay30d float64 // liq_sales_per_day_30d

	DaysWithSales7d  int // liq_days_with_sales_7d
	DaysWithSales30d int // liq_days_with_sales_30d

	TopDayShare7d *float64 // liq_top_day_share_7d
	VolumeTrend7d *float64 // liq_volume_trend_7d
}

// RiskStats — насколько рискованно (колонки risk_*).
type RiskStats struct {
	FwdP10     *float64 // risk_fwd_p10: плохой сценарий за H дней
	FwdP50     *float64 // risk_fwd_p50: типичное изменение за H дней
	FwdDays    int      // risk_fwd_days: какое H использовали
	FwdSamples int      // risk_fwd_samples

	Volatility7d   *float64 // risk_volatility_7d
	Volatility30d  *float64 // risk_volatility_30d
	MaxDrawdown90d *float64 // risk_max_drawdown_90d
}

// QualityStats — можно ли верить данным (колонки q_*).
type QualityStats struct {
	FirstSaleAt   *time.Time    // q_first_sale_at
	AgeDays       *int          // q_age_days
	LastSaleAt    *time.Time    // q_last_sale_at
	LastSalePrice *domain.Price // q_last_sale_price
	MixedPhases   bool          // q_mixed_phases
}

// ptr возвращает указатель на копию значения: ptr(0.5) → *float64.
// Нужен, потому что в Go нельзя взять адрес литерала или результата функции: &0.5 не компилируется.
func ptr[T any](v T) *T {
	return &v
}
