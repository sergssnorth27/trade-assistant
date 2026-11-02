package book

import (
	"slices"
	"time"

	"github.com/sergssnorth27/trade-assistant/internal/domain"
)

type Lot struct {
	ID        int64
	Price     domain.Price
	UpdatedAt time.Time
}

// Тип не безопасен для использования из нескольких горутин.
type Book struct {
	lots    []Lot
	maxLots int
}

func New(maxLots int) *Book {
	if maxLots <= 0 {
		panic("book: maxLots must be positive")
	}
	return &Book{
		lots:    make([]Lot, 0, maxLots),
		maxLots: maxLots,
	}
}

// Upsert добавляет лот или обновляет существующий (по ID).
// Возвращает true, если стакан изменился.
func (b *Book) Upsert(lot Lot) bool {
	// 1. Если лот уже есть — убираем старую версию.
	if i := slices.IndexFunc(b.lots, func(l Lot) bool {
		return l.ID == lot.ID
	}); i >= 0 {
		b.lots = slices.Delete(b.lots, i, i+1)
	}

	// 2. Стакан полон, а новый лот дороже самого дорогого — не вставляем.
	if len(b.lots) == b.maxLots && compareLots(lot, b.lots[len(b.lots)-1]) >= 0 {
		return false
	}

	// 3. Ищем позицию: перед первым лотом, который дороже нового.
	pos := len(b.lots)
	for i, l := range b.lots {
		if compareLots(lot, l) < 0 {
			pos = i
			break
		}
	}

	//4. Вставляем
	b.lots = slices.Insert(b.lots, pos, lot)

	// 5. Если стало больше лимита — отрезаем самый дорогой.
	if len(b.lots) > b.maxLots {
		b.lots = b.lots[:b.maxLots]
	}
	return true
}

// Has сообщает, есть ли лот с таким ID в стакане.
func (b *Book) Has(id int64) bool {
	has := slices.ContainsFunc(b.lots, func(l Lot) bool {
		return l.ID == id
	})
	return has
}

// Remove удаляет лот по ID. Возвращает true, если он был в стакане.
func (b *Book) Remove(id int64) bool {
	for i, l := range b.lots {
		if l.ID == id {
			b.lots = append(b.lots[:i], b.lots[i+1:]...)
			return true
		}
	}
	return false
}

// Best — самый дешёвый лот.
func (b *Book) Best() (Lot, bool) {
	if len(b.lots) <= 0 {
		return Lot{}, false
	}
	return b.lots[0], true
}

// Lots — копия лотов по возрастанию цены.
func (b *Book) Lots() []Lot {
	return slices.Clone(b.lots)
}

func (b *Book) Len() int {
	return len(b.lots)
}

// compareLots: отрицательное число, если a дешевле b, 0 — если одинаковые, положительное — если a дороже.
// При равной цене порядок по ID, чтобы сортировка была всегда одинаковой.
func compareLots(a, b Lot) int {
	if a.Price < b.Price {
		return -1
	}
	if a.Price > b.Price {
		return 1
	}
	// цены равны — сравниваем по ID
	if a.ID < b.ID {
		return -1
	}
	if a.ID > b.ID {
		return 1
	}
	return 0
}
