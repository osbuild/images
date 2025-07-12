package workload

import (
	"github.com/osbuild/images/pkg/rpmmd"
)

var _ = Workload(&WorkloadConf{})

type WorkloadConf struct {
	Packages         []string           `yaml:"packages"`
	Modules          []string           `yaml:"modules"`
	Repos            []rpmmd.RepoConfig `yaml:"repos"`
	EnabledServices  []string           `yaml:"enabled_services"`
	DisabledServices []string           `yaml:"disabled_services"`
	MaskedServices   []string           `yaml:"masked_services"`
}

func (wc *WorkloadConf) GetPackages() []string {
	return wc.Packages
}

func (wc *WorkloadConf) GetEnabledModules() []string {
	return wc.Modules
}
func (wc *WorkloadConf) GetRepos() []rpmmd.RepoConfig {
	return wc.Repos
}
func (wc *WorkloadConf) GetServices() []string {
	return wc.EnabledServices
}
func (wc *WorkloadConf) GetDisabledServices() []string {
	return wc.DisabledServices
}
func (wc *WorkloadConf) GetMaskedServices() []string {
	return wc.MaskedServices
}
