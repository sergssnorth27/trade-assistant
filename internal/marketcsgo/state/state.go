package state

import (
	"context"

	"github.com/sergssnorth27/trade-assistant/internal/marketcsgo/book"
	"github.com/sergssnorth27/trade-assistant/internal/marketcsgo/snapshot"
	"github.com/sergssnorth27/trade-assistant/internal/marketcsgo/ws"
)

// nameResolver — всё, что владельцу нужно от словаря.
type nameResolver interface {
	Name(id int) (string, bool)
}

type Owner struct {
	books map[string]*book.Book
	names nameResolver
	watch string
	stats stats
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

func New(names nameResolver, watch string) *Owner

func (o *Owner) Run(ctx context.Context, snaps <-chan snapshot.Snapshot, events <-chan ws.Event) error

func (o *Owner) applySnapshot(s snapshot.Snapshot)

func (o *Owner) applySnapshot(s snapshot.Snapshot)
func (o *Owner) applyEvent(ev ws.Event)
