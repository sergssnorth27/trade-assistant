package steamstats

import (
	"slices"

	"github.com/sergssnorth27/trade-assistant/internal/steam"
)

// weightedPercentile возвращает цену, ниже которой прошла доля p (от 0 до 1) всех продаж. false — если продаж нет
func weightedPercentile(points []steam.PricePoint, p float64) (float64, bool) {
	purchases := sumPurchases(points)
	if purchases == 0 {
		return 0, false
	}

	sorted := slices.Clone(points)
	slices.SortFunc(sorted, func(a, b steam.PricePoint) int {
		if a.Median > b.Median {
			return 1
		} else if a.Median < b.Median {
			return -1
		}
		return 0
	})

	counterPurchases := 0
	number := float64(purchases) * p
	for _, v := range sorted {
		counterPurchases += v.Purchases
		if float64(counterPurchases) >= number {
			return v.Median, true
		}
	}
	return 0, false
}
