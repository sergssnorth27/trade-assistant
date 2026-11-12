package steamstats

import (
	"errors"
	"math"
	"time"

	"github.com/sergssnorth27/trade-assistant/internal/domain"
	"github.com/sergssnorth27/trade-assistant/internal/steam"
)

var ErrNotUSD = errors.New("history currency no USD")

// toPrice переводит цену из долларов (float64) в тысячные доли доллара.
func toPrice(v float64) domain.Price {
	return domain.Price(math.Round(v * 1000))
}

func convertWindowStatsToPriceWindow(ws windowStats) PriceWindow {
	if !ws.OK {
		return PriceWindow{} // все три поля nil
	}
	return PriceWindow{
		Median: ptr(toPrice(ws.Median)),
		P25:    ptr(toPrice(ws.P25)),
		P75:    ptr(toPrice(ws.P75)),
	}
}

func priceOrNil(v float64, ok bool) *domain.Price {
	if !ok {
		return nil
	}
	return ptr(toPrice(v))
}

func floatOrNil(v float64, ok bool) *float64 {
	if !ok {
		return nil
	}
	return &v
}

// Compute считает всю статистику по истории одного варианта предмета.
func Compute(hashName, groupID string, h steam.VariantHistory, now time.Time, cfg Config) (VariantStats, error) {
	if h.Currency != 1 {
		return VariantStats{}, ErrNotUSD
	}

	variant := VariantStats{
		HashName:       hashName,
		GroupID:        groupID,
		ComputedAt:     now,
		SourceCurrency: h.Currency,
		PointsCount:    len(h.Prices),
		UsdRate:        1,
	}
	variant.Quality.MixedPhases = isMixedPhases(hashName)

	if len(h.Prices) == 0 {
		return variant, nil // истории нет — только служебные поля, остальное NULL
	}
	variant.LastPointAt = ptr(h.Prices[len(h.Prices)-1].Time)

	wS24Hours := calcWindow(h.Prices, now, 24*time.Hour)
	wS3Days := calcWindow(h.Prices, now, 3*24*time.Hour)
	wS7Days := calcWindow(h.Prices, now, 7*24*time.Hour)
	wS30Days := calcWindow(h.Prices, now, 30*24*time.Hour)
	wS60Days := calcWindow(h.Prices, now, 60*24*time.Hour)
	wS90Days := calcWindow(h.Prices, now, 90*24*time.Hour)

	variant.Price.W24h = convertWindowStatsToPriceWindow(wS24Hours)
	variant.Price.W3d = convertWindowStatsToPriceWindow(wS3Days)
	variant.Price.W7d = convertWindowStatsToPriceWindow(wS7Days)
	variant.Price.W30d = convertWindowStatsToPriceWindow(wS30Days)
	variant.Price.W60d = convertWindowStatsToPriceWindow(wS60Days)
	variant.Price.W90d = convertWindowStatsToPriceWindow(wS90Days)

	daily := dailyPoints(h.Prices)

	w7Days := windowPoints(daily, now, 7*24*time.Hour)
	w30Days := windowPoints(daily, now, 30*24*time.Hour)
	w60Days := windowPoints(daily, now, 60*24*time.Hour)
	w90Days := windowPoints(daily, now, 90*24*time.Hour)

	min7Days, max7Days, ok7Days := minMaxDaily(w7Days)
	min30Days, max30Days, ok30Days := minMaxDaily(w30Days)
	min60Days, max60Days, ok60Days := minMaxDaily(w60Days)
	min90Days, max90Days, ok90Days := minMaxDaily(w90Days)

	variant.Price.MinDaily7d = priceOrNil(min7Days, ok7Days)
	variant.Price.MinDaily30d = priceOrNil(min30Days, ok30Days)
	variant.Price.MinDaily60d = priceOrNil(min60Days, ok60Days)
	variant.Price.MinDaily90d = priceOrNil(min90Days, ok90Days)

	variant.Price.MaxDaily7d = priceOrNil(max7Days, ok7Days)
	variant.Price.MaxDaily30d = priceOrNil(max30Days, ok30Days)
	variant.Price.MaxDaily60d = priceOrNil(max60Days, ok60Days)
	variant.Price.MaxDaily90d = priceOrNil(max90Days, ok90Days)

	variant.Price.Ratio24h7d = floatOrNil(ratio(wS24Hours, wS7Days))
	variant.Price.Ratio3d7d = floatOrNil(ratio(wS3Days, wS7Days))
	variant.Price.Ratio7d30d = floatOrNil(ratio(wS7Days, wS30Days))
	variant.Price.Ratio30d90d = floatOrNil(ratio(wS30Days, wS90Days))

	tPer7Day, r2_7Day, ok7Day := trend(w7Days)
	tPer30Day, r2_30Day, ok30Day := trend(w30Days)

	variant.Price.Trend7d = floatOrNil(tPer7Day, ok7Day)
	variant.Price.Trend30d = floatOrNil(tPer30Day, ok30Day)
	variant.Price.TrendR2_7d = floatOrNil(r2_7Day, ok7Day)
	variant.Price.TrendR2_30d = floatOrNil(r2_30Day, ok30Day)

	ath, atl, ok := athAtl(daily)
	variant.Price.ATH = priceOrNil(ath.Median, ok)
	variant.Price.ATL = priceOrNil(atl.Median, ok)
	if ok {
		variant.Price.ATHAt = ptr(ath.Time)
		variant.Price.ATLAt = ptr(atl.Time)
	}

	variant.Liquidity.Sales24h = wS24Hours.Sales
	variant.Liquidity.Sales3d = wS3Days.Sales
	variant.Liquidity.Sales7d = wS7Days.Sales
	variant.Liquidity.Sales30d = wS30Days.Sales
	variant.Liquidity.Sales60d = wS60Days.Sales
	variant.Liquidity.Sales90d = wS90Days.Sales

	variant.Liquidity.SalesPerDay7d = float64(wS7Days.Sales) / 7
	variant.Liquidity.SalesPerDay30d = float64(wS30Days.Sales) / 30

	variant.Liquidity.DaysWithSales7d = daysWithSales(w7Days)
	variant.Liquidity.DaysWithSales30d = daysWithSales(w30Days)

	variant.Liquidity.TopDayShare7d = floatOrNil(topDayShare(w7Days))
	variant.Liquidity.VolumeTrend7d = floatOrNil(volumeTrend(h.Prices, now))

	fwd := forwardReturns(w90Days, cfg.HorizonDays)
	variant.Risk.FwdP10 = floatOrNil(percentile(fwd, 0.1))
	variant.Risk.FwdP50 = floatOrNil(percentile(fwd, 0.5))

	variant.Risk.FwdDays = cfg.HorizonDays
	variant.Risk.FwdSamples = len(fwd)

	variant.Risk.Volatility7d = floatOrNil(volatility(w7Days))
	variant.Risk.Volatility30d = floatOrNil(volatility(w30Days))
	variant.Risk.MaxDrawdown90d = floatOrNil(maxDrawdown(w90Days))

	first, last, okFirstLast := firstLastSale(h.Prices)
	if okFirstLast {
		variant.Quality.FirstSaleAt = ptr(first.Time)
		variant.Quality.LastSaleAt = ptr(last.Time)
		variant.Quality.LastSalePrice = ptr(toPrice(last.Median))
		variant.Quality.AgeDays = ptr(int(now.Sub(first.Time).Hours() / 24))
	}

	return variant, nil
}
