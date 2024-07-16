package osbuild_test

import (
	"testing"

	"github.com/osbuild/images/pkg/osbuild"
	"github.com/stretchr/testify/require"
)

func TestAnacondaStageOptions(t *testing.T) {

	type testCase struct {
		additional []string
		expected   []string
	}

	testCases := map[string]testCase{
		"zero-add": {
			additional: []string{},
			expected: []string{
				"org.fedoraproject.Anaconda.Modules.Payloads",
				"org.fedoraproject.Anaconda.Modules.Network",
				"org.fedoraproject.Anaconda.Modules.Storage",
			},
		},
		"no-op": {
			additional: []string{
				"org.fedoraproject.Anaconda.Modules.Payloads",
				"org.fedoraproject.Anaconda.Modules.Network",
				"org.fedoraproject.Anaconda.Modules.Storage",
			},
			expected: []string{
				"org.fedoraproject.Anaconda.Modules.Payloads",
				"org.fedoraproject.Anaconda.Modules.Network",
				"org.fedoraproject.Anaconda.Modules.Storage",
			},
		},
		"add-users": {
			additional: []string{
				"org.fedoraproject.Anaconda.Modules.Users",
			},
			expected: []string{
				"org.fedoraproject.Anaconda.Modules.Payloads",
				"org.fedoraproject.Anaconda.Modules.Network",
				"org.fedoraproject.Anaconda.Modules.Storage",
				"org.fedoraproject.Anaconda.Modules.Users",
			},
		},
		"add-nonsense": {
			additional: []string{
				"org.osbuild.not.anaconda.module",
			},
			expected: []string{
				"org.fedoraproject.Anaconda.Modules.Payloads",
				"org.fedoraproject.Anaconda.Modules.Network",
				"org.fedoraproject.Anaconda.Modules.Storage",
				"org.osbuild.not.anaconda.module",
			},
		},
	}

	for name := range testCases {
		tc := testCases[name]
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			options := osbuild.NewAnacondaStageOptions(tc.additional)

			require.NotNil(options)
			require.ElementsMatch(options.KickstartModules, tc.expected)
		})
	}

}
