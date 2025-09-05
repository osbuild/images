package randutil

import (
	"math/rand"
)

const (
	AsciiLower  = "abcdefghijklmnopqrstuvwxyz"
	AsciiUpper  = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	AsciiDigit  = "0123456789"
	AsciiSymbol = "!@#$%^&*()-_=+[]{}|;:',.<>/?`~"
)

// String returns a (weak) random string of the length "n". When not
// passing a seqs it will generate the string based on ascii
// lower,upper,digits and a bunch of (ascii) symbols.
//
// With seqs passed it will select based on the provided
// sequences, i.e. String(10, AsciiDigits) will produce 10
// random digits.
//
// Do not use to generate secrets.
func String(n int, seqs ...string) string {
	if len(seqs) == 0 {
		seqs = []string{AsciiLower, AsciiUpper, AsciiDigit, AsciiSymbol}
	}

	var inp []byte
	for _, s := range seqs {
		inp = append(inp, []byte(s)...)
	}

	b := make([]byte, n)
	for i := range b {
		// nolint:gosec
		b[i] = inp[rand.Intn(len(inp))]
	}
	return string(b)
}
