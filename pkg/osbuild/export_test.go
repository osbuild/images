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

func MockOSBuildStoreCmd(s string) (restore func()) {
	saved := osbuildStoreCmd
	osbuildStoreCmd = s
	return func() {
		osbuildStoreCmd = saved
	}
}
