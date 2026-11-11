package state

import (
	"context"
	"log"
	"time"

	"github.com/sergssnorth27/trade-assistant/internal/domain"
	"github.com/sergssnorth27/trade-assistant/internal/marketcsgo/book"
	"github.com/sergssnorth27/trade-assistant/internal/marketcsgo/snapshot"
	"github.com/sergssnorth27/trade-assistant/internal/marketcsgo/ws"
)

const (
	maxLots = 10
	tombTTL = 15 * time.Minute
)

// nameResolver — всё, что владельцу нужно от словаря.
type nameResolver interface {
	Name(id int) (string, bool)
}

type Owner struct {
	books    map[string]*book.Book
	names    nameResolver
	watch    string
	stats    stats
	tombs    map[int64]time.Time
	requests chan bookRequest
}

type stats struct {
	beforeSnapshot int // события до первого снапшота
	unknownName    int // name_id нет в словаре
	unknownEvent   int // неизвестный тип события
	upserts        int // new/change применены
	removes        int // remove/sold нашли лот в стакане
	removeMiss     int // remove/sold, а лота в стакане нет
	changeHit      int // change для лота, который уже был в стакане
}

func New(names nameResolver, watch string) *Owner {
	return &Owner{names: names, watch: watch, tombs: make(map[int64]time.Time), requests: make(chan bookRequest)}
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

func (o *Owner) applySnapshot(s snapshot.Snapshot) {
	if o.books == nil {
		o.books = s.Books
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
	switch ev.Event {
	case "new", "change":
		b, ok := o.books[name]
		if !ok {
			b = book.New(maxLots)
			o.books[name] = b
		}
		if ev.Event == "change" && b.Has(ev.ID) {
			o.stats.changeHit++
		}
		b.Upsert(book.Lot{
			ID:        ev.ID,
			Price:     domain.Price(ev.Price),
			UpdatedAt: ev.ReceivedAt,
		})
		o.stats.upserts++
	case "remove", "sold":
		if b, ok := o.books[name]; ok && b.Remove(ev.ID) {
			o.stats.removes++
		} else {
			o.stats.removeMiss++
		}
		o.tombs[ev.ID] = ev.ReceivedAt
	default:
		o.stats.unknownEvent++
	}
}

func (o *Owner) logStats() {
	log.Printf("tombs=%d", len(o.tombs))
	log.Printf("state: books=%d", len(o.books))
	log.Printf("stats: beforeSnapshot: %v, unknownName: %v, unknownEvent: %v, upserts: %v, removes: %v, removeMiss: %v, changeHit: %v", o.stats.beforeSnapshot, o.stats.unknownName, o.stats.unknownEvent, o.stats.upserts, o.stats.removes, o.stats.removeMiss, o.stats.changeHit)

	b, ok := o.books[o.watch]
	if !ok {
		log.Printf("state: %s: нет стакана", o.watch)
		return
	}
	log.Printf("state: %s: %v", o.watch, b.Lots())
}

func (o *Owner) cleanTombs(now time.Time) {
	for id, at := range o.tombs {
		if now.Sub(at) > tombTTL {
			delete(o.tombs, id)
		}
	}
}
