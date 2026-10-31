package services

import "testing"

func TestNormalizePhone(t *testing.T) {
	ok := map[string]string{
		"+254 712 345 678":  "254712345678",
		"254712345678":      "254712345678",
		"0712 345 678":      "254712345678",
		"00256 772 123456":  "256772123456",
		"+1 (415) 555-0100": "14155550100",
		"":                  "",
	}
	for in, want := range ok {
		if got, err := NormalizePhone(in); err != nil || got != want {
			t.Errorf("%q: got %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"12345", "+254abc", "0", "+0712345678", "2547123456789012345"} {
		if _, err := NormalizePhone(bad); err == nil {
			t.Errorf("%q: expected an error", bad)
		}
	}
}
