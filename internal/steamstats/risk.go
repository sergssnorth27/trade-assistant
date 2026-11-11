package steamstats

import (
	"math"
	"slices"
	"time"

	"github.com/sergssnorth27/trade-assistant/internal/steam"
)

// volatility возвращает стандартное отклонение дневных изменений цены ln(m_i / m_{i−1}).
// false — если дней меньше 3 (меньше двух изменений).
func volatility(days []steam.PricePoint) (float64, bool) {
	if len(days) < 3 {
		return 0, false
	}
	returns := make([]float64, 0, len(days)-1)
	sumDif := 0.0
	for i := 1; i < len(days); i++ {
		diff := math.Log(days[i].Median / days[i-1].Median)
		returns = append(returns, diff)
		sumDif += diff
	}
	avgDif := sumDif / float64(len(returns))
	changeAvg := 0.0
	for _, r := range returns {
		changeAvg += (r - avgDif) * (r - avgDif)
	}
	avgSqrt := changeAvg / float64(len(returns)-1)
	return math.Sqrt(avgSqrt), true
}

// maxDrawdown возвращает наибольшее падение дневной медианы от пика до последующего дна, долей (0.25 = −25%).
// false — если дней меньше 2.
func maxDrawdown(days []steam.PricePoint) (float64, bool) {
	if len(days) < 2 {
		return 0, false
	}
	peak := days[0].Median
	maxDD := 0.0
	for _, day := range days {
		if day.Median > peak {
			peak = day.Median
		}
		dd := (peak - day.Median) / peak
		if dd > maxDD {
			maxDD = dd
		}
	}
	return maxDD, true
}

// forwardReturns возвращает изменения цены за h дней: для каждого дня, у которого есть день ровно через h дней,
// значение «медиана через h дней / медиана в этот день − 1».
func forwardReturns(days []steam.PricePoint, h int) []float64 {
	byDay := make(map[time.Time]float64)
	for _, v := range days {
		byDay[v.Time] = v.Median
	}

	result := make([]float64, 0, len(days))
	for _, day := range days {
		nextDay := day.Time.Add(time.Duration(h) * 24 * time.Hour)
		v, ok := byDay[nextDay]
		if ok {
			diff := (v / day.Median) - 1
			result = append(result, diff)
		}
	}
	return result
}

// percentile возвращает p-й перцентиль (p от 0 до 1) обычного списка чисел. false — если список пуст.
func percentile(values []float64, p float64) (float64, bool) {
	if len(values) == 0 {
		return 0, false
	}
	sorted := slices.Clone(values)
	slices.Sort(sorted)
	elementP := float64(len(sorted)-1) * p
	return sorted[int(elementP)], true
}
