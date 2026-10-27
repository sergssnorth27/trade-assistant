package converter

import (
	"fmt"
	"strconv"
	"strings"
)

// DollarPrice — цена, пришедшая в долларах: 0.15, 85.244, 10 (числом или строкой).
// Хранит тысячные. Парсится из десятичного текста, без float.
type DollarPrice int64

func (p *DollarPrice) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	v, err := ParseThousandths(s)
	if err != nil {
		return fmt.Errorf("dollar price %q: %w", s, err)
	}
	*p = DollarPrice(v)
	return nil
}

func ParseThousandths(s string) (int64, error) {
	intPart, fracPart, hasDot := strings.Cut(s, ".")

	if intPart == "" || (hasDot && fracPart == "") {
		return 0, fmt.Errorf("invalid number %q", s)
	}

	if len(fracPart) > 3 {
		return 0, fmt.Errorf("too many decimal places in %q", s)
	}

	fracPart += strings.Repeat("0", 3-len(fracPart))
	digits := intPart + fracPart

	// Только цифры: так отсекаем минус, плюс, пробелы, буквы.
	for _, r := range digits {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("invalid number %q", s)
		}
	}

	v, err := strconv.ParseInt(digits, 10, 64) // заодно поймает переполнение
	if err != nil {
		return 0, fmt.Errorf("parse %q: %w", s, err)
	}
	return v, nil
}
