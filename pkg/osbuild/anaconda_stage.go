package osbuild

type AnacondaStageOptions struct {
	// Kickstart modules to enable
	KickstartModules []string `json:"kickstart-modules"`
}

func (AnacondaStageOptions) isStageOptions() {}

// Configure basic aspects of the Anaconda installer
func NewAnacondaStage(options *AnacondaStageOptions) *Stage {
	return &Stage{
		Type:    "org.osbuild.anaconda",
		Options: options,
	}
}

func defaultModuleStates() map[string]bool {
	return map[string]bool{
		"org.fedoraproject.Anaconda.Modules.Localization": false,
		"org.fedoraproject.Anaconda.Modules.Network":      true,
		"org.fedoraproject.Anaconda.Modules.Payloads":     true,
		"org.fedoraproject.Anaconda.Modules.Runtime":      false,
		"org.fedoraproject.Anaconda.Modules.Security":     false,
		"org.fedoraproject.Anaconda.Modules.Services":     false,
		"org.fedoraproject.Anaconda.Modules.Storage":      true,
		"org.fedoraproject.Anaconda.Modules.Subscription": false,
		"org.fedoraproject.Anaconda.Modules.Timezone":     false,
		"org.fedoraproject.Anaconda.Modules.Users":        false,
	}
}

func setModuleStates(states map[string]bool, enable, disable []string) {
	for _, modname := range enable {
		states[modname] = true
	}
	for _, modname := range disable {
		states[modname] = false
	}
}

func filterEnabledModules(moduleStates map[string]bool) []string {
	enabled := make([]string, 0, len(moduleStates))
	for modname, state := range moduleStates {
		if state {
			enabled = append(enabled, modname)
		}
	}
	return enabled
}

func NewAnacondaStageOptions(enableModules, disableModules []string) *AnacondaStageOptions {
	states := defaultModuleStates()
	setModuleStates(states, enableModules, disableModules)

	return &AnacondaStageOptions{
		KickstartModules: filterEnabledModules(states),
	}
}
