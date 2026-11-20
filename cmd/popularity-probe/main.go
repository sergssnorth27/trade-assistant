package main

import (
	"cmp"
	"context"
	"fmt"
	"log"
	"net/http"
	"slices"
	"time"

	"github.com/sergssnorth27/trade-assistant/internal/marketcsgo/popularity"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	start := time.Now()
	sales, err := popularity.Load(ctx, &http.Client{Timeout: time.Minute})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("loaded %d names in %s\n\n", len(sales), time.Since(start).Round(time.Millisecond))

	// Сколько предметов пройдёт фильтр при разных порогах
	// и сколько займёт проход full-history при 2 запросах в секунду.
	for _, min := range []int{1, 5, 10, 20, 25, 30, 35, 50, 100, 300} {
		n := 0
		for _, s := range sales {
			if s >= min {
				n++
			}
		}
		fmt.Printf("sales7d >= %-4d → %6d предметов, full-history ≈ %s\n",
			min, n, (time.Duration(n) * time.Second / 2).Round(time.Minute))
	}

	type row struct {
		name  string
		sales int
	}
	rows := make([]row, 0, len(sales))
	for name, s := range sales {
		rows = append(rows, row{name, s})
	}
	slices.SortFunc(rows, func(a, b row) int { return cmp.Compare(b.sales, a.sales) })

	fmt.Println("\nтоп-15 по продажам:")
	for _, r := range rows[:min(15, len(rows))] {
		fmt.Printf("  %6d  %s\n", r.sales, r.name)
	}

	for _, name := range []string{"AK-47 | Slate (Field-Tested)", "Gamma Case", "USP-S | Blueprint (Minimal Wear)"} {
		fmt.Printf("\n%s: %d", name, sales[name])
	}
	fmt.Println()
}
