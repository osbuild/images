package osbuild_test

import (
	"testing"

	"github.com/osbuild/images/pkg/osbuild"
	"github.com/stretchr/testify/require"
)

func TestAnacondaStageOptions(t *testing.T) {

	type testCase struct {
		enable   []string
		disable  []string
		expected []string
	}

	testCases := map[string]testCase{
		"empty-args": {
			expected: []string{
				"org.fedoraproject.Anaconda.Modules.Payloads",
				"org.fedoraproject.Anaconda.Modules.Network",
				"org.fedoraproject.Anaconda.Modules.Storage",
			},
		},
		"no-op": {
			enable: []string{
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
			enable: []string{
				"org.fedoraproject.Anaconda.Modules.Users",
			},
			expected: []string{
				"org.fedoraproject.Anaconda.Modules.Payloads",
				"org.fedoraproject.Anaconda.Modules.Network",
				"org.fedoraproject.Anaconda.Modules.Storage",
				"org.fedoraproject.Anaconda.Modules.Users",
			},
		},
		"add-multi": {
			enable: []string{
				"org.fedoraproject.Anaconda.Modules.Subscription",
				"org.fedoraproject.Anaconda.Modules.Timezone",
				"org.fedoraproject.Anaconda.Modules.Users",
			},
			expected: []string{
				"org.fedoraproject.Anaconda.Modules.Payloads",
				"org.fedoraproject.Anaconda.Modules.Network",
				"org.fedoraproject.Anaconda.Modules.Storage",
				"org.fedoraproject.Anaconda.Modules.Subscription",
				"org.fedoraproject.Anaconda.Modules.Timezone",
				"org.fedoraproject.Anaconda.Modules.Users",
			},
		},
		"add-nonsense": {
			enable: []string{
				"org.osbuild.not.anaconda.module",
			},
			expected: []string{
				"org.fedoraproject.Anaconda.Modules.Payloads",
				"org.fedoraproject.Anaconda.Modules.Network",
				"org.fedoraproject.Anaconda.Modules.Storage",
				"org.osbuild.not.anaconda.module",
			},
		},
		"no-op-disable": {
			disable: []string{
				"org.fedoraproject.Anaconda.Modules.Users",
			},
			expected: []string{
				"org.fedoraproject.Anaconda.Modules.Payloads",
				"org.fedoraproject.Anaconda.Modules.Network",
				"org.fedoraproject.Anaconda.Modules.Storage",
			},
		},
		"disable-all": {
			disable: []string{
				"org.fedoraproject.Anaconda.Modules.Payloads",
				"org.fedoraproject.Anaconda.Modules.Network",
				"org.fedoraproject.Anaconda.Modules.Storage",
			},
			expected: nil,
		},
		"disable-one": {
			disable: []string{
				"org.fedoraproject.Anaconda.Modules.Storage",
			},
			expected: []string{
				"org.fedoraproject.Anaconda.Modules.Payloads",
				"org.fedoraproject.Anaconda.Modules.Network",
			},
		},
		"enable-then-disable": {
			enable: []string{
				"org.fedoraproject.Anaconda.Modules.Services",
			},
			disable: []string{
				"org.fedoraproject.Anaconda.Modules.Services",
			},
			expected: []string{
				"org.fedoraproject.Anaconda.Modules.Storage",
				"org.fedoraproject.Anaconda.Modules.Payloads",
				"org.fedoraproject.Anaconda.Modules.Network",
			},
		},
		"enable-then-disable-nonsense": {
			enable: []string{
				"org.osbuild.not.anaconda.module.2",
			},
			disable: []string{
				"org.osbuild.not.anaconda.module.2",
			},
			expected: []string{
				"org.fedoraproject.Anaconda.Modules.Storage",
				"org.fedoraproject.Anaconda.Modules.Payloads",
				"org.fedoraproject.Anaconda.Modules.Network",
			},
		},
		"enable-then-disable-multi": {
			enable: []string{
				"org.fedoraproject.Anaconda.Modules.Subscription",
				"org.fedoraproject.Anaconda.Modules.Timezone",
				"org.fedoraproject.Anaconda.Modules.Users",
			},
			disable: []string{
				"org.fedoraproject.Anaconda.Modules.Subscription",
				"org.fedoraproject.Anaconda.Modules.Timezone",
				"org.fedoraproject.Anaconda.Modules.Users",
			},
			expected: []string{
				"org.fedoraproject.Anaconda.Modules.Storage",
				"org.fedoraproject.Anaconda.Modules.Payloads",
				"org.fedoraproject.Anaconda.Modules.Network",
			},
		},
	}

	for name := range testCases {
		tc := testCases[name]
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			options := osbuild.NewAnacondaStageOptions(tc.enable, tc.disable)

			require.NotNil(options)
			require.ElementsMatch(options.KickstartModules, tc.expected)
		})
	}

}
