package state

import (
	"context"

	"github.com/sergssnorth27/trade-assistant/internal/marketcsgo/book"
)

// BookView — копия стакана для чтения снаружи владельца.
type BookView struct {
	Name  string
	Found bool
	Lots  []book.Lot
}

// bookRequest — запрос «дай стакан этого предмета» и канал, куда прислать ответ.
type bookRequest struct {
	name  string
	reply chan BookView
}

// Book запрашивает у владельца копию стакана.
// Безопасно вызывать из любой горутины: сам стакан читает только владелец.
func (o *Owner) Book(ctx context.Context, name string) (BookView, error) {
	req := bookRequest{name: name, reply: make(chan BookView, 1)}

	select {
	case o.requests <- req: // отдали запрос владельцу
	case <-ctx.Done():
		return BookView{}, ctx.Err()
	}

	select {
	case v := <-req.reply: // ждём ответ
		return v, nil
	case <-ctx.Done():
		return BookView{}, ctx.Err()
	}
}

// answer выполняется внутри Run, то есть в горутине владельца.
func (o *Owner) answer(req bookRequest) {
	v := BookView{Name: req.name}
	if b, ok := o.books[req.name]; ok {
		v.Found = true
		v.Lots = b.Lots() // копия, а не сам стакан
	}
	req.reply <- v // буфер 1: не зависнем, даже если спросивший уже ушёл по таймауту
}
