package common

import (
	"strings"
)

// ShlexQuote implements a variant of pythons shlex.quote()
// in go (c.f. https://github.com/python/cpython/blob/3.14/Lib/shlex.py#L320)
func ShlexQuote(s string) string {
	// use single quotes, and put single quotes into double quotes
	// the string $'b is then quoted as '$'"'"'b'
	return `'` + strings.Replace(s, `'`, `'"'"'`, -1) + `'`
}
