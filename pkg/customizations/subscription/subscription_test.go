package subscription

import (
	"fmt"
	"testing"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/blueprint"
	"github.com/stretchr/testify/assert"
)

func TestRHSMConfigFromBP(t *testing.T) {
	type testCase struct {
		bp       *blueprint.RHSMCustomization
		expected *RHSMConfig
	}

	testCases := []testCase{
		{
			bp:       nil,
			expected: nil,
		},
		{
			bp:       &blueprint.RHSMCustomization{},
			expected: nil,
		},
		{
			bp: &blueprint.RHSMCustomization{
				Config: &blueprint.RHSMConfig{},
			},
			expected: &RHSMConfig{},
		},
		{
			bp: &blueprint.RHSMCustomization{
				Config: &blueprint.RHSMConfig{
					DNFPlugins: &blueprint.SubManDNFPluginsConfig{
						ProductID: &blueprint.DNFPluginConfig{
							Enabled: common.ToPtr(true),
						},
						SubscriptionManager: &blueprint.DNFPluginConfig{
							Enabled: common.ToPtr(false),
						},
					},
					SubscriptionManager: &blueprint.SubManConfig{
						RHSMConfig: &blueprint.SubManRHSMConfig{
							ManageRepos: common.ToPtr(true),
						},
						RHSMCertdConfig: &blueprint.SubManRHSMCertdConfig{
							AutoRegistration: common.ToPtr(false),
						},
					},
				},
			},
			expected: &RHSMConfig{
				DnfPlugins: SubManDNFPluginsConfig{
					ProductID: DNFPluginConfig{
						Enabled: common.ToPtr(true),
					},
					SubscriptionManager: DNFPluginConfig{
						Enabled: common.ToPtr(false),
					},
				},
				SubMan: SubManConfig{
					Rhsm: SubManRHSMConfig{
						ManageRepos: common.ToPtr(true),
					},
					Rhsmcertd: SubManRHSMCertdConfig{
						AutoRegistration: common.ToPtr(false),
					},
				},
			},
		},
		{
			bp: &blueprint.RHSMCustomization{
				Config: &blueprint.RHSMConfig{
					DNFPlugins: &blueprint.SubManDNFPluginsConfig{
						ProductID: &blueprint.DNFPluginConfig{
							Enabled: common.ToPtr(true),
						},
						SubscriptionManager: &blueprint.DNFPluginConfig{},
					},
					SubscriptionManager: &blueprint.SubManConfig{
						RHSMConfig: &blueprint.SubManRHSMConfig{},
						RHSMCertdConfig: &blueprint.SubManRHSMCertdConfig{
							AutoRegistration: common.ToPtr(false),
						},
					},
				},
			},
			expected: &RHSMConfig{
				DnfPlugins: SubManDNFPluginsConfig{
					ProductID: DNFPluginConfig{
						Enabled: common.ToPtr(true),
					},
					SubscriptionManager: DNFPluginConfig{},
				},
				SubMan: SubManConfig{
					Rhsm: SubManRHSMConfig{},
					Rhsmcertd: SubManRHSMCertdConfig{
						AutoRegistration: common.ToPtr(false),
					},
				},
			},
		},
		{
			bp: &blueprint.RHSMCustomization{
				Config: &blueprint.RHSMConfig{
					DNFPlugins: &blueprint.SubManDNFPluginsConfig{
						ProductID: &blueprint.DNFPluginConfig{
							Enabled: common.ToPtr(true),
						},
					},
					SubscriptionManager: &blueprint.SubManConfig{
						RHSMConfig: &blueprint.SubManRHSMConfig{
							ManageRepos: common.ToPtr(true),
						},
					},
				},
			},
			expected: &RHSMConfig{
				DnfPlugins: SubManDNFPluginsConfig{
					ProductID: DNFPluginConfig{
						Enabled: common.ToPtr(true),
					},
				},
				SubMan: SubManConfig{
					Rhsm: SubManRHSMConfig{
						ManageRepos: common.ToPtr(true),
					},
				},
			},
		},
	}

	for idx, tc := range testCases {
		t.Run(fmt.Sprintf("case #%d", idx), func(t *testing.T) {
			rhsmConfig := RHSMConfigFromBP(tc.bp)
			assert.EqualValues(t, tc.expected, rhsmConfig)
		})
	}
}
