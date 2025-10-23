package osbuild

type (
	StatusJSON = statusJSON
)

var (
	NewSyncedWriter = newSyncedWriter
)

func MockOSBuildCmd(s string) (restore func()) {
	saved := osbuildCmd
	osbuildCmd = s
	return func() {
		osbuildCmd = saved
	}
}
