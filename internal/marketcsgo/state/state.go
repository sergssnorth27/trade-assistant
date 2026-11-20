package state

import (
	"context"
	"log"
	"time"

	"github.com/sergssnorth27/trade-assistant/internal/domain"
	"github.com/sergssnorth27/trade-assistant/internal/evaluator"
	"github.com/sergssnorth27/trade-assistant/internal/marketcsgo/book"
	"github.com/sergssnorth27/trade-assistant/internal/marketcsgo/snapshot"
	"github.com/sergssnorth27/trade-assistant/internal/marketcsgo/ws"
)

const (
	maxLots = 10
	tombTTL = 15 * time.Minute
)

// thresholdLookup — откуда брать порог покупки по имени предмета (evaluator.Thresholds).
type thresholdLookup interface {
	Get(name string) (evaluator.Threshold, bool)
}

// nameResolver — всё, что владельцу нужно от словаря.
type nameResolver interface {
	Name(id int) (string, bool)
}

type Owner struct {
	books    map[string]*book.Book
	names    nameResolver
	stats    stats
	tombs    map[int64]time.Time
	requests chan bookRequest

	thresholds  thresholdLookup
	hits        chan<- evaluator.Hit // только запись: Owner отправляет, читает обработчик
	droppedHits int
}

type stats struct {
	beforeSnapshot int // события до первого снапшота
	unknownName    int // name_id нет в словаре
	unknownEvent   int // неизвестный тип события
	upserts        int // new/change применены
	removes        int // remove/sold нашли лот в стакане
	removeMiss     int // remove/sold, а лота в стакане нет
	changeHit      int // change для лота, который уже был в стакане
	badID          int
}

func New(names nameResolver, thresholds thresholdLookup, hits chan<- evaluator.Hit) *Owner {
	return &Owner{names: names, tombs: make(map[int64]time.Time), requests: make(chan bookRequest), thresholds: thresholds, hits: hits}
}

func (o *Owner) Run(ctx context.Context, snaps <-chan snapshot.Snapshot, events <-chan ws.Event) error {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case s := <-snaps:
			o.applySnapshot(s)
		case ev := <-events:
			o.applyEvent(ev)
		case req := <-o.requests:
			o.answer(req)
		case <-ticker.C:
			o.cleanTombs(time.Now())
			o.logStats()
		}
	}
}

// checkBestLots проверяет самый дешёвый лот каждого стакана.
// Вызывается после снапшота: ловит лоты, которые уже лежали на маркете,
// и предметы, ставшие выгодными после обновления порогов.
func (o *Owner) checkBestLots() {
	for name, b := range o.books {
		if lot, ok := b.Best(); ok {
			o.checkLot(name, lot.Price)
		}
	}
}

func (o *Owner) applySnapshot(s snapshot.Snapshot) {
	if o.books == nil {
		o.books = s.Books
		o.checkBestLots() // +
		return
	}

	start := time.Now()
	merged := make(map[string]*book.Book, len(s.Books))
	for name, snapBook := range s.Books {
		if b := mergeBook(o.books[name], snapBook, s.Time, o.tombs, maxLots); b.Len() > 0 {
			merged[name] = b
		}
	}

	for name, cur := range o.books {
		if _, ok := s.Books[name]; ok {
			continue
		}
		if b := mergeBook(cur, nil, s.Time, o.tombs, maxLots); b.Len() > 0 {
			merged[name] = b
		}
	}

	o.books = merged
	log.Printf("state: merged snapshot %s: %d books in %v",
		s.Time.Format(time.TimeOnly), len(merged), time.Since(start).Round(time.Millisecond))
	o.checkBestLots() // +

}
func (o *Owner) applyEvent(ev ws.Event) {
	if o.books == nil {
		o.stats.beforeSnapshot++
		return
	}
	name, ok := o.names.Name(ev.NameID)
	if !ok {
		o.stats.unknownName++
		return
	}
	id, err := ev.ID.Int64()
	if err != nil {
		o.stats.badID++ // новое поле в stats
		return
	}

	switch ev.Event {
	case "new", "change":
		b, ok := o.books[name]
		if !ok {
			b = book.New(maxLots)
			o.books[name] = b
		}
		if ev.Event == "change" && b.Has(id) {
			o.stats.changeHit++
		}
		b.Upsert(book.Lot{
			ID:        id,
			Price:     domain.Price(ev.Price),
			UpdatedAt: ev.ReceivedAt,
		})
		o.checkLot(name, domain.Price(ev.Price))
		o.stats.upserts++
	case "remove", "sold":
		if b, ok := o.books[name]; ok && b.Remove(id) {
			o.stats.removes++
		} else {
			o.stats.removeMiss++
		}
		o.tombs[id] = ev.ReceivedAt
	default:
		o.stats.unknownEvent++
	}
}

func (o *Owner) logStats() {
	log.Printf("tombs=%d", len(o.tombs))
	log.Printf("state: books=%d", len(o.books))
	log.Printf("stats: beforeSnapshot: %v, unknownName: %v, unknownEvent: %v, upserts: %v, removes: %v, removeMiss: %v, changeHit: %v", o.stats.beforeSnapshot, o.stats.unknownName, o.stats.unknownEvent, o.stats.upserts, o.stats.removes, o.stats.removeMiss, o.stats.changeHit)
	log.Printf("evaluator: dropped hits: %d", o.droppedHits) // +
	log.Printf("bad id: %d", o.stats.badID)                  // +

}

func (o *Owner) cleanTombs(now time.Time) {
	for id, at := range o.tombs {
		if now.Sub(at) > tombTTL {
			delete(o.tombs, id)
		}
	}
}

// checkLot шлёт сигнал, если лот дешевле порога покупки. Никогда не ждёт: Owner не должен тормозить.
func (o *Owner) checkLot(name string, price domain.Price) {
	t, ok := o.thresholds.Get(name)
	if !ok || price > t.MaxBuy {
		return
	}
	select {
	case o.hits <- evaluator.Hit{Name: name, LotPrice: price, Threshold: t, At: time.Now()}:
	default:
		o.droppedHits++
	}
}
