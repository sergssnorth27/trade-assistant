package steamstats

import (
	"math"
	"time"

	"github.com/sergssnorth27/trade-assistant/internal/steam"
)

func ratio(a, b windowStats) (float64, bool) {
	if a.Sales == 0 || b.Sales == 0 {
		return 0, false
	}
	result := a.Median / b.Median
	return result, true
}

// minMaxDaily возвращает минимальную и максимальную дневную медиану за окно d. false — если дней с продажами нет.
func minMaxDaily(daily []steam.PricePoint, now time.Time, d time.Duration) (minPrice, maxPrice float64, ok bool) {
	window := windowPoints(daily, now, d)
	if len(window) == 0 {
		return 0, 0, false
	}
	minPrice, maxPrice = window[0].Median, window[0].Median
	for _, p := range window {
		if p.Median < minPrice {
			minPrice = p.Median
		}
		if p.Median > maxPrice {
			maxPrice = p.Median
		}
	}
	return minPrice, maxPrice, true
}

// linearRegression возвращает наклон b и R² прямой, проведённой методом наименьших квадратов.
// false — если точек меньше 2, длины слайсов разные или все x одинаковые.
func linearRegression(xs, ys []float64) (b, r2 float64, ok bool) {
	n := len(xs)
	if n < 2 || n != len(ys) {
		return 0, 0, false
	}

	var sumX, sumY float64
	for i := range xs {
		sumX += xs[i]
		sumY += ys[i]
	}
	meanX := sumX / float64(n)
	meanY := sumY / float64(n)

	var sxy, sxx, syy float64
	for i := range xs {
		dx := xs[i] - meanX
		dy := ys[i] - meanY
		sxy += dx * dy
		sxx += dx * dx
		syy += dy * dy
	}

	if sxx == 0 { // все x одинаковые — наклон не определён
		return 0, 0, false
	}
	b = sxy / sxx

	if syy == 0 { // все y одинаковые — идеально ровная линия
		return b, 1, true
	}
	r2 = sxy * sxy / (sxx * syy)
	return b, r2, true
}

//b = сумма(dx·dy) / сумма(dx²) = −13 / 10 = −1.3
//r2 = sxy*sxy / (sxx*syy), верните b, r2, true.

// trend возвращает изменение цены в день (−0.01 = −1%/день) и R² по дневным медианам за окно d.
// false — если дней с продажами меньше 3.
func trend(daily []steam.PricePoint, now time.Time, d time.Duration) (perDay, r2 float64, ok bool) {
	w := windowPoints(daily, now, d)
	if len(w) < 3 {
		return 0, 0, false
	}

	xs := make([]float64, len(w))
	ys := make([]float64, len(w))
	for i, p := range w {
		xs[i] = p.Time.Sub(w[0].Time).Hours() / 24 // номер дня от начала окна
		ys[i] = math.Log(p.Median)                 // логарифм цены, чтобы наклон был в процентах
	}

	b, r2, ok := linearRegression(xs, ys)
	if !ok {
		return 0, 0, false
	}
	return math.Exp(b) - 1, r2, true
}

// athAtl возвращает дни с максимальной и минимальной дневной медианой за всю историю.
// false — если дней с продажами нет.
func athAtl(daily []steam.PricePoint) (ath, atl steam.PricePoint, ok bool) {
	if len(daily) == 0 {
		return steam.PricePoint{}, steam.PricePoint{}, false
	}
	ath = daily[0]
	atl = daily[0]
	for _, day := range daily {
		if day.Median > ath.Median {
			ath = day
		}
		if day.Median < atl.Median {
			atl = day
		}
	}
	return ath, atl, true
}
