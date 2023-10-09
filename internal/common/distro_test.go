package common

import (
	"reflect"
	"strings"
	"testing"
)

func TestOSRelease(t *testing.T) {
	var cases = []struct {
		Input     string
		OSRelease map[string]string
	}{
		{
			``,
			map[string]string{},
		},
		{
			`NAME=Fedora
VERSION="30 (Workstation Edition)"
ID=fedora
VERSION_ID=30
VERSION_CODENAME=""
PLATFORM_ID="platform:f30"
PRETTY_NAME="Fedora 30 (Workstation Edition)"
VARIANT="Workstation Edition"
VARIANT_ID=workstation`,
			map[string]string{
				"NAME":             "Fedora",
				"VERSION":          "30 (Workstation Edition)",
				"ID":               "fedora",
				"VERSION_ID":       "30",
				"VERSION_CODENAME": "",
				"PLATFORM_ID":      "platform:f30",
				"PRETTY_NAME":      "Fedora 30 (Workstation Edition)",
				"VARIANT":          "Workstation Edition",
				"VARIANT_ID":       "workstation",
			},
		},
	}

	for i, c := range cases {
		r := strings.NewReader(c.Input)

		osrelease, err := readOSRelease(r)
		if err != nil {
			t.Fatalf("%d: readOSRelease: %v", i, err)
		}

		if !reflect.DeepEqual(osrelease, c.OSRelease) {
			t.Fatalf("%d: readOSRelease returned unexpected result: %#v", i, osrelease)
		}
	}
}

func TestVersionLessThan(t *testing.T) {
	type testCases struct {
		Name     string
		VersionA string
		VersionB string
		Expected bool
	}

	cases := []testCases{
		{
			Name:     "8 < 8.1",
			VersionA: "8",
			VersionB: "8.1",
			Expected: true,
		},
		{
			Name:     "8.1 < 8.2",
			VersionA: "8.1",
			VersionB: "8.2",
			Expected: true,
		},
		{
			Name:     "8 < 9",
			VersionA: "8",
			VersionB: "9",
			Expected: true,
		},
		{
			Name:     "8.1 < 9",
			VersionA: "8.1",
			VersionB: "9",
			Expected: true,
		},
		{
			Name:     "8.1 < 9.1",
			VersionA: "8.1",
			VersionB: "9.1",
			Expected: true,
		},
		{
			Name:     "8 < 8.10",
			VersionA: "8",
			VersionB: "8.10",
			Expected: true,
		},
		{
			Name:     "8.1 < 8.10",
			VersionA: "8.1",
			VersionB: "8.10",
			Expected: true,
		},
		{
			Name:     "8.10 < 8.11",
			VersionA: "8.10",
			VersionB: "8.11",
			Expected: true,
		},
		{
			Name:     "8.10 > 8.6",
			VersionA: "8.10",
			VersionB: "8.6",
			Expected: false,
		},
		{
			Name:     "8-stream < 9-stream",
			VersionA: "8-stream",
			VersionB: "9-stream",
			Expected: true,
		},
		{
			Name:     "9-stream < 9.1",
			VersionA: "9-stream",
			VersionB: "9.1",
			Expected: true,
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			if VersionLessThan(c.VersionA, c.VersionB) != c.Expected {
				t.Fatalf("VersionLessThan(%s, %s) returned unexpected result", c.VersionA, c.VersionB)
			}
		})
	}
}
