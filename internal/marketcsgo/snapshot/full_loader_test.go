package snapshot

import (
	"encoding/json"
	"testing"
)

func row(t *testing.T, s string) []json.RawMessage {
	var r []json.RawMessage
	if err := json.Unmarshal([]byte(s), &r); err != nil {
		t.Fatalf("bad test row %q: %v", s, err)
	}
	return r
}

// func newColumns(format []string) (columns, error)

func TestNewColumns(t *testing.T) {
	tests := []struct {
		name    string
		format  []string
		want    columns
		wantErr bool
	}{
		{
			name:   "real order",
			format: []string{"price", "id", "market_hash_name", "classid"},
			want:   columns{price: 0, id: 1, name: 2},
		},
		{
			name:   "other order",
			format: []string{"classid", "market_hash_name", "id", "price"},
			want:   columns{price: 3, id: 2, name: 1},
		},
		{name: "no price", format: []string{"id", "market_hash_name"}, wantErr: true},
		{name: "no id", format: []string{"price", "market_hash_name"}, wantErr: true},
		{name: "no name", format: []string{"price", "id"}, wantErr: true},
		{name: "empty", format: nil, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newColumns(tt.format)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}

}
