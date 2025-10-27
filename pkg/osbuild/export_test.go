package osbuild

type (
	StatusJSON = statusJSON
)

func MockOsbuildCmd(s string) (restore func()) {
	saved := osbuildCmd
	osbuildCmd = s
	return func() {
		osbuildCmd = saved
	}
}
