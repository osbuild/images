package assertx

import (
	"fmt"
	"regexp"

	"github.com/stretchr/testify/assert"
)

// needed until https://github.com/stretchr/testify/issues/1304 is fixed
func PanicsWithErrorRegexp(t assert.TestingT, reg *regexp.Regexp, f assert.PanicTestFunc) (assertOk bool) {
	defer func() {
		var message interface{} // nolint: gosimple

		message = recover()
		if message == nil {
			return
		}
		err, ok := message.(error)
		if !ok || err == nil {
			assert.Fail(t, fmt.Sprintf("func %#v should return an error but got: %[2]v (type %[2]T)", f, message))
			return
		}
		assertOk = assert.Regexp(t, reg, err.Error())
	}()

	f()
	return assert.Fail(t, fmt.Sprintf("func %#v should panic but did not", f))
}
