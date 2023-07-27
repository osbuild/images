package workload

type Custom struct {
	BaseWorkload
	PackagesInclude  []string
	Services         []string
	DisabledServices []string
}

func (p *Custom) GetPackagesInclude() []string {
	return p.PackagesInclude
}

func (p *Custom) GetServices() []string {
	return p.Services
}

// TODO: Does this belong here? What kind of workload requires
// services to be disabled?
func (p *Custom) GetDisabledServices() []string {
	return p.DisabledServices
}
