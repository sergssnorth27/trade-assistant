// Package marketstats считает статистику продаж market.csgo по истории
// из full-history. Ни HTTP, ни БД — на вход продажи, на выход структура.
package marketstats

import (
	"time"

	"github.com/sergssnorth27/trade-assistant/internal/domain"
	"github.com/sergssnorth27/trade-assistant/internal/marketcsgo/history"
	"github.com/sergssnorth27/trade-assistant/internal/stats"
)

// maxHistory — сколько последних продаж отдаёт full-history.
// Если продаж столько же, история обрезана, и длинные окна могут быть не покрыты.
const maxHistory = 6000

// Window — продажи за окно.
type Window struct {
	Median *domain.Price // nil — продаж нет или окно не покрыто историей
	P25    *domain.Price
	Sales  int
}

// MarketStats — статистика предмета на маркете.
type MarketStats struct {
	HashName     string
	MarketID     int64
	ComputedAt   time.Time
	W24h         Window
	W7d          Window
	W30d         Window
	W90d         Window
	OldestSaleAt *time.Time // до какого момента хватило истории
	LastSaleAt   *time.Time
}

// Compute считает статистику по продажам. Порядок продаж не важен.
func Compute(hashName string, id int64, sales []history.Sale, now time.Time) MarketStats {
	s := MarketStats{HashName: hashName, MarketID: id, ComputedAt: now}
	if len(sales) == 0 {
		return s
	}

	oldest, last := sales[0].Time, sales[0].Time
	for _, sale := range sales[1:] {
		if sale.Time.Before(oldest) {
			oldest = sale.Time
		}
		if sale.Time.After(last) {
			last = sale.Time
		}
	}
	s.OldestSaleAt = &oldest
	s.LastSaleAt = &last

	truncated := len(sales) >= maxHistory

	s.W24h = window(sales, now, 24*time.Hour, oldest, truncated)
	s.W7d = window(sales, now, 7*24*time.Hour, oldest, truncated)
	s.W30d = window(sales, now, 30*24*time.Hour, oldest, truncated)
	s.W90d = window(sales, now, 90*24*time.Hour, oldest, truncated)
	return s
}

// window считает продажи за последние d от now.
// truncated — история обрезана (maxHistory продаж). Если при этом самая старая
// продажа новее начала окна, окно не покрыто — возвращаем пустое:
// медиана по части окна была бы неправдой.
func window(sales []history.Sale, now time.Time, d time.Duration, oldest time.Time, truncated bool) Window {
	start := now.Add(-d)
	if truncated && oldest.After(start) {
		return Window{}
	}

	prices := make([]float64, 0)
	for _, sale := range sales {
		if !sale.Time.Before(start) && !sale.Time.After(now) {
			prices = append(prices, float64(sale.Price))
		}
	}
	if len(prices) == 0 {
		return Window{}
	}

	w := Window{Sales: len(prices)}
	if v, ok := stats.Percentile(prices, 0.5); ok {
		p := domain.Price(v)
		w.Median = &p
	}
	if v, ok := stats.Percentile(prices, 0.25); ok {
		p := domain.Price(v)
		w.P25 = &p
	}
	return w
}
