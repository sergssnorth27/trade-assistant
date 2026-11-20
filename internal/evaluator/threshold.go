package evaluator

import (
	"math"
	"time"

	"github.com/sergssnorth27/trade-assistant/internal/domain"
	"github.com/sergssnorth27/trade-assistant/internal/pricing"
	"github.com/sergssnorth27/trade-assistant/internal/steamstats"
)

// Config — пороги и параметры оценки. В комментарии у каждого поля — стартовое значение.
type Config struct {
	TargetProfit float64 // Сколько хотим заработать от вложенного: 0.30 = купили за $10, получили после комиссии ≥ $13. Старт: 0.30.
	HoldDays     int     // Сколько дней пройдёт от покупки на market.csgo до продажи в Steam (trade hold).

	// Свежесть данных
	MaxStatsAge    time.Duration // Статистика Steam старше этого считается устаревшей — по ней не торгуем. Старт: 24h.
	MaxLastSaleAge time.Duration // Если последняя продажа в Steam была раньше этого — предмет «умер», продать не выйдет. Старт: 48h.

	// Ликвидность: сможем ли продать
	MinSellPrice        domain.Price // Дешевле этого не берём: минимальная комиссия Steam (1 цент) съедает прибыль. Старт: $0.20.
	MinSales7d          int          // Минимум продаж за 7 дней. Меньше — медиане нельзя верить и продать трудно. Старт: 35 (≈ 5 в день).
	MinSales24hReliable int          // Медиане за 24h верим (в цене продажи и в стоп-сигнале), только если за сутки было не меньше продаж. Меньше — пара сделок по цене заявки даёт ложную картину. Старт: 10.
	MinDaysWithSales7d  int          // Минимум дней с продажами из последних 7. Отсекает предметы, которые продаются рывками. Старт: 6.
	MaxTopDayShare      float64      // Максимальная доля продаж недели, пришедшаяся на один день. Больше — объём держится на одном всплеске. Старт: 0.5.

	// Падение цены
	DropThreshold float64 // Стоп-сигнал: если медиана за 24h или 3d ниже медианы за 7d больше чем на эту долю — цена падает прямо сейчас, не покупаем. 0.10 = упала на 10%+. Старт: 0.10.
	MinTrendR2    float64 // Минимальный R² тренда за 7 дней, чтобы делать поправку на падение. Ниже — точки разбросаны, тренду не верим. Старт: 0.3.

	// Риск по истории
	MinFwdSamples int     // Минимум примеров «купил → продал через H дней», чтобы доверять risk_fwd_p10. Меньше — проверку риска пропускаем. Старт: 20.
	MaxFwdLoss    float64 // Отказ, если в худших 10% случаев цена за H дней падала сильнее этой доли. 0.20 = падение на 20%+. Старт: 0.20.

	// Количество
	MaxQtyPerItem  int     // Больше этого числа штук одного предмета не берём, даже если лотов и спроса хватает. Старт: 3.
	MaxMarketShare float64 // Какую долю продаж Steam за H дней можно занять своими лотами, не сбивая цену. 0.05 = если за неделю продают 100 штук, берём не больше 5. Старт: 0.05.
}

// Threshold — до какой цены предмет выгодно покупать.
type Threshold struct {
	MaxBuy    domain.Price
	SellPrice domain.Price
	Receive   domain.Price
	QtyCap    int
}

// RejectReason — почему предмет не подходит. Пустая строка — подходит.
type RejectReason string

const (
	RejectStale       RejectReason = "stale_stats"
	RejectMixedPhases RejectReason = "mixed_phases"
	RejectNoSales     RejectReason = "no_sales"
	RejectIrregular   RejectReason = "irregular"
	RejectDead        RejectReason = "dead"
	RejectFalling     RejectReason = "falling_now"
	RejectRisky       RejectReason = "risky"
	RejectTooCheap    RejectReason = "too_cheap"
)

// DefaultConfig возвращает стартовые значения порогов из evaluator-v1.md.
// Их подбирают по журналу бумажной торговли.
func DefaultConfig() Config {
	return Config{
		TargetProfit: 0.25,
		HoldDays:     7,

		MaxStatsAge:    24 * time.Hour,
		MaxLastSaleAge: 48 * time.Hour,

		MinSellPrice:        400,
		MinSales7d:          35,
		MinDaysWithSales7d:  6,
		MaxTopDayShare:      0.5,
		MinSales24hReliable: 10,

		DropThreshold: 0.10,
		MinTrendR2:    0.3,

		MinFwdSamples: 20,
		MaxFwdLoss:    0.20,

		MaxQtyPerItem:  3,
		MaxMarketShare: 0.05,
	}
}

// Threshold считает порог покупки по статистике Steam. Если предмет не подходит — пустой Threshold и причина.
func ComputeThreshold(s *steamstats.VariantStats, now time.Time, cfg Config) (Threshold, RejectReason) {
	if now.Sub(s.ComputedAt) > cfg.MaxStatsAge {
		return Threshold{}, RejectStale
	}
	if s.Quality.MixedPhases {
		return Threshold{}, RejectMixedPhases
	}
	if s.Price.W7d.Median == nil || s.Liquidity.Sales7d < cfg.MinSales7d {
		return Threshold{}, RejectNoSales
	}

	if s.Liquidity.DaysWithSales7d < cfg.MinDaysWithSales7d ||
		s.Liquidity.TopDayShare7d == nil || *s.Liquidity.TopDayShare7d > cfg.MaxTopDayShare {
		return Threshold{}, RejectIrregular
	}
	if s.Quality.LastSaleAt == nil || now.Sub(*s.Quality.LastSaleAt) > cfg.MaxLastSaleAge {
		return Threshold{}, RejectDead
	}
	drop := 1 - cfg.DropThreshold

	falling24h := s.Liquidity.Sales24h >= cfg.MinSales24hReliable &&
		s.Price.Ratio24h7d != nil && *s.Price.Ratio24h7d < drop
	falling3d := s.Price.Ratio3d7d != nil && *s.Price.Ratio3d7d < drop

	if falling24h || falling3d {
		return Threshold{}, RejectFalling
	}

	if s.Risk.FwdSamples >= cfg.MinFwdSamples && s.Risk.FwdP10 != nil && *s.Risk.FwdP10 < -cfg.MaxFwdLoss {
		return Threshold{}, RejectRisky
	}

	sell := float64(*s.Price.W7d.Median)
	if s.Price.W24h.Median != nil && s.Liquidity.Sales24h >= cfg.MinSales24hReliable {
		sell = min(sell, float64(*s.Price.W24h.Median))
	}

	if s.Price.Trend7d != nil && *s.Price.Trend7d < 0 &&
		s.Price.TrendR2_7d != nil && *s.Price.TrendR2_7d >= cfg.MinTrendR2 {
		sell = sell * math.Pow(1+*s.Price.Trend7d, float64(cfg.HoldDays))
	}

	sellPrice := domain.Price(math.Floor(sell))
	if sellPrice < cfg.MinSellPrice {
		return Threshold{}, RejectTooCheap
	}
	receive := pricing.SellerReceives(sellPrice)
	maxBuy := domain.Price(math.Floor(float64(receive) / (1 + cfg.TargetProfit)))

	qtyCap := min(float64(cfg.MaxQtyPerItem), math.Ceil(s.Liquidity.SalesPerDay7d*float64(cfg.HoldDays)*cfg.MaxMarketShare))

	return Threshold{
		MaxBuy:    maxBuy,
		SellPrice: sellPrice,
		Receive:   receive,
		QtyCap:    int(qtyCap),
	}, ""
}

// BuildThresholds считает пороги по всем вариантам.
// Возвращает пороги подходящих вариантов и число отказов по каждой причине — для лога.
func BuildThresholds(stats []steamstats.VariantStats, now time.Time, cfg Config) (map[string]Threshold, map[RejectReason]int) {
	results := make(map[string]Threshold, len(stats))
	rejects := make(map[RejectReason]int)
	for _, variant := range stats {
		threshold, rejectReason := ComputeThreshold(&variant, now, cfg)
		if rejectReason != "" {
			rejects[rejectReason]++
			continue
		}
		results[variant.HashName] = threshold
	}
	return results, rejects
}
