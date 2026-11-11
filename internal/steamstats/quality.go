package steamstats

import (
	"strings"

	"github.com/sergssnorth27/trade-assistant/internal/steam"
)

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

// isMixedPhases сообщает, смешивает ли Steam в истории этого предмета разные фазы (Doppler).
func isMixedPhases(name string) bool {
	contains := strings.Contains(name, "Doppler")
	return contains
}
