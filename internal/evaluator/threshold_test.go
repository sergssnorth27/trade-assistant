package evaluator

import (
	"testing"
	"time"

	"github.com/sergssnorth27/trade-assistant/internal/domain"
	"github.com/sergssnorth27/trade-assistant/internal/steamstats"
)

func ptr[T any](v T) *T { return &v }

var now = time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)

// goodStats — предмет, который проходит все проверки. Каждый случай теста портит в нём одно поле.
func goodStats() *steamstats.VariantStats {
	s := &steamstats.VariantStats{ComputedAt: now.Add(-time.Hour)}

	s.Price.W7d.Median = ptr(domain.Price(15400))  // $15.40
	s.Price.W24h.Median = ptr(domain.Price(15100)) // $15.10
	s.Price.Ratio24h7d = ptr(15100.0 / 15400)      // 0.98 — не падение
	s.Price.Ratio3d7d = ptr(1.0)
	s.Price.Trend7d = ptr(-0.005) // −0.5% в день
	s.Price.TrendR2_7d = ptr(0.6) // тренду верим

	s.Liquidity.Sales7d = 490
	s.Liquidity.Sales24h = 40
	s.Liquidity.SalesPerDay7d = 70
	s.Liquidity.DaysWithSales7d = 7
	s.Liquidity.TopDayShare7d = ptr(0.2)

	s.Risk.FwdSamples = 80
	s.Risk.FwdP10 = ptr(-0.08)

	s.Quality.LastSaleAt = ptr(now.Add(-time.Hour))
	return s
}

func TestComputeThreshold_Calculation(t *testing.T) {
	got, reason := ComputeThreshold(goodStats(), now, DefaultConfig())
	if reason != "" {
		t.Fatalf("unexpected reject: %s", reason)
	}

	want := Threshold{SellPrice: 14579, Receive: 12680, MaxBuy: 9753, QtyCap: 3}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestComputeThreshold_Rejects(t *testing.T) {
	tests := []struct {
		name   string
		modify func(s *steamstats.VariantStats)
		want   RejectReason
	}{
		{"stale stats", func(s *steamstats.VariantStats) { s.ComputedAt = now.Add(-48 * time.Hour) }, RejectStale},
		{"doppler", func(s *steamstats.VariantStats) { s.Quality.MixedPhases = true }, RejectMixedPhases},
		{"no median", func(s *steamstats.VariantStats) { s.Price.W7d.Median = nil }, RejectNoSales},
		{"few sales", func(s *steamstats.VariantStats) { s.Liquidity.Sales7d = 10 }, RejectNoSales},
		{"few days", func(s *steamstats.VariantStats) { s.Liquidity.DaysWithSales7d = 3 }, RejectIrregular},
		{"one-day spike", func(s *steamstats.VariantStats) { s.Liquidity.TopDayShare7d = ptr(0.8) }, RejectIrregular},
		{"no recent sale", func(s *steamstats.VariantStats) { s.Quality.LastSaleAt = ptr(now.Add(-72 * time.Hour)) }, RejectDead},
		{"falling 24h", func(s *steamstats.VariantStats) { s.Price.Ratio24h7d = ptr(0.85) }, RejectFalling},
		{"falling 3d", func(s *steamstats.VariantStats) { s.Price.Ratio3d7d = ptr(0.85) }, RejectFalling},
		{"risky", func(s *steamstats.VariantStats) { s.Risk.FwdP10 = ptr(-0.30) }, RejectRisky},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := goodStats()
			tt.modify(s)
			_, got := ComputeThreshold(s, now, DefaultConfig())
			if got != tt.want {
				t.Errorf("reject = %q, want %q", got, tt.want)
			}
		})
	}
}
