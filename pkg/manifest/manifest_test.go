package manifest_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osbuild/images/pkg/manifest"
	"github.com/osbuild/images/pkg/osbuild"
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

func findStage(name string, stages []*osbuild.Stage) *osbuild.Stage {
	for _, s := range stages {
		if s.Type == name {
			return s
		}
	}
	return nil
}

func findStages(name string, stages []*osbuild.Stage) []*osbuild.Stage {
	var foundStages []*osbuild.Stage
	for _, s := range stages {
		if s.Type == name {
			foundStages = append(foundStages, s)
		}
	}
	return foundStages
}
