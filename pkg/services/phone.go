package services

import (
	"errors"
	"strings"
)

// ErrInvalidPhone is returned for a phone number that isn't 8-15 digits with its country code
var ErrInvalidPhone = errors.New("enter a valid phone number with its country code")

// NormalizePhone stores phone numbers one way: digits only, country code first, no "+" (e.g. 254712345678,
// which is also what M-Pesa takes). It accepts "+254 712 345 678", "00254712345678", "254712345678" and, as
// Kenya is the default, a local "0712 345 678". Empty stays empty.
func NormalizePhone(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", nil
	}
	var digits strings.Builder
	for i, r := range s {
		switch {
		case r >= '0' && r <= '9':
			digits.WriteRune(r)
		case r == '+' && i == 0, r == ' ', r == '-', r == '(', r == ')', r == '.':
		default:
			return "", ErrInvalidPhone
		}
	}
	d := digits.String()
	switch {
	case strings.HasPrefix(d, "00"):
		d = d[2:]
	case strings.HasPrefix(d, "0") && !strings.HasPrefix(s, "+"):
		d = "254" + d[1:]
	}
	if len(d) < 8 || len(d) > 15 || d[0] == '0' {
		return "", ErrInvalidPhone
	}
	return d, nil
}
