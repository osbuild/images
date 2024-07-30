package subscription

import (
	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/blueprint"
)

// The ImageOptions specify subscription-specific image options
// ServerUrl denotes the host to register the system with
// BaseUrl specifies the repository URL for DNF
type ImageOptions struct {
	Organization  string `json:"organization"`
	ActivationKey string `json:"activation_key"`
	ServerUrl     string `json:"server_url"`
	BaseUrl       string `json:"base_url"`
	Insights      bool   `json:"insights"`
	Rhc           bool   `json:"rhc"`
}

type RHSMStatus string

const (
	RHSMConfigWithSubscription RHSMStatus = "with-subscription"
	RHSMConfigNoSubscription   RHSMStatus = "no-subscription"
)

// Subscription Manager [rhsm] configuration
type SubManRHSMConfig struct {
	ManageRepos *bool
}

// Subscription Manager [rhsmcertd] configuration
type SubManRHSMCertdConfig struct {
	AutoRegistration *bool
}

// Subscription Manager 'rhsm.conf' configuration
type SubManConfig struct {
	Rhsm      SubManRHSMConfig
	Rhsmcertd SubManRHSMCertdConfig
}

type DNFPluginConfig struct {
	Enabled *bool
}

type SubManDNFPluginsConfig struct {
	ProductID           DNFPluginConfig
	SubscriptionManager DNFPluginConfig
}

type RHSMConfig struct {
	DnfPlugins SubManDNFPluginsConfig
	YumPlugins SubManDNFPluginsConfig
	SubMan     SubManConfig
}

// RHSMConfigFromBP creates a RHSMConfig from a blueprint RHSMCustomization
func RHSMConfigFromBP(bpRHSM *blueprint.RHSMCustomization) *RHSMConfig {
	if bpRHSM == nil || bpRHSM.Config == nil {
		return nil
	}

	c := &RHSMConfig{}

	if plugins := bpRHSM.Config.DNFPlugins; plugins != nil {
		if plugins.ProductID != nil && plugins.ProductID.Enabled != nil {
			c.DnfPlugins.ProductID.Enabled = common.ToPtr(*plugins.ProductID.Enabled)
		}
		if plugins.SubscriptionManager != nil && plugins.SubscriptionManager.Enabled != nil {
			c.DnfPlugins.SubscriptionManager.Enabled = common.ToPtr(*plugins.SubscriptionManager.Enabled)
		}
	}

	// NB: YUMPlugins are not exposed to end users as a customization

	if subMan := bpRHSM.Config.SubscriptionManager; subMan != nil {
		if subMan.RHSMConfig != nil && subMan.RHSMConfig.ManageRepos != nil {
			c.SubMan.Rhsm.ManageRepos = common.ToPtr(*subMan.RHSMConfig.ManageRepos)
		}
		if subMan.RHSMCertdConfig != nil && subMan.RHSMCertdConfig.AutoRegistration != nil {
			c.SubMan.Rhsmcertd.AutoRegistration = common.ToPtr(*subMan.RHSMCertdConfig.AutoRegistration)
		}
	}

	return c
}
