package osbuild_test

import (
	"testing"

	"github.com/osbuild/images/pkg/customizations/anaconda"
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
				anaconda.ModulePayloads,
				anaconda.ModuleNetwork,
				anaconda.ModuleStorage,
			},
		},
		"no-op": {
			enable: []string{
				anaconda.ModulePayloads,
				anaconda.ModuleNetwork,
				anaconda.ModuleStorage,
			},
			expected: []string{
				anaconda.ModulePayloads,
				anaconda.ModuleNetwork,
				anaconda.ModuleStorage,
			},
		},
		"add-users": {
			enable: []string{
				anaconda.ModuleUsers,
			},
			expected: []string{
				anaconda.ModulePayloads,
				anaconda.ModuleNetwork,
				anaconda.ModuleStorage,
				anaconda.ModuleUsers,
			},
		},
		"add-multi": {
			enable: []string{
				anaconda.ModuleSubscription,
				anaconda.ModuleTimezone,
				anaconda.ModuleUsers,
			},
			expected: []string{
				anaconda.ModulePayloads,
				anaconda.ModuleNetwork,
				anaconda.ModuleStorage,
				anaconda.ModuleSubscription,
				anaconda.ModuleTimezone,
				anaconda.ModuleUsers,
			},
		},
		"add-nonsense": {
			enable: []string{
				"org.osbuild.not.anaconda.module",
			},
			expected: []string{
				anaconda.ModulePayloads,
				anaconda.ModuleNetwork,
				anaconda.ModuleStorage,
				"org.osbuild.not.anaconda.module",
			},
		},
		"no-op-disable": {
			disable: []string{
				anaconda.ModuleUsers,
			},
			expected: []string{
				anaconda.ModulePayloads,
				anaconda.ModuleNetwork,
				anaconda.ModuleStorage,
			},
		},
		"disable-all": {
			disable: []string{
				anaconda.ModulePayloads,
				anaconda.ModuleNetwork,
				anaconda.ModuleStorage,
			},
			expected: nil,
		},
		"disable-one": {
			disable: []string{
				anaconda.ModuleStorage,
			},
			expected: []string{
				anaconda.ModulePayloads,
				anaconda.ModuleNetwork,
			},
		},
		"enable-then-disable": {
			enable: []string{
				anaconda.ModuleServices,
			},
			disable: []string{
				anaconda.ModuleServices,
			},
			expected: []string{
				anaconda.ModuleStorage,
				anaconda.ModulePayloads,
				anaconda.ModuleNetwork,
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
				anaconda.ModuleStorage,
				anaconda.ModulePayloads,
				anaconda.ModuleNetwork,
			},
		},
		"enable-then-disable-multi": {
			enable: []string{
				anaconda.ModuleSubscription,
				anaconda.ModuleTimezone,
				anaconda.ModuleUsers,
			},
			disable: []string{
				anaconda.ModuleSubscription,
				anaconda.ModuleTimezone,
				anaconda.ModuleUsers,
			},
			expected: []string{
				anaconda.ModuleStorage,
				anaconda.ModulePayloads,
				anaconda.ModuleNetwork,
			},
		},
	}

	for name := range testCases {
		tc := testCases[name]
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			options := osbuild.NewAnacondaStageOptionsLegacy(tc.enable, tc.disable)

			require.NotNil(options)
			require.ElementsMatch(options.KickstartModules, tc.expected)
		})
	}

}
