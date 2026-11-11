package steamstats

import (
	"slices"
	"time"

	"github.com/sergssnorth27/trade-assistant/internal/steam"
)

// internal/steamstats/
// ├── percentile.go     weightedPercentile, percentile           (уже есть)
// ├── daily.go          dailyPoints                              (уже есть)
// ├── window.go         windowPoints, sumPurchases, windowStats, calcWindow
// ├── price.go          ratio, minMaxDaily, trend, athAtl
// ├── liquidity.go      daysWithSales, topDayShare, volumeTrend
// ├── risk.go           forwardReturns, volatility, maxDrawdown
// ├── quality.go        firstLastSale, isMixedPhases
// ├── stats.go          VariantStats, PriceStats, LiquidityStats, RiskStats, QualityStats, Config
// └── compute.go        Compute, toUSD

func dailyPoints(points []steam.PricePoint) []steam.PricePoint {
	m := make(map[time.Time][]steam.PricePoint)
	for _, p := range points {
		day := p.Time.Truncate(24 * time.Hour)
		m[day] = append(m[day], p)
	}

	result := make([]steam.PricePoint, 0, len(m))
	for k, v := range m {
		median, ok := weightedPercentile(v, 0.5)
		if !ok {
			continue
		}

		purchases := sumPurchases(v)
		result = append(result, steam.PricePoint{Time: k, Median: median, Purchases: purchases})
	}
	slices.SortFunc(result, func(a, b steam.PricePoint) int {
		return a.Time.Compare(b.Time)
	})

	return result
}
