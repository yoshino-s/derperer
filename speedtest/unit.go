package speedtest

import (
	"fmt"
	"strings"
)

type Unit struct {
	Value float64
	Uint  string
}

func (b Unit) String() string {
	switch {
	case b.Value < 1024:
		return fmt.Sprintf("%.2f%s", b.Value, b.Uint)
	case b.Value < 1024*1024:
		return fmt.Sprintf("%.2fK%s", b.Value/1024, b.Uint)
	case b.Value < 1024*1024*1024:
		return fmt.Sprintf("%.2fM%s", b.Value/1024/1024, b.Uint)
	default:
		return fmt.Sprintf("%.2fG%s", b.Value/1024/1024/1024, b.Uint)
	}
}

func ParseUnit(s string, unit string) (Unit, error) {
	s = strings.TrimSuffix(s, unit)
	// get last char
	c := s[len(s)-1]
	var f float64 = 1
	switch c {
	case 'K':
		s = strings.TrimSuffix(s, "K")
		f = 1024
	case 'M':
		s = strings.TrimSuffix(s, "M")
		f = 1024 * 1024
	case 'G':
		s = strings.TrimSuffix(s, "G")
		f = 1024 * 1024 * 1024
	}
	var n float64

	if _, err := fmt.Sscanf(s, "%f", &n); err != nil {
		return Unit{}, err
	}
	return Unit{n * f, unit}, nil
}
