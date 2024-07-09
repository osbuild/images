package otkdistro_test

import (
	"fmt"
	"testing"

	"github.com/osbuild/images/pkg/otkdistro"
	"github.com/stretchr/testify/require"
)

func TestDistroLoad(t *testing.T) {
	require := require.New(t)

	// TODO: we can write the fragments during the test setup and make the
	// whole test self-contained
	distro, err := otkdistro.New("../../test/data/otk/fakedistro")
	require.NoError(err)
	require.Equal("FakeDistro", distro.Name())
	require.Equal("42.0", distro.OsVersion())
	require.Equal("42", distro.Releasever())
	// TODO: check runner when it's added

	archImageTypes := make([]string, 0)
	for _, archName := range distro.ListArches() {
		arch, err := distro.GetArch(archName)
		require.NoError(err)

		for _, imageTypeName := range arch.ListImageTypes() {
			archImageTypes = append(archImageTypes, fmt.Sprintf("%s/%s", archName, imageTypeName))
		}
	}

	expected := []string{
		"aarch64/qcow2",
		"fakearch/qcow2",
		"x86_64/qcow2",
	}

	require.ElementsMatch(expected, archImageTypes)
}
