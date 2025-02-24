package experimentalflags

import (
	"os"
	"strconv"
	"strings"
)

const envKEY = "IMAGE_BUILDER_EXPERIMENTAL"

func experimentalOptions() map[string]string {
	expMap := map[string]string{}

	env := os.Getenv(envKEY)
	if env == "" {
		return expMap
	}

	for _, s := range strings.Split(env, ",") {
		l := strings.SplitN(s, "=", 2)
		switch len(l) {
		case 1:
			expMap[l[0]] = "true"
		case 2:
			expMap[l[0]] = l[1]
		}
	}

	return expMap
}

// Bool returns true if there is a boolean option with the given
// option name
func Bool(option string) bool {
	expMap := experimentalOptions()
	b, err := strconv.ParseBool(expMap[option])
	if err != nil {
		// not much we can do for invalid inputs, just assume false
		return false
	}
	return b
}

func String(option string) string {
	expMap := experimentalOptions()
	return expMap[option]
}
