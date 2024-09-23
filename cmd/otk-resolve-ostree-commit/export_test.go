package main

var (
	Run = run
)

func MockEnvLookup() (restore func()) {
	saved := osLookupEnv
	osLookupEnv = func(key string) (string, bool) {
		if key == "OTK_UNDER_TEST" {
			return "1", true
		}
		return "", false
	}
	return func() {
		osLookupEnv = saved
	}
}
