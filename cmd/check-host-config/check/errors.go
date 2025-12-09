package check

import (
	"errors"
	"fmt"
	"strings"
)

var ErrCheckSkipped = errors.New("skip")
var ErrCheckFailed = errors.New("fail")
var ErrCheckWarning = errors.New("warn")

func Skip(reason string) error {
	return fmt.Errorf("%w: %s", ErrCheckSkipped, reason)
}

func Pass() error {
	return nil
}

func Fail(reason ...string) error {
	msg := strings.Join(reason, " ")
	return fmt.Errorf("%w: %s", ErrCheckFailed, msg)
}

func Warning(reason ...string) error {
	msg := strings.Join(reason, " ")
	return fmt.Errorf("%w: %s", ErrCheckWarning, msg)
}

func IsSkip(err error) bool {
	return errors.Is(err, ErrCheckSkipped)
}

func IsFail(err error) bool {
	return errors.Is(err, ErrCheckFailed)
}

func IsWarning(err error) bool {
	return errors.Is(err, ErrCheckWarning)
}

func IconFor(err error) string {
	switch {
	case err == nil:
		return "🟢"
	case IsSkip(err):
		return "🔵"
	case IsWarning(err):
		return "🟠"
	case IsFail(err):
		return "🔴"
	default:
		return "🔴"
	}
}
