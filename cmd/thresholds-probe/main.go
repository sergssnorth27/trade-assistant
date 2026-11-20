// thresholds-probe — строит пороги по статистике из БД и выгружает всё в thresholds.csv для анализа.
package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/sergssnorth27/trade-assistant/internal/domain"
	"github.com/sergssnorth27/trade-assistant/internal/evaluator"
	"github.com/sergssnorth27/trade-assistant/internal/storage/postgres"
)

func main() {
	_ = godotenv.Load()
	ctx := context.Background()

	pool, err := postgres.NewPool(ctx, os.Getenv("GOOSE_DBSTRING"))
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	stats, err := postgres.NewSteamVariantStatsRepo(pool).LoadForThresholds(ctx)
	if err != nil {
		log.Fatal(err)
	}

	f, err := os.Create("thresholds.csv")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()

	w.Write([]string{
		"name", "status",
		"median_7d", "median_24h", "sales_7d", "sales_24h", "sales_per_day_7d",
		"days_with_sales_7d", "top_day_share_7d",
		"ratio_24h_7d", "ratio_3d_7d", "trend_7d", "trend_r2_7d",
		"fwd_samples", "fwd_p10", "last_sale_age_h",
		"sell", "receive", "max_buy", "qty_cap",
	})

	now := time.Now()
	cfg := evaluator.DefaultConfig()
	ok := 0
	for i := range stats {
		s := &stats[i]
		t, reason := evaluator.ComputeThreshold(s, now, cfg)
		status := string(reason)
		if reason == "" {
			status = "ok"
			ok++
		}

		lastSaleAge := ""
		if s.Quality.LastSaleAt != nil {
			lastSaleAge = fmt.Sprintf("%.1f", now.Sub(*s.Quality.LastSaleAt).Hours())
		}

		w.Write([]string{
			s.HashName, status,
			price(s.Price.W7d.Median), price(s.Price.W24h.Median),
			strconv.Itoa(s.Liquidity.Sales7d), strconv.Itoa(s.Liquidity.Sales24h),
			fmt.Sprintf("%.2f", s.Liquidity.SalesPerDay7d),
			strconv.Itoa(s.Liquidity.DaysWithSales7d), float(s.Liquidity.TopDayShare7d),
			float(s.Price.Ratio24h7d), float(s.Price.Ratio3d7d),
			float(s.Price.Trend7d), float(s.Price.TrendR2_7d),
			strconv.Itoa(s.Risk.FwdSamples), float(s.Risk.FwdP10), lastSaleAge,
			priceIf(reason == "", t.SellPrice), priceIf(reason == "", t.Receive),
			priceIf(reason == "", t.MaxBuy), qtyIf(reason == "", t.QtyCap),
		})
	}
	log.Printf("written thresholds.csv: %d variants, %d ok", len(stats), ok)
}

// Цены — в долларах, чтобы CSV было удобно читать: 15400 → "15.400".
func price(p *domain.Price) string {
	if p == nil {
		return ""
	}
	return fmt.Sprintf("%.3f", float64(*p)/1000)
}

func priceIf(ok bool, p domain.Price) string {
	if !ok {
		return ""
	}
	return price(&p)
}

func qtyIf(ok bool, q int) string {
	if !ok {
		return ""
	}
	return strconv.Itoa(q)
}

func float(f *float64) string {
	if f == nil {
		return ""
	}
	return fmt.Sprintf("%.4f", *f)
}
