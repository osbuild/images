package disk

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenUniqueString(t *testing.T) {
	type testCase struct {
		base     string
		existing map[string]bool
		exp      string
	}

	testCases := map[string]testCase{
		"simple": {
			base: "root",
			existing: map[string]bool{
				"one": true,
				"two": true,
			},
			exp: "root",
		},
		"collision": {
			base: "root",
			existing: map[string]bool{
				"one":  true,
				"two":  true,
				"root": true,
			},
			exp: "root00",
		},
		"collision-2": {
			base: "word",
			existing: map[string]bool{
				"word":   true,
				"word00": true,
				"word01": true,
				"other":  true,
			},
			exp: "word02",
		},
	}

	for name := range testCases {
		tc := testCases[name]
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			out, err := genUniqueString(tc.base, tc.existing)
			assert.NoError(err)
			assert.Equal(tc.exp, out)
		})
	}
}

func TestGenUniqueStringManyCollisions(t *testing.T) {
	type testCase struct {
		base        string
		ncollisions int
		exp         string
		errmsg      string
	}

	testCases := map[string]testCase{
		"baseword33": {
			base:        "baseword",
			ncollisions: 33,
			exp:         "baseword33",
		},
		"somany99": {
			base:        "somany",
			ncollisions: 99,
			exp:         "somany99",
		},
		"tk42102": {
			base:        "tk421",
			ncollisions: 2,
			exp:         "tk42102",
		},
		"so-many-collisions": {
			base:        "all-the-collisions",
			ncollisions: 100,
			errmsg:      `name collision: could not generate unique version of "all-the-collisions"`,
		},
	}

	for name := range testCases {
		tc := testCases[name]
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			existing := map[string]bool{
				tc.base: true,
			}
			for n := 0; n < tc.ncollisions; n++ {
				existing[fmt.Sprintf("%s%02d", tc.base, n)] = true
			}
			out, err := genUniqueString(tc.base, existing)
			if tc.errmsg == "" {
				assert.NoError(err)
				assert.Equal(tc.exp, out)
			} else {
				assert.EqualError(err, tc.errmsg)
			}
		})
	}
}
