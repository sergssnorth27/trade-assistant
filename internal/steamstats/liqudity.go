package steamstats

import (
	"time"

	"github.com/sergssnorth27/trade-assistant/internal/steam"
)

// daysWithSales возвращает, сколько дней были продажи.
func daysWithSales(days []steam.PricePoint) int {
	return len(days)
}

// topDayShare возвращает долю продаж самого активного дня от всех продаж.
// false — если продаж нет.
func topDayShare(days []steam.PricePoint) (float64, bool) {
	count := 0
	maxCount := 0
	for _, day := range days {
		if day.Purchases > maxCount {
			maxCount = day.Purchases
		}
		count += day.Purchases
	}
	if count == 0 {
		return 0, false
	}
	result := float64(maxCount) / float64(count)

	return result, true
}

// volumeTrend возвращает отношение продаж за последние 7 дней к продажам за 7 дней до них.
// false — если в предыдущие 7 дней продаж не было.
func volumeTrend(points []steam.PricePoint, now time.Time) (float64, bool) {
	w7 := windowPoints(points, now, 7*24*time.Hour)
	w14 := windowPoints(points, now, 14*24*time.Hour)
	w7Count := sumPurchases(w7)
	w14Count := sumPurchases(w14)
	prev := w14Count - w7Count
	if prev == 0 {
		return 0, false
	}
	result := float64(w7Count) / float64(prev)
	return result, true
}
