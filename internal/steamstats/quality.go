package steamstats

import (
	"strings"

	"github.com/sergssnorth27/trade-assistant/internal/steam"
)

// mixedVariantMarkers — предметы, у которых под одним именем продаются
// разные по цене варианты (фазы, цвета). Медиана Steam для них бессмысленна.
var mixedVariantMarkers = []string{
	"Doppler", // покрывает и Gamma Doppler
	"Charm | Semi-Precious",
}

// firstLastSale возвращает первую и последнюю точку с продажами. false — если продаж не было.
func firstLastSale(points []steam.PricePoint) (first, last steam.PricePoint, ok bool) {
	found := false
	if len(points) == 0 {
		return first, last, false
	}
	for i := 0; i < len(points); i++ {
		if points[i].Purchases > 0 {
			first = points[i]
			found = true
			break
		}
	}
	if found == false {
		return first, last, false
	}
	for i := len(points) - 1; i >= 0; i-- {
		if points[i].Purchases > 0 {
			last = points[i]
			break
		}
	}
	return first, last, true
}

func isMixedPhases(hashName string) bool {
	for _, m := range mixedVariantMarkers {
		if strings.Contains(hashName, m) {
			return true
		}
	}
	return false
}
