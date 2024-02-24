package common

import (
	"testing"
)

func TestVersionLessThan(t *testing.T) {
	type testCases struct {
		Name     string
		VersionA string
		VersionB string
		Expected bool
	}

	cases := []testCases{
		{
			Name:     "8 < 8",
			VersionA: "8",
			VersionB: "8",
			Expected: false,
		},
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

func TestVersionGreaterThanOrEqual(t *testing.T) {
	type testCases struct {
		Name     string
		VersionA string
		VersionB string
		Expected bool
	}

	cases := []testCases{
		{
			Name:     "8.1 > 8",
			VersionA: "8.1",
			VersionB: "8",
			Expected: true,
		},
		{
			Name:     "8.2 > 8.1",
			VersionA: "8.2",
			VersionB: "8.1",
			Expected: true,
		},
		{
			Name:     "9 > 8",
			VersionA: "9",
			VersionB: "8",
			Expected: true,
		},
		{
			Name:     "9 > 8.1",
			VersionA: "9",
			VersionB: "8.1",
			Expected: true,
		},
		{
			Name:     "9.1 > 8.1",
			VersionA: "9.1",
			VersionB: "8.1",
			Expected: true,
		},
		{
			Name:     "8.10 > 8",
			VersionA: "8.10",
			VersionB: "8",
			Expected: true,
		},
		{
			Name:     "8.10 > 8.1",
			VersionA: "8.10",
			VersionB: "8.1",
			Expected: true,
		},
		{
			Name:     "8.11 > 8.10",
			VersionA: "8.11",
			VersionB: "8.10",
			Expected: true,
		},
		{
			Name:     "8.6 > 8.10",
			VersionA: "8.6",
			VersionB: "8.10",
			Expected: false,
		},
		{
			Name:     "9-stream > 8-stream",
			VersionA: "9-stream",
			VersionB: "8-stream",
			Expected: true,
		},
		{
			Name:     "9.1 > 9-stream",
			VersionA: "9.1",
			VersionB: "9-stream",
			Expected: true,
		},
		{
			Name:     "9.1 >= 9.1",
			VersionA: "9.1",
			VersionB: "9.1",
			Expected: true,
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			if VersionGreaterThanOrEqual(c.VersionA, c.VersionB) != c.Expected {
				t.Fatalf("VersionGreaterThanOrEqual(%s, %s) returned unexpected result", c.VersionA, c.VersionB)
			}
		})
	}
}
