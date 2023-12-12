package common

import (
	"bufio"
	"errors"
	"io"
	"os"
	"strings"

	"github.com/hashicorp/go-version"
)

// GetHostDistroName returns the name of the host distribution, such as
// "fedora-32" or "rhel-8.2". It does so by reading the /etc/os-release file.
func GetHostDistroName() (string, error) {
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return "", err
	}
	defer f.Close()
	osrelease, err := readOSRelease(f)
	if err != nil {
		return "", err
	}

	name := osrelease["ID"] + "-" + osrelease["VERSION_ID"]

	return name, nil
}

func readOSRelease(r io.Reader) (map[string]string, error) {
	osrelease := make(map[string]string)
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, errors.New("readOSRelease: invalid input")
		}

		key := strings.TrimSpace(parts[0])
		// drop all surrounding whitespace and double-quotes
		value := strings.Trim(strings.TrimSpace(parts[1]), "\"")
		osrelease[key] = value
	}

	return osrelease, nil
}

// Returns true if the version represented by the first argument is
// semantically older than the second.
//
// Meant to be used for comparing distro versions for differences between minor
// releases.
//
// Provided version strings are of any characters which are not
// digits or periods, and then split on periods.
// Assumes any missing components are 0, so 8 < 8.1.
// Evaluates to false if a and b are equal.
func VersionLessThan(a, b string) bool {
	aV, err := version.NewVersion(a)
	if err != nil {
		panic(err)
	}
	bV, err := version.NewVersion(b)
	if err != nil {
		panic(err)
	}

	return aV.LessThan(bV)
}
