package steamstats

import (
	"testing"
	"time"

	"github.com/sergssnorth27/trade-assistant/internal/steam"
)

func TestWeightedPercentile(t *testing.T) {
	tests := []struct {
		name          string
		inPricePoints []steam.PricePoint
		inP           float64
		want          float64
		wantOk        bool
	}{
		{name: "base", inPricePoints: []steam.PricePoint{{Time: time.Now(), Median: 90, Purchases: 1}, {Time: time.Now(), Median: 100, Purchases: 1}, {Time: time.Now(), Median: 110, Purchases: 1}}, inP: 0.5, want: 100, wantOk: true},
		{name: "empty proints", inPricePoints: []steam.PricePoint{{}}, inP: 0.5, want: 0, wantOk: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := weightedPercentile(tt.inPricePoints, tt.inP)
			if ok != tt.wantOk {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOk)
			}
			if !tt.wantOk {
				return
			}

			if got != tt.want {
				t.Errorf("weightedPercentile got: %v, want: %v", got, tt.want)
			}
		})
	}
}
