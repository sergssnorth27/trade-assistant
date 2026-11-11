package steamstats

import (
	"time"

	"github.com/sergssnorth27/trade-assistant/internal/steam"
)

// windowPoints(points, now, d) []PricePoint	Точки за последние d от now
func windowPoints(points []steam.PricePoint, now time.Time, d time.Duration) []steam.PricePoint {
	startTime := now.Add(-d)
	var result []steam.PricePoint
	for _, v := range points {
		if v.Time.Compare(startTime) == 1 {
			result = append(result, v)
		}
	}
	return result
}

// 1. sumPurchases(points []steam.PricePoint) int — цикл, который складывает Purchases.
// Это 5 строк. Потом замените ею такие же циклы в dailyPoints и weightedPercentile.
func sumPurchases(points []steam.PricePoint) int {
	count := 0
	for _, p := range points {
		count += p.Purchases
	}
	return count
}

type windowStats struct {
	Median, P25, P75 float64
	Sales            int
	OK               bool
}

func calcWindow(points []steam.PricePoint, now time.Time, d time.Duration) windowStats {
	var ws windowStats
	pricePoints := windowPoints(points, now, d)

	purchases := sumPurchases(pricePoints)
	if purchases == 0 {
		return windowStats{}
	}
	ws.Sales = purchases

	p25, _ := weightedPercentile(pricePoints, 0.25)
	ws.P25 = p25

	median, _ := weightedPercentile(pricePoints, 0.5)
	ws.Median = median

	p75, _ := weightedPercentile(pricePoints, 0.75)
	ws.P75 = p75

	ws.OK = true
	return ws
}
