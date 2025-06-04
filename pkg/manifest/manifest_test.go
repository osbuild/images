package manifest_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osbuild/images/pkg/manifest"
)

func TestDistroUnmarshal(t *testing.T) {
	var distro manifest.Distro

	for _, tc := range []struct {
		inp      string
		expected manifest.Distro
	}{
		{`"rhel-10"`, manifest.DISTRO_EL10},
		{`"rhel-9"`, manifest.DISTRO_EL9},
		{`"rhel-8"`, manifest.DISTRO_EL8},
		{`"rhel-7"`, manifest.DISTRO_EL7},
		{`"fedora"`, manifest.DISTRO_FEDORA},
	} {
		err := distro.UnmarshalJSON([]byte(tc.inp))
		assert.NoError(t, err)
		assert.Equal(t, tc.expected, distro)
	}
}
