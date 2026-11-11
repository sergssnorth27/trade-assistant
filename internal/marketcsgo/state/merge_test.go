package state

import (
	"testing"
	"time"

	"github.com/sergssnorth27/trade-assistant/internal/domain"
	"github.com/sergssnorth27/trade-assistant/internal/marketcsgo/book"
)

// t0 — момент, когда снят снапшот.
var t0 = time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)

// at возвращает время t0 + d.
func at(d time.Duration) time.Time { return t0.Add(d) }

// newBook собирает стакан из лотов.
func newBook(lots ...book.Lot) *book.Book {
	b := book.New(maxLots)
	for _, l := range lots {
		b.Upsert(l)
	}
	return b
}

// want — ожидаемый лот в результате: ID и цена.
type want struct {
	id    int64
	price domain.Price
}

// checkBook сравнивает лоты стакана с ожидаемыми (по порядку).
func checkBook(t *testing.T, b *book.Book, expected []want) {
	t.Helper()
	got := b.Lots()
	if len(got) != len(expected) {
		t.Fatalf("got %d lots %v, want %d %v", len(got), got, len(expected), expected)
	}
	for i, w := range expected {
		if got[i].ID != w.id || got[i].Price != w.price {
			t.Errorf("lot %d = {id %d, price %v}, want {id %d, price %v}",
				i, got[i].ID, got[i].Price, w.id, w.price)
		}
	}
}

// История с цветными лотами: откат цены, воскрешение, потерянный sold.
func TestMergeBookStory(t *testing.T) {
	cur := newBook(
		book.Lot{ID: 1, Price: 9000, UpdatedAt: at(1 * time.Minute)},   // Красный: сокет в 12:01 → $9
		book.Lot{ID: 3, Price: 15000, UpdatedAt: at(-5 * time.Minute)}, // Зелёный: из прошлого снапшота
		book.Lot{ID: 4, Price: 20000, UpdatedAt: at(-5 * time.Minute)}, // Жёлтый: продан, sold потерялся
	)
	snap := newBook(
		book.Lot{ID: 1, Price: 10000, UpdatedAt: t0}, // Красный $10 (устарело)
		book.Lot{ID: 2, Price: 12000, UpdatedAt: t0}, // Синий $12 (продан в 12:02)
		book.Lot{ID: 3, Price: 15000, UpdatedAt: t0}, // Зелёный $15
	)
	tombs := map[int64]time.Time{2: at(2 * time.Minute)} // Синий удалён в 12:02

	got := mergeBook(cur, snap, t0, tombs, maxLots)

	checkBook(t, got, []want{
		{1, 9000},  // Красный: версия сокета, без отката
		{3, 15000}, // Зелёный: из снапшота
	}) // Синий не воскрес, Жёлтый пропал
}

// Сокет изменил лот ровно в момент снапшота — доверяем сокету.
func TestMergeBookSameTime(t *testing.T) {
	cur := newBook(book.Lot{ID: 1, Price: 9000, UpdatedAt: t0})
	snap := newBook(book.Lot{ID: 1, Price: 10000, UpdatedAt: t0})

	got := mergeBook(cur, snap, t0, nil, maxLots)

	checkBook(t, got, []want{{1, 9000}})
}

// Лот удалён ДО снапшота, а снапшот снова его содержит — верим снапшоту.
func TestMergeBookTombBeforeSnapshot(t *testing.T) {
	snap := newBook(book.Lot{ID: 1, Price: 10000, UpdatedAt: t0})
	tombs := map[int64]time.Time{1: at(-time.Minute)}

	got := mergeBook(nil, snap, t0, tombs, maxLots)

	checkBook(t, got, []want{{1, 10000}})
}

// Новый лот из сокета после снапшота, которого в снапшоте нет, остаётся.
func TestMergeBookNewFromSocket(t *testing.T) {
	cur := newBook(book.Lot{ID: 5, Price: 8000, UpdatedAt: at(30 * time.Second)})
	snap := newBook(book.Lot{ID: 1, Price: 10000, UpdatedAt: t0})

	got := mergeBook(cur, snap, t0, nil, maxLots)

	checkBook(t, got, []want{{5, 8000}, {1, 10000}})
}

// Предмета нет у нас — берём снапшот целиком.
func TestMergeBookNilCur(t *testing.T) {
	snap := newBook(
		book.Lot{ID: 1, Price: 10000, UpdatedAt: t0},
		book.Lot{ID: 2, Price: 11000, UpdatedAt: t0},
	)

	got := mergeBook(nil, snap, t0, nil, maxLots)

	checkBook(t, got, []want{{1, 10000}, {2, 11000}})
}

// Предмета нет в снапшоте — остаются только лоты, изменённые сокетом после снапшота.
func TestMergeBookNilSnap(t *testing.T) {
	cur := newBook(
		book.Lot{ID: 1, Price: 9000, UpdatedAt: at(time.Minute)},  // после снапшота — остаётся
		book.Lot{ID: 2, Price: 9500, UpdatedAt: at(-time.Minute)}, // до снапшота — пропадает
	)

	got := mergeBook(cur, nil, t0, nil, maxLots)

	checkBook(t, got, []want{{1, 9000}})
}

// Оба nil — пустой стакан без паники.
func TestMergeBookBothNil(t *testing.T) {
	got := mergeBook(nil, nil, t0, nil, maxLots)
	if got.Len() != 0 {
		t.Errorf("Len() = %d, want 0", got.Len())
	}
}

// Лотов больше maxLots — остаются самые дешёвые.
func TestMergeBookTrim(t *testing.T) {
	cur := book.New(maxLots)
	snap := book.New(maxLots)
	for i := int64(1); i <= 6; i++ {
		cur.Upsert(book.Lot{ID: i, Price: domain.Price(i * 1000), UpdatedAt: at(time.Minute)}) // 1000..6000
		snap.Upsert(book.Lot{ID: 100 + i, Price: domain.Price(i*1000 + 500), UpdatedAt: t0})   // 1500..6500
	}

	got := mergeBook(cur, snap, t0, nil, maxLots)

	if got.Len() != maxLots {
		t.Fatalf("Len() = %d, want %d", got.Len(), maxLots)
	}
	lots := got.Lots()
	if lots[0].Price != 1000 || lots[maxLots-1].Price != 5500 {
		t.Errorf("prices from %v to %v, want 1.000 to 5.500", lots[0].Price, lots[maxLots-1].Price)
	}
}
