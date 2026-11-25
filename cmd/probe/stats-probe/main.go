// stats-probe — ручная проверка steamstats.Compute на сохранённой странице группы Steam.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"slices"
	"strings"

	"github.com/sergssnorth27/trade-assistant/internal/steam"
	"github.com/sergssnorth27/trade-assistant/internal/steamstats"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: stats-probe <page.html> [part of variant name]")
	}
	path := os.Args[1]
	filter := ""
	if len(os.Args) > 2 {
		filter = os.Args[2]
	}

	html, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("read %s: %v", path, err)
	}

	histories, err := steam.ParseGroupHistory(html)
	if err != nil {
		log.Fatalf("parse: %v", err)
	}

	// стабильный порядок вывода
	names := make([]string, 0, len(histories))
	for name := range histories {
		names = append(names, name)
	}
	slices.Sort(names)

	cfg := steamstats.Config{HorizonDays: 7}

	for _, name := range names {
		h := histories[name]
		if len(h.Prices) == 0 {
			fmt.Printf("%-55s  no history\n", name)
			continue
		}

		// Фикстура старая: «сейчас» = время последней точки, иначе все окна пустые.
		now := h.Prices[len(h.Prices)-1].Time

		// Фикстура в рублях (валюта 5). Для прогона притворяемся, что это доллары:
		// числа будут «рубли × 1000», но видно, что расчёт работает.
		origCurrency := h.Currency
		h.Currency = 1

		if strings.Contains(name, filter) && filter != "" {
			for _, p := range h.Prices[len(h.Prices)-10:] {
				fmt.Printf("  %s  median=%.2f  purchases=%d\n", p.Time.Format("2006-01-02 15:04"), p.Median, p.Purchases)
			}
		}

		stats, err := steamstats.Compute(name, "probe", h, now, cfg)
		if err != nil {
			log.Fatalf("compute %s: %v", name, err)
		}

		fmt.Printf("%-55s  points=%-5d sales7d=%-5d median7d=%-8s trend30d=%-8s src_currency=%d\n",
			name, stats.PointsCount, stats.Liquidity.Sales7d,
			fmtPrice(stats.Price.W7d.Median), fmtFloat(stats.Price.Trend30d), origCurrency)

		if filter != "" && strings.Contains(name, filter) {
			out, _ := json.MarshalIndent(stats, "", "  ")
			fmt.Println(string(out))
		}
	}
}

func fmtPrice[T ~int64](p *T) string {
	if p == nil {
		return "NULL"
	}
	return fmt.Sprintf("%.2f", float64(*p)/1000)
}

func fmtFloat(f *float64) string {
	if f == nil {
		return "NULL"
	}
	return fmt.Sprintf("%.4f", *f)
}
