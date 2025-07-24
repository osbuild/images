package defs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatchAndNormalizeHappy(t *testing.T) {
	for _, tc := range []struct {
		reStr, nameVer string
		expected       string
	}{
		// simple cases, no capture groups
		{`rhel-10\.[0-9]{1,2}`, "rhel-100", ""},
		{`rhel-10\.[0-9]{1,2}`, "rhel-10.0", "rhel-10.0"},
		// capture groups for major/minor
		{`(?P<name>rhel)-(?P<major>8)\.?(?P<minor>[0-9]{1,2})`, "rhel-8.10", "rhel-8.10"},
		{`(?P<name>rhel)-(?P<major>8)\.?(?P<minor>[0-9]{1,2})`, "rhel-810", "rhel-8.10"},
		// capture groups for just major
		{`(?P<name>centos)-(?P<major>[0-9])stream`, "centos-9stream", "centos-9"},
		// normalizing strange things works
		{`(?P<major>[0-9])-(?P<name>foo)`, "8-foo", "foo-8"},
	} {
		found, err := matchAndNormalize(tc.reStr, tc.nameVer)
		assert.NoError(t, err)
		assert.Equal(t, found, tc.expected)
	}
}

func TestMatchAndNormalizeSad(t *testing.T) {
	for _, tc := range []struct {
		reStr, nameVer string
		expectedErr    string
	}{
		// simple cases, bad regex
		{`rhel-10[`, "rhel-100", `cannot use "rhel-10[": error parsing regexp: missing closing ]`},
		// incomplete capture groups
		{`rhel-([0-9]+)`, "rhel-100", `invalid number of submatches for "rhel-([0-9]+)" "rhel-100" (2)`},
		// too many capture groups
		{`(rhel)-([0-9])([0-9])([0-9])`, "rhel-100", `invalid number of submatches for "(rhel)-([0-9])([0-9])([0-9])" "rhel-100" (5)`},
		// capture groups have incorrect names
		{`(?P<missingName>centos)-(?P<major>[0-9])stream`, "centos-9stream", `cannot find submatch field "name"`},
		{`(?P<name>centos)-(?P<missingMajor>[0-9])stream`, "centos-9stream", `cannot find submatch field "major"`},
		{`(?P<name>rhel)-(?P<major>8)\.?(?P<missingMinor>[0-9]{1,2})`, "rhel-8.10", `cannot find submatch field "minor"`},
	} {
		_, err := matchAndNormalize(tc.reStr, tc.nameVer)
		assert.ErrorContains(t, err, tc.expectedErr)
	}
}
