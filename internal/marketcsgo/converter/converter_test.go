package converter

import "testing"

func TestParseThousandths(t *testing.T) {
	tests := []struct {
		in      string
		want    int64
		wantErr bool
	}{
		{"0.15", 150, false},
		{"10.374", 10374, false},
		{"10", 10000, false},
		{"0.001", 1, false},
		{"10.3213", 0, true},
		{"", 0, true},
		{"abc", 0, true},
		{"-1", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := ParseThousandths(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}
