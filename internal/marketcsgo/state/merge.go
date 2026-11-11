package state

import (
	"time"

	"github.com/sergssnorth27/trade-assistant/internal/marketcsgo/book"
)

// mergeBook сверяет наш стакан cur со стаканом из снапшота snap, снятого в момент t.
// Версия сокета остаётся, если он менял лот после t. Лоты, удалённые сокетом после t,
// снапшот не возвращает. Всё остальное берётся из снапшота.
func mergeBook(cur, snap *book.Book, t time.Time, tombs map[int64]time.Time, maxLots int) *book.Book {
	result := book.New(maxLots)

	// Проход 1: что сокет знает новее снапшота.
	if cur != nil {
		for _, lot := range cur.Lots() {
			if !lot.UpdatedAt.Before(t) {
				result.Upsert(lot)
			}
		}
	}

	// Проход 2: всё остальное — из снапшота.
	if snap != nil {
		for _, lot := range snap.Lots() {
			if result.Has(lot.ID) {
				continue // уже взяли версию сокета
			}
			if deleted, ok := tombs[lot.ID]; ok && !deleted.Before(t) {
				continue // не воскрешаем
			}
			result.Upsert(lot)
		}
	}

	return result
}
