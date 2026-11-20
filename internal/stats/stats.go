package stats

import "slices"

// percentile возвращает p-й перцентиль (p от 0 до 1) обычного списка чисел. false — если список пуст.
func Percentile(values []float64, p float64) (float64, bool) {
	if len(values) == 0 {
		return 0, false
	}
	sorted := slices.Clone(values)
	slices.Sort(sorted)
	elementP := float64(len(sorted)-1) * p
	return sorted[int(elementP)], true
}
