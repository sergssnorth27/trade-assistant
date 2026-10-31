package book

import (
	"slices"
	"testing"

	"github.com/sergssnorth27/trade-assistant/internal/domain"
)

func lot(id int64, price domain.Price) Lot {
	return Lot{ID: id, Price: price}
}

// ids возвращает ID лотов стакана по порядку.
func ids(b *Book) []int64 {
	var out []int64
	for _, l := range b.Lots() {
		out = append(out, l.ID)
	}
	return out
}

func TestEmptyBook(t *testing.T) {
	b := New(3)

	len := b.Len()
	_, correct := b.Best()
	if len != 0 {
		t.Errorf("no correct let, %v", len)
	}
	if correct != false {
		t.Errorf("no correct best want=false, got %v", correct)
	}
}

func TestUpsertSorts(t *testing.T) {
	b := New(3)
	b.Upsert(lot(1, 9200))
	b.Upsert(lot(2, 9050))
	b.Upsert(lot(3, 9100))

	if got, want := ids(b), []int64{2, 3, 1}; !slices.Equal(got, want) {
		t.Errorf("ids = %v, want %v", got, want)
	}
}

func TestUpsertLowPriceIntoFull(t *testing.T) {
	b := New(3)
	b.Upsert(lot(1, 9200))
	b.Upsert(lot(2, 9050))
	b.Upsert(lot(3, 9100))

	changed := b.Upsert(lot(4, 8500))
	if !changed {
		t.Error("book not changed")
	}
	if got, want := ids(b), []int64{4, 2, 3}; !slices.Equal(got, want) {
		t.Errorf("ids = %v, want %v", got, want)
	}
}

func TestUpsertHighPriceIntoFull(t *testing.T) {
	b := New(3)
	b.Upsert(lot(1, 9200))
	b.Upsert(lot(2, 9050))
	b.Upsert(lot(3, 9100))

	changed := b.Upsert(lot(4, 10000))
	if changed {
		t.Error("book changed")
	}
	if got, want := ids(b), []int64{2, 3, 1}; !slices.Equal(got, want) {
		t.Errorf("ids = %v, want %v", got, want)
	}
}

func TestLastItemChangedOnLowPrice(t *testing.T) {
	b := New(3)
	b.Upsert(lot(1, 9200))
	b.Upsert(lot(2, 9050))
	b.Upsert(lot(3, 9100))

	changed := b.Upsert(lot(1, 8500))
	if !changed {
		t.Error("book not changed")
	}

	if got, want := ids(b), []int64{1, 2, 3}; !slices.Equal(got, want) {
		t.Errorf("ids = %v, want %v", got, want)
	}
}

func TestFirstItemChangedOnHighPrice(t *testing.T) {
	b := New(3)
	b.Upsert(lot(1, 9200))
	b.Upsert(lot(2, 9050))
	b.Upsert(lot(3, 9100))

	changed := b.Upsert(lot(2, 10000))
	if !changed {
		t.Error("book not changed")
	}

	if got, want := ids(b), []int64{3, 1, 2}; !slices.Equal(got, want) {
		t.Errorf("ids = %v, want %v", got, want)
	}
}

func TestRemoveLot(t *testing.T) {
	b := New(3)
	b.Upsert(lot(1, 9200))
	b.Upsert(lot(2, 9050))
	b.Upsert(lot(3, 9100))

	RemoveExisting := b.Remove(1)
	if !RemoveExisting {
		t.Error("remove existing = false")
	}
	RemoveNoExisting := b.Remove(10)
	if RemoveNoExisting {
		t.Error("remove no existing = true")
	}

	if got, want := ids(b), []int64{2, 3}; !slices.Equal(got, want) {
		t.Errorf("ids = %v, want %v", got, want)
	}
}

func TestAddSamePriceLot(t *testing.T) {
	b := New(3)
	b.Upsert(lot(4, 9200))
	b.Upsert(lot(1, 9200))
	b.Upsert(lot(10, 9200))

	if got, want := ids(b), []int64{1, 4, 10}; !slices.Equal(got, want) {
		t.Errorf("ids = %v, want %v", got, want)
	}
}

func TestCopyBook(t *testing.T) {
	b := New(3)
	b.Upsert(lot(1, 9200))
	b.Upsert(lot(2, 9050))
	b.Upsert(lot(3, 9100))

	copyBook := b.Lots()
	copyBook[0].Price = 1

	lot, _ := b.Best()
	if lot.Price != 9050 {
		t.Errorf("copy not working, price %v", lot.Price)
	}
}

func TestNewPanicsOnZero(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected panic")
		}
	}()
	New(0)
}
