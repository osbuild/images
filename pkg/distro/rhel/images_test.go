package rhel

import (
	"testing"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/arch"
	"github.com/osbuild/images/pkg/blueprint"
	"github.com/osbuild/images/pkg/customizations/subscription"
	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/manifest"
	"github.com/osbuild/images/pkg/platform"
	"github.com/osbuild/images/pkg/rpmmd"
	"github.com/stretchr/testify/assert"
)

func TestOsCustomizationsRHSM(t *testing.T) {
	type testCase struct {
		name        string
		ic          *distro.ImageConfig
		io          *distro.ImageOptions
		bpc         *blueprint.Customizations
		expectedOsc *manifest.OSCustomizations
	}

	testCases := []testCase{
		{
			name:        "no rhsm config at all; subscription in the image options",
			ic:          &distro.ImageConfig{},
			io:          &distro.ImageOptions{Subscription: &subscription.ImageOptions{}},
			bpc:         &blueprint.Customizations{},
			expectedOsc: &manifest.OSCustomizations{},
		},
		{
			name:        "no rhsm config at all; no subscription in the image options",
			ic:          &distro.ImageConfig{},
			io:          &distro.ImageOptions{},
			bpc:         &blueprint.Customizations{},
			expectedOsc: &manifest.OSCustomizations{},
		},
		{
			name: "rhsm config in the image config, no subscription in the image options",
			ic: &distro.ImageConfig{
				RHSMConfig: map[subscription.RHSMStatus]*subscription.RHSMConfig{
					subscription.RHSMConfigNoSubscription: {
						DnfPlugins: subscription.SubManDNFPluginsConfig{
							ProductID: subscription.DNFPluginConfig{
								Enabled: common.ToPtr(false),
							},
							SubscriptionManager: subscription.DNFPluginConfig{
								Enabled: common.ToPtr(false),
							},
						},
						SubMan: subscription.SubManConfig{
							Rhsm: subscription.SubManRHSMConfig{
								ManageRepos: common.ToPtr(false),
							},
							Rhsmcertd: subscription.SubManRHSMCertdConfig{
								AutoRegistration: common.ToPtr(true),
							},
						},
					},
					subscription.RHSMConfigWithSubscription: {
						DnfPlugins: subscription.SubManDNFPluginsConfig{
							ProductID: subscription.DNFPluginConfig{
								Enabled: common.ToPtr(true),
							},
							SubscriptionManager: subscription.DNFPluginConfig{
								Enabled: common.ToPtr(true),
							},
						},
						SubMan: subscription.SubManConfig{
							Rhsm: subscription.SubManRHSMConfig{
								ManageRepos: common.ToPtr(true),
							},
							Rhsmcertd: subscription.SubManRHSMCertdConfig{
								AutoRegistration: common.ToPtr(false),
							},
						},
					},
				},
			},
			io:  &distro.ImageOptions{},
			bpc: &blueprint.Customizations{},
			expectedOsc: &manifest.OSCustomizations{
				RHSMConfig: &subscription.RHSMConfig{
					DnfPlugins: subscription.SubManDNFPluginsConfig{
						ProductID: subscription.DNFPluginConfig{
							Enabled: common.ToPtr(false),
						},
						SubscriptionManager: subscription.DNFPluginConfig{
							Enabled: common.ToPtr(false),
						},
					},
					SubMan: subscription.SubManConfig{
						Rhsm: subscription.SubManRHSMConfig{
							ManageRepos: common.ToPtr(false),
						},
						Rhsmcertd: subscription.SubManRHSMCertdConfig{
							AutoRegistration: common.ToPtr(true),
						},
					},
				},
			},
		},
		{
			name: "rhsm config in the image config, subscription in the image options",
			ic: &distro.ImageConfig{
				RHSMConfig: map[subscription.RHSMStatus]*subscription.RHSMConfig{
					subscription.RHSMConfigNoSubscription: {
						DnfPlugins: subscription.SubManDNFPluginsConfig{
							ProductID: subscription.DNFPluginConfig{
								Enabled: common.ToPtr(false),
							},
							SubscriptionManager: subscription.DNFPluginConfig{
								Enabled: common.ToPtr(false),
							},
						},
						SubMan: subscription.SubManConfig{
							Rhsm: subscription.SubManRHSMConfig{
								ManageRepos: common.ToPtr(false),
							},
							Rhsmcertd: subscription.SubManRHSMCertdConfig{
								AutoRegistration: common.ToPtr(true),
							},
						},
					},
					subscription.RHSMConfigWithSubscription: {
						DnfPlugins: subscription.SubManDNFPluginsConfig{
							ProductID: subscription.DNFPluginConfig{
								Enabled: common.ToPtr(true),
							},
							SubscriptionManager: subscription.DNFPluginConfig{
								Enabled: common.ToPtr(true),
							},
						},
						SubMan: subscription.SubManConfig{
							Rhsm: subscription.SubManRHSMConfig{
								ManageRepos: common.ToPtr(true),
							},
							Rhsmcertd: subscription.SubManRHSMCertdConfig{
								AutoRegistration: common.ToPtr(false),
							},
						},
					},
				},
			},
			io:  &distro.ImageOptions{Subscription: &subscription.ImageOptions{}},
			bpc: &blueprint.Customizations{},
			expectedOsc: &manifest.OSCustomizations{
				RHSMConfig: &subscription.RHSMConfig{
					DnfPlugins: subscription.SubManDNFPluginsConfig{
						ProductID: subscription.DNFPluginConfig{
							Enabled: common.ToPtr(true),
						},
						SubscriptionManager: subscription.DNFPluginConfig{
							Enabled: common.ToPtr(true),
						},
					},
					SubMan: subscription.SubManConfig{
						Rhsm: subscription.SubManRHSMConfig{
							ManageRepos: common.ToPtr(true),
						},
						Rhsmcertd: subscription.SubManRHSMCertdConfig{
							AutoRegistration: common.ToPtr(false),
						},
					},
				},
			},
		},
		{
			name: "no rhsm config in the image config, rhsm config in the BP, subscription in the image options",
			ic:   &distro.ImageConfig{},
			io:   &distro.ImageOptions{Subscription: &subscription.ImageOptions{}},
			bpc: &blueprint.Customizations{
				RHSM: &blueprint.RHSMCustomization{
					Config: &blueprint.RHSMConfig{
						DNFPlugins: &blueprint.SubManDNFPluginsConfig{
							ProductID: &blueprint.DNFPluginConfig{
								Enabled: common.ToPtr(true),
							},
							SubscriptionManager: &blueprint.DNFPluginConfig{
								Enabled: common.ToPtr(true),
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
			},
			expectedOsc: &manifest.OSCustomizations{
				RHSMConfig: &subscription.RHSMConfig{
					DnfPlugins: subscription.SubManDNFPluginsConfig{
						ProductID: subscription.DNFPluginConfig{
							Enabled: common.ToPtr(true),
						},
						SubscriptionManager: subscription.DNFPluginConfig{
							Enabled: common.ToPtr(true),
						},
					},
					SubMan: subscription.SubManConfig{
						Rhsm: subscription.SubManRHSMConfig{
							ManageRepos: common.ToPtr(true),
						},
						Rhsmcertd: subscription.SubManRHSMCertdConfig{
							AutoRegistration: common.ToPtr(false),
						},
					},
				},
			},
		},
		{
			name: "no rhsm config in the image config, rhsm config in the BP, no subscription in the image options",
			ic:   &distro.ImageConfig{},
			io:   &distro.ImageOptions{},
			bpc: &blueprint.Customizations{
				RHSM: &blueprint.RHSMCustomization{
					Config: &blueprint.RHSMConfig{
						DNFPlugins: &blueprint.SubManDNFPluginsConfig{
							ProductID: &blueprint.DNFPluginConfig{
								Enabled: common.ToPtr(true),
							},
							SubscriptionManager: &blueprint.DNFPluginConfig{
								Enabled: common.ToPtr(true),
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
			},
			expectedOsc: &manifest.OSCustomizations{
				RHSMConfig: &subscription.RHSMConfig{
					DnfPlugins: subscription.SubManDNFPluginsConfig{
						ProductID: subscription.DNFPluginConfig{
							Enabled: common.ToPtr(true),
						},
						SubscriptionManager: subscription.DNFPluginConfig{
							Enabled: common.ToPtr(true),
						},
					},
					SubMan: subscription.SubManConfig{
						Rhsm: subscription.SubManRHSMConfig{
							ManageRepos: common.ToPtr(true),
						},
						Rhsmcertd: subscription.SubManRHSMCertdConfig{
							AutoRegistration: common.ToPtr(false),
						},
					},
				},
			},
		},
		{
			name: "rhsm config in the image config, rhsm config in the BP, no subscription in the image options",
			ic: &distro.ImageConfig{
				RHSMConfig: map[subscription.RHSMStatus]*subscription.RHSMConfig{
					subscription.RHSMConfigNoSubscription: {
						DnfPlugins: subscription.SubManDNFPluginsConfig{
							ProductID: subscription.DNFPluginConfig{
								Enabled: common.ToPtr(true),
							},
							SubscriptionManager: subscription.DNFPluginConfig{
								Enabled: common.ToPtr(false),
							},
						},
						SubMan: subscription.SubManConfig{
							Rhsm: subscription.SubManRHSMConfig{
								ManageRepos: common.ToPtr(false),
							},
							Rhsmcertd: subscription.SubManRHSMCertdConfig{
								AutoRegistration: common.ToPtr(true),
							},
						},
					},
					subscription.RHSMConfigWithSubscription: {
						DnfPlugins: subscription.SubManDNFPluginsConfig{
							ProductID: subscription.DNFPluginConfig{
								Enabled: common.ToPtr(false),
							},
							SubscriptionManager: subscription.DNFPluginConfig{
								Enabled: common.ToPtr(true),
							},
						},
						SubMan: subscription.SubManConfig{
							Rhsm: subscription.SubManRHSMConfig{
								ManageRepos: common.ToPtr(true),
							},
							Rhsmcertd: subscription.SubManRHSMCertdConfig{
								AutoRegistration: common.ToPtr(false),
							},
						},
					},
				},
			},
			io: &distro.ImageOptions{},
			bpc: &blueprint.Customizations{
				RHSM: &blueprint.RHSMCustomization{
					Config: &blueprint.RHSMConfig{
						DNFPlugins: &blueprint.SubManDNFPluginsConfig{
							SubscriptionManager: &blueprint.DNFPluginConfig{
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
			},
			expectedOsc: &manifest.OSCustomizations{
				RHSMConfig: &subscription.RHSMConfig{
					DnfPlugins: subscription.SubManDNFPluginsConfig{
						ProductID: subscription.DNFPluginConfig{
							Enabled: common.ToPtr(true),
						},
						SubscriptionManager: subscription.DNFPluginConfig{
							Enabled: common.ToPtr(true),
						},
					},
					SubMan: subscription.SubManConfig{
						Rhsm: subscription.SubManRHSMConfig{
							ManageRepos: common.ToPtr(true),
						},
						Rhsmcertd: subscription.SubManRHSMCertdConfig{
							AutoRegistration: common.ToPtr(true),
						},
					},
				},
			},
		},
		{
			name: "rhsm config in the image config, rhsm config in the BP, subscription in the image options",
			ic: &distro.ImageConfig{
				RHSMConfig: map[subscription.RHSMStatus]*subscription.RHSMConfig{
					subscription.RHSMConfigNoSubscription: {
						DnfPlugins: subscription.SubManDNFPluginsConfig{
							ProductID: subscription.DNFPluginConfig{
								Enabled: common.ToPtr(true),
							},
							SubscriptionManager: subscription.DNFPluginConfig{
								Enabled: common.ToPtr(false),
							},
						},
						SubMan: subscription.SubManConfig{
							Rhsm: subscription.SubManRHSMConfig{
								ManageRepos: common.ToPtr(false),
							},
							Rhsmcertd: subscription.SubManRHSMCertdConfig{
								AutoRegistration: common.ToPtr(true),
							},
						},
					},
					subscription.RHSMConfigWithSubscription: {
						DnfPlugins: subscription.SubManDNFPluginsConfig{
							ProductID: subscription.DNFPluginConfig{
								Enabled: common.ToPtr(false),
							},
							SubscriptionManager: subscription.DNFPluginConfig{
								Enabled: common.ToPtr(true),
							},
						},
						SubMan: subscription.SubManConfig{
							Rhsm: subscription.SubManRHSMConfig{
								ManageRepos: common.ToPtr(true),
							},
							Rhsmcertd: subscription.SubManRHSMCertdConfig{
								AutoRegistration: common.ToPtr(false),
							},
						},
					},
				},
			},
			io: &distro.ImageOptions{Subscription: &subscription.ImageOptions{}},
			bpc: &blueprint.Customizations{
				RHSM: &blueprint.RHSMCustomization{
					Config: &blueprint.RHSMConfig{
						DNFPlugins: &blueprint.SubManDNFPluginsConfig{
							ProductID: &blueprint.DNFPluginConfig{
								Enabled: common.ToPtr(true),
							},
						},
						SubscriptionManager: &blueprint.SubManConfig{
							RHSMCertdConfig: &blueprint.SubManRHSMCertdConfig{
								AutoRegistration: common.ToPtr(true),
							},
						},
					},
				},
			},
			expectedOsc: &manifest.OSCustomizations{
				RHSMConfig: &subscription.RHSMConfig{
					DnfPlugins: subscription.SubManDNFPluginsConfig{
						ProductID: subscription.DNFPluginConfig{
							Enabled: common.ToPtr(true),
						},
						SubscriptionManager: subscription.DNFPluginConfig{
							Enabled: common.ToPtr(true),
						},
					},
					SubMan: subscription.SubManConfig{
						Rhsm: subscription.SubManRHSMConfig{
							ManageRepos: common.ToPtr(true),
						},
						Rhsmcertd: subscription.SubManRHSMCertdConfig{
							AutoRegistration: common.ToPtr(true),
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testDistro, err := NewDistribution("rhel", 9, 0)
			assert.NoError(t, err)
			testArch := NewArchitecture(testDistro, arch.ARCH_X86_64)
			it := &ImageType{DefaultImageConfig: tc.ic}
			testArch.AddImageTypes(&platform.X86{}, it)

			osc, err := osCustomizations(it, rpmmd.PackageSet{}, *tc.io, nil, tc.bpc)
			assert.NoError(t, err)
			assert.EqualValues(t, tc.expectedOsc.RHSMConfig, osc.RHSMConfig)
		})
	}
}

func TestPartitionTypeNotCrashing(t *testing.T) {
	it := &ImageType{}
	assert.Equal(t, it.PartitionType(), "")
}
