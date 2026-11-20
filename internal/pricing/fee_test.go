package pricing

import (
	"testing"

	"github.com/sergssnorth27/trade-assistant/internal/domain"
)

func TestSellerReceives(t *testing.T) {
	tests := []struct {
		name  string
		buyer domain.Price
		want  domain.Price
	}{
		{"real lot $60.00", 60000, 52190},
		{"search example $4.56", 4560, 3980}, // комиссии 19 + 39 центов, как в ответе поиска Steam
		{"$10.00", 10000, 8700},
		{"$100.00", 100000, 86970},
		{"minimum $0.03", 30, 10},
		{"too cheap $0.02", 20, 0},
		{"zero", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SellerReceives(tt.buyer)
			if got != tt.want {
				t.Errorf("SellerReceives(%d) = %d, want %d", tt.buyer, got, tt.want)
			}
		})
	}
}
