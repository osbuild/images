package progress

import (
	"io"
)

type (
	TerminalProgressBar = terminalProgressBar
	DebugProgressBar    = debugProgressBar
	PlainProgressBar    = plainProgressBar
)

func MockOsStderr(w io.Writer) (restore func()) {
	saved := osStderr
	osStderr = w
	return func() {
		osStderr = saved
	}
}

func MockIsattyIsTerminal(fn func(uintptr) bool) (restore func()) {
	saved := isattyIsTerminal
	isattyIsTerminal = fn
	return func() {
		isattyIsTerminal = saved
	}
}
