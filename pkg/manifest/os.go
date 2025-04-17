package manifest

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/internal/environment"
	"github.com/osbuild/images/internal/workload"
	"github.com/osbuild/images/pkg/arch"
	"github.com/osbuild/images/pkg/container"
	"github.com/osbuild/images/pkg/customizations/bootc"
	"github.com/osbuild/images/pkg/customizations/fsnode"
	"github.com/osbuild/images/pkg/customizations/oscap"
	"github.com/osbuild/images/pkg/customizations/shell"
	"github.com/osbuild/images/pkg/customizations/subscription"
	"github.com/osbuild/images/pkg/customizations/users"
	"github.com/osbuild/images/pkg/disk"
	"github.com/osbuild/images/pkg/osbuild"
	"github.com/osbuild/images/pkg/ostree"
	"github.com/osbuild/images/pkg/platform"
	"github.com/osbuild/images/pkg/rhsm/facts"
	"github.com/osbuild/images/pkg/rpmmd"
)

// OSCustomizations encapsulates all configuration applied to the base
// operating system independently of where and how it is integrated and what
// workload it is running.
// TODO: move out kernel/bootloader/cloud-init/... to other
//
//	abstractions, this should ideally only contain things that
//	can always be applied.
type OSCustomizations struct {

	// Packages to install in addition to the ones required by the pipeline.
	// These are the statically defined packages for the image type.
	BasePackages []string

	// Packages to exclude from the base package set. This is useful in
	// case of weak dependencies, comps groups, or where multiple packages
	// can satisfy a dependency. Must not conflict with the included base
	// package set.
	ExcludeBasePackages []string

	// Additional repos to install the base packages from.
	ExtraBaseRepos []rpmmd.RepoConfig

	// Containers to embed in the image (source specification)
	// TODO: move to workload
	Containers []container.SourceSpec

	// KernelName indicates that a kernel is installed, and names the kernel
	// package.
	KernelName string

	// KernelOptionsAppend are appended to the kernel commandline
	KernelOptionsAppend []string

	// KernelOptionsBootloader controls whether kernel command line options
	// should be specified in the bootloader grubenv configuration. Otherwise
	// they are specified in /etc/kernel/cmdline (default).
	//
	// NB: The kernel options need to be still specified in /etc/default/grub
	// under the GRUB_CMDLINE_LINUX variable. The reason is that it is used by
	// the 10_linux script executed by grub2-mkconfig to override the kernel
	// options in /etc/kernel/cmdline if the file has older timestamp than
	// /etc/default/grub.
	//
	// This should only be used for RHEL 8 and CentOS 8 images that use grub
	// (non s390x).  Newer releases (9+) should keep this disabled.
	KernelOptionsBootloader bool

	GPGKeyFiles      []string
	Language         string
	Keyboard         *string
	X11KeymapLayouts []string
	Hostname         string
	Timezone         string
	EnabledServices  []string
	DisabledServices []string
	MaskedServices   []string
	DefaultTarget    string

	// SELinux policy, when set it enables the labeling of the tree with the
	// selected profile
	SElinux string

	SELinuxForceRelabel *bool

	// Do not install documentation
	ExcludeDocs bool

	Groups []users.Group
	Users  []users.User

	ShellInit []shell.InitFile

	// TODO: drop osbuild types from the API
	Firewall             *osbuild.FirewallStageOptions
	Grub2Config          *osbuild.GRUB2Config
	Sysconfig            []*osbuild.SysconfigStageOptions
	SystemdLogind        []*osbuild.SystemdLogindStageOptions
	CloudInit            []*osbuild.CloudInitStageOptions
	Modprobe             []*osbuild.ModprobeStageOptions
	DracutConf           []*osbuild.DracutConfStageOptions
	SystemdDropin        []*osbuild.SystemdUnitStageOptions
	SystemdUnit          []*osbuild.SystemdUnitCreateStageOptions
	Authselect           *osbuild.AuthselectStageOptions
	SELinuxConfig        *osbuild.SELinuxConfigStageOptions
	Tuned                *osbuild.TunedStageOptions
	Tmpfilesd            []*osbuild.TmpfilesdStageOptions
	PamLimitsConf        []*osbuild.PamLimitsConfStageOptions
	Sysctld              []*osbuild.SysctldStageOptions
	DNFConfig            []*osbuild.DNFConfigStageOptions
	DNFAutomaticConfig   *osbuild.DNFAutomaticConfigStageOptions
	YUMConfig            *osbuild.YumConfigStageOptions
	YUMRepos             []*osbuild.YumReposStageOptions
	SshdConfig           *osbuild.SshdConfigStageOptions
	GCPGuestAgentConfig  *osbuild.GcpGuestAgentConfigOptions
	AuthConfig           *osbuild.AuthconfigStageOptions
	PwQuality            *osbuild.PwqualityConfStageOptions
	ChronyConfig         *osbuild.ChronyStageOptions
	WAAgentConfig        *osbuild.WAAgentConfStageOptions
	UdevRules            *osbuild.UdevRulesStageOptions
	WSLConfig            *osbuild.WSLConfStageOptions
	InsightsClientConfig *osbuild.InsightsClientConfigStageOptions
	Presets              []osbuild.Preset
	ContainersStorage    *string

	// OpenSCAP config
	OpenSCAPRemediationConfig *oscap.RemediationConfig

	Subscription *subscription.ImageOptions
	// The final RHSM config to be applied to the image
	RHSMConfig *subscription.RHSMConfig
	RHSMFacts  *facts.ImageOptions

	// Custom directories to create in the image. The stages for the
	// directories defined here are always added at the end of the pipeline.
	Directories []*fsnode.Directory

	// Custom files to create in the image. The stages for the files defined
	// here are always added at the end of the pipeline.
	Files []*fsnode.File

	CACerts []string

	FIPS bool

	// NoBLS configures the image bootloader with traditional menu entries
	// instead of BLS. Required for legacy systems like RHEL 7.
	NoBLS bool

	// InstallWeakDeps enables installation of weak dependencies for packages
	// that are statically defined for the pipeline.
	// Defaults to True.
	InstallWeakDeps bool

	// Determines if the machine id should be set to "uninitialized" which allows
	// "ConditionFirstBoot" to work in systemd
	MachineIdUninitialized bool

	// MountUnits creates systemd .mount units to describe the filesystem
	// instead of writing to /etc/fstab
	MountUnits bool
}

// OS represents the filesystem tree of the target image. This roughly
// corresponds to the root filesystem once an instance of the image is running.
type OS struct {
	Base

	// OSCustomizations to apply to the base OS
	OSCustomizations OSCustomizations

	// Environment the system will run in
	Environment environment.Environment
	// Workload to install on top of the base system
	Workload workload.Workload
	// Ref of ostree commit (optional). If empty the tree cannot be in an ostree commit
	OSTreeRef string
	// OSTreeParent source spec (optional). If nil the new commit (if
	// applicable) will have no parent
	OSTreeParent *ostree.SourceSpec
	// Enabling Bootupd runs bootupctl generate-update-metadata in the tree to
	// transform /usr/lib/ostree-boot into a bootupd-compatible update
	// payload. Only works with ostree-based images.
	Bootupd bool

	// Add a bootc config file to the image (for bootable containers)
	BootcConfig *bootc.Config

	// Partition table, if nil the tree cannot be put on a partitioned disk
	PartitionTable *disk.PartitionTable

	// content-related fields
	repos            []rpmmd.RepoConfig
	packageSpecs     []rpmmd.PackageSpec
	moduleSpecs      []rpmmd.ModuleSpec
	containerSpecs   []container.Spec
	ostreeParentSpec *ostree.CommitSpec

	platform  platform.Platform
	kernelVer string

	OSProduct string
	OSVersion string
	OSNick    string

	inlineData []string
}

// NewOS creates a new OS pipeline. build is the build pipeline to use for
// building the OS pipeline. platform is the target platform for the final
// image. repos are the repositories to install RPMs from.
func NewOS(buildPipeline Build, platform platform.Platform, repos []rpmmd.RepoConfig) *OS {
	name := "os"
	p := &OS{
		Base:     NewBase(name, buildPipeline),
		repos:    filterRepos(repos, name),
		platform: platform,
	}
	buildPipeline.addDependent(p)
	return p
}

func (p *OS) getPackageSetChain(Distro) []rpmmd.PackageSet {
	platformPackages := p.platform.GetPackages()

	var environmentPackages []string
	if p.Environment != nil {
		environmentPackages = p.Environment.GetPackages()
	}

	var partitionTablePackages []string
	if p.PartitionTable != nil {
		partitionTablePackages = p.PartitionTable.GetBuildPackages()
	}

	if p.OSCustomizations.KernelName != "" {
		// kernel is considered part of the platform package set
		platformPackages = append(platformPackages, p.OSCustomizations.KernelName)
	}

	customizationPackages := make([]string, 0)
	if p.OSCustomizations.ChronyConfig != nil {
		customizationPackages = append(customizationPackages, "chrony")
	}

	if p.OSCustomizations.SElinux != "" {
		customizationPackages = append(customizationPackages, fmt.Sprintf("selinux-policy-%s", p.OSCustomizations.SElinux))
	}

	if p.OSCustomizations.OpenSCAPRemediationConfig != nil {
		customizationPackages = append(customizationPackages, "openscap-scanner", "scap-security-guide", "xz")
	}

	// Make sure the right packages are included for subscriptions
	// rhc always uses insights, and depends on subscription-manager
	// non-rhc uses subscription-manager and optionally includes Insights
	if p.OSCustomizations.Subscription != nil {
		customizationPackages = append(customizationPackages, "subscription-manager")
		if p.OSCustomizations.Subscription.Rhc {
			customizationPackages = append(customizationPackages, "rhc", "insights-client", "rhc-worker-playbook")
		} else if p.OSCustomizations.Subscription.Insights {
			customizationPackages = append(customizationPackages, "insights-client")
		}
	}

	if len(p.OSCustomizations.Users) > 0 {
		// org.osbuild.users runs useradd, usermod, passwd, and
		// mkhomedir_helper in the os tree using chroot. Most image types
		// should already have the required packages, but some minimal image
		// types, like 'tar' don't, so let's add them for the stage to run and
		// to enable user management in the image.
		customizationPackages = append(customizationPackages, "shadow-utils", "pam", "passwd")

	}

	if p.OSCustomizations.Firewall != nil {
		// Make sure firewalld is available in the image.
		// org.osbuild.firewall runs 'firewall-offline-cmd' in the os tree
		// using chroot, so we don't need a build package for this.
		customizationPackages = append(customizationPackages, "firewalld")
	}

	osRepos := append(p.repos, p.OSCustomizations.ExtraBaseRepos...)

	// merge all package lists for the pipeline
	baseOSPackages := make([]string, 0)
	baseOSPackages = append(baseOSPackages, platformPackages...)
	baseOSPackages = append(baseOSPackages, environmentPackages...)
	baseOSPackages = append(baseOSPackages, partitionTablePackages...)
	baseOSPackages = append(baseOSPackages, p.OSCustomizations.BasePackages...)

	chain := []rpmmd.PackageSet{
		{
			Include:         baseOSPackages,
			Exclude:         p.OSCustomizations.ExcludeBasePackages,
			Repositories:    osRepos,
			InstallWeakDeps: p.OSCustomizations.InstallWeakDeps,
		},
		{
			// Depsolve customization packages separately to avoid conflicts with base
			// package exclusion.
			// See https://github.com/osbuild/images/issues/1323
			Include:      customizationPackages,
			Repositories: osRepos,
			// Although 'false' is the default value, set it explicitly to make
			// it visible that we are not adding weak dependencies.
			InstallWeakDeps: false,
		},
	}

	if p.Workload != nil {
		workloadPackages := p.Workload.GetPackages()
		if len(workloadPackages) > 0 {
			ps := rpmmd.PackageSet{
				Include:      workloadPackages,
				Repositories: append(osRepos, p.Workload.GetRepos()...),
				// Although 'false' is the default value, set it explicitly to make
				// it visible that we are not adding weak dependencies.
				InstallWeakDeps: false,
			}

			workloadModules := p.Workload.GetEnabledModules()
			if len(workloadModules) > 0 {
				ps.EnabledModules = workloadModules
			}
			chain = append(chain, ps)
		}
	}

	return chain
}

func (p *OS) getContainerSources() []container.SourceSpec {
	return p.OSCustomizations.Containers
}

func tomlPkgsFor(distro Distro) []string {
	switch distro {
	case DISTRO_EL7:
		// nothing needs toml in rhel7
		panic("no support for toml on rhel7")
	case DISTRO_EL8:
		// deprecated, needed for backwards compatibility (EL8 manifests)
		return []string{"python3-pytoml"}
	case DISTRO_EL9:
		// older unmaintained lib, needed for backwards compatibility
		return []string{"python3-toml"}
	default:
		// No extra package needed for reading, on rhel10 and
		// fedora as stdlib has "tomlib" but we need tomli-w
		// for writing
		return []string{"python3-tomli-w"}
	}
}

func (p *OS) getBuildPackages(distro Distro) []string {
	packages := p.platform.GetBuildPackages()
	if p.PartitionTable != nil {
		packages = append(packages, p.PartitionTable.GetBuildPackages()...)
	}
	packages = append(packages, "rpm")
	if p.OSTreeRef != "" {
		packages = append(packages, "rpm-ostree")
	}
	if p.OSCustomizations.SElinux != "" {
		packages = append(packages, "policycoreutils", fmt.Sprintf("selinux-policy-%s", p.OSCustomizations.SElinux))
	}
	if len(p.OSCustomizations.CloudInit) > 0 {
		switch distro {
		case DISTRO_EL7:
			packages = append(packages, "python3-PyYAML")
		default:
			packages = append(packages, "python3-pyyaml")
		}
	}
	if len(p.OSCustomizations.DNFConfig) > 0 || p.OSCustomizations.RHSMConfig != nil || p.OSCustomizations.WSLConfig != nil {
		packages = append(packages, "python3-iniparse")
	}

	if len(p.OSCustomizations.Containers) > 0 {
		if p.OSCustomizations.ContainersStorage != nil {
			packages = append(packages, tomlPkgsFor(distro)...)
		}
		packages = append(packages, "skopeo")
	}

	if p.OSCustomizations.OpenSCAPRemediationConfig != nil && p.OSCustomizations.OpenSCAPRemediationConfig.TailoringConfig != nil {
		packages = append(packages, "openscap-utils")
	}

	if p.BootcConfig != nil {
		packages = append(packages, tomlPkgsFor(distro)...)
	}

	return packages
}

func (p *OS) getOSTreeCommitSources() []ostree.SourceSpec {
	if p.OSTreeParent == nil {
		return nil
	}

	return []ostree.SourceSpec{
		*p.OSTreeParent,
	}
}

func (p *OS) getOSTreeCommits() []ostree.CommitSpec {
	if p.ostreeParentSpec == nil {
		return nil
	}
	return []ostree.CommitSpec{*p.ostreeParentSpec}
}

func (p *OS) getPackageSpecs() []rpmmd.PackageSpec {
	return p.packageSpecs
}

func (p *OS) getContainerSpecs() []container.Spec {
	return p.containerSpecs
}

func (p *OS) serializeStart(inputs Inputs) {
	if len(p.packageSpecs) > 0 {
		panic("double call to serializeStart()")
	}

	p.packageSpecs = inputs.Depsolved.Packages
	p.moduleSpecs = inputs.Depsolved.Modules
	p.containerSpecs = inputs.Containers
	if len(inputs.Commits) > 0 {
		if len(inputs.Commits) > 1 {
			panic("pipeline supports at most one ostree commit")
		}
		p.ostreeParentSpec = &inputs.Commits[0]
	}

	if p.OSCustomizations.KernelName != "" {
		p.kernelVer = rpmmd.GetVerStrFromPackageSpecListPanic(p.packageSpecs, p.OSCustomizations.KernelName)
	}

	p.repos = append(p.repos, inputs.Depsolved.Repos...)
}

func (p *OS) serializeEnd() {
	if len(p.packageSpecs) == 0 {
		panic("serializeEnd() call when serialization not in progress")
	}
	p.kernelVer = ""
	p.packageSpecs = nil
	p.containerSpecs = nil
	p.ostreeParentSpec = nil
}

func (p *OS) serialize() osbuild.Pipeline {
	if len(p.packageSpecs) == 0 {
		panic("serialization not started")
	}

	pipeline := p.Base.serialize()

	if p.ostreeParentSpec != nil {
		pipeline.AddStage(osbuild.NewOSTreePasswdStage("org.osbuild.source", p.ostreeParentSpec.Checksum))
	}

	// collect all repos for this pipeline to create the repository options
	allRepos := append(p.repos, p.OSCustomizations.ExtraBaseRepos...)
	if p.Workload != nil {
		allRepos = append(allRepos, p.Workload.GetRepos()...)
	}
	rpmOptions := osbuild.NewRPMStageOptions(allRepos)
	if p.OSCustomizations.ExcludeDocs {
		if rpmOptions.Exclude == nil {
			rpmOptions.Exclude = &osbuild.Exclude{}
		}
		rpmOptions.Exclude.Docs = true
	}
	rpmOptions.GPGKeysFromTree = p.OSCustomizations.GPGKeyFiles
	if p.OSTreeRef != "" {
		rpmOptions.OSTreeBooted = common.ToPtr(true)
		rpmOptions.DBPath = "/usr/share/rpm"
		// The dracut-config-rescue package will create a rescue kernel when
		// installed. This creates an issue with ostree-based images because
		// rpm-ostree requires that only one kernel exists in the image.
		// Disabling dracut for ostree-based systems resolves this issue.
		// Dracut will be run by rpm-ostree itself while composing the image.
		// https://github.com/osbuild/images/issues/624
		rpmOptions.DisableDracut = true
	}
	pipeline.AddStage(osbuild.NewRPMStage(rpmOptions, osbuild.NewRpmStageSourceFilesInputs(p.packageSpecs)))

	if !p.OSCustomizations.NoBLS {
		// If the /boot is on a separate partition, the prefix for the BLS stage must be ""
		if p.PartitionTable == nil || p.PartitionTable.FindMountable("/boot") == nil {
			pipeline.AddStage(osbuild.NewFixBLSStage(&osbuild.FixBLSStageOptions{}))
		} else {
			pipeline.AddStage(osbuild.NewFixBLSStage(&osbuild.FixBLSStageOptions{Prefix: common.ToPtr("")}))
		}
	}

	if len(p.containerSpecs) > 0 {
		var storagePath string
		if containerStore := p.OSCustomizations.ContainersStorage; containerStore != nil {
			storagePath = *containerStore
		}

		for _, stage := range osbuild.GenContainerStorageStages(storagePath, p.containerSpecs) {
			pipeline.AddStage(stage)
		}
	}

	if p.OSCustomizations.Language != "" {
		pipeline.AddStage(osbuild.NewLocaleStage(&osbuild.LocaleStageOptions{Language: p.OSCustomizations.Language}))
	}

	if p.OSCustomizations.Keyboard != nil {
		keymapOptions := &osbuild.KeymapStageOptions{Keymap: *p.OSCustomizations.Keyboard}
		if len(p.OSCustomizations.X11KeymapLayouts) > 0 {
			keymapOptions.X11Keymap = &osbuild.X11KeymapOptions{Layouts: p.OSCustomizations.X11KeymapLayouts}
		}
		pipeline.AddStage(osbuild.NewKeymapStage(keymapOptions))
	}

	if p.OSCustomizations.Hostname != "" {
		pipeline.AddStage(osbuild.NewHostnameStage(&osbuild.HostnameStageOptions{Hostname: p.OSCustomizations.Hostname}))
	}

	if p.OSCustomizations.Timezone != "" {
		pipeline.AddStage(osbuild.NewTimezoneStage(&osbuild.TimezoneStageOptions{Zone: p.OSCustomizations.Timezone}))
	}

	if p.OSCustomizations.ChronyConfig != nil {
		pipeline.AddStage(osbuild.NewChronyStage(p.OSCustomizations.ChronyConfig))
	}

	if len(p.OSCustomizations.Groups) > 0 {
		pipeline.AddStage(osbuild.GenGroupsStage(p.OSCustomizations.Groups))
	}

	if len(p.OSCustomizations.Users) > 0 {
		if p.OSTreeRef != "" {
			// for ostree, writing the key during user creation is
			// redundant and can cause issues so create users without keys
			// and write them on first boot
			usersStageSansKeys, err := osbuild.GenUsersStage(p.OSCustomizations.Users, true)
			if err != nil {
				// TODO: move encryption into weldr
				panic("password encryption failed")
			}
			pipeline.AddStage(usersStageSansKeys)
			pipeline.AddStage(osbuild.NewFirstBootStage(usersFirstBootOptions(p.OSCustomizations.Users)))
		} else {
			usersStage, err := osbuild.GenUsersStage(p.OSCustomizations.Users, false)
			if err != nil {
				// TODO: move encryption into weldr
				panic("password encryption failed")
			}
			pipeline.AddStage(usersStage)
		}
	}

	if p.OSCustomizations.Firewall != nil {
		pipeline.AddStage(osbuild.NewFirewallStage(p.OSCustomizations.Firewall))
	}

	for _, sysconfigConfig := range p.OSCustomizations.Sysconfig {
		pipeline.AddStage(osbuild.NewSysconfigStage(sysconfigConfig))
	}

	for _, systemdLogindConfig := range p.OSCustomizations.SystemdLogind {
		pipeline.AddStage(osbuild.NewSystemdLogindStage(systemdLogindConfig))
	}

	for _, cloudInitConfig := range p.OSCustomizations.CloudInit {
		pipeline.AddStage(osbuild.NewCloudInitStage(cloudInitConfig))
	}

	for _, modprobeConfig := range p.OSCustomizations.Modprobe {
		pipeline.AddStage(osbuild.NewModprobeStage(modprobeConfig))
	}

	for _, dracutConfConfig := range p.OSCustomizations.DracutConf {
		pipeline.AddStage(osbuild.NewDracutConfStage(dracutConfConfig))
	}

	for _, systemdUnitConfig := range p.OSCustomizations.SystemdDropin {
		pipeline.AddStage(osbuild.NewSystemdUnitStage(systemdUnitConfig))
	}

	for _, systemdUnitCreateConfig := range p.OSCustomizations.SystemdUnit {
		pipeline.AddStage(osbuild.NewSystemdUnitCreateStage(systemdUnitCreateConfig))
	}

	if p.OSCustomizations.Authselect != nil {
		pipeline.AddStage(osbuild.NewAuthselectStage(p.OSCustomizations.Authselect))
	}

	if p.OSCustomizations.SELinuxConfig != nil {
		pipeline.AddStage(osbuild.NewSELinuxConfigStage(p.OSCustomizations.SELinuxConfig))
	}

	if p.OSCustomizations.Tuned != nil {
		pipeline.AddStage(osbuild.NewTunedStage(p.OSCustomizations.Tuned))
	}

	for _, tmpfilesdConfig := range p.OSCustomizations.Tmpfilesd {
		pipeline.AddStage(osbuild.NewTmpfilesdStage(tmpfilesdConfig))
	}

	for _, pamLimitsConfConfig := range p.OSCustomizations.PamLimitsConf {
		pipeline.AddStage(osbuild.NewPamLimitsConfStage(pamLimitsConfConfig))
	}

	for _, sysctldConfig := range p.OSCustomizations.Sysctld {
		pipeline.AddStage(osbuild.NewSysctldStage(sysctldConfig))
	}

	for _, dnfConfig := range p.OSCustomizations.DNFConfig {
		pipeline.AddStage(osbuild.NewDNFConfigStage(dnfConfig))
	}

	if p.OSCustomizations.DNFAutomaticConfig != nil {
		pipeline.AddStage(osbuild.NewDNFAutomaticConfigStage(p.OSCustomizations.DNFAutomaticConfig))
	}

	for _, yumRepo := range p.OSCustomizations.YUMRepos {
		pipeline.AddStage(osbuild.NewYumReposStage(yumRepo))
	}

	if p.OSCustomizations.YUMConfig != nil {
		pipeline.AddStage(osbuild.NewYumConfigStage(p.OSCustomizations.YUMConfig))
	}

	if p.OSCustomizations.GCPGuestAgentConfig != nil {
		pipeline.AddStage(osbuild.NewGcpGuestAgentConfigStage(p.OSCustomizations.GCPGuestAgentConfig))
	}

	if p.OSCustomizations.SshdConfig != nil {
		pipeline.AddStage(osbuild.NewSshdConfigStage(p.OSCustomizations.SshdConfig))
	}

	if p.OSCustomizations.InsightsClientConfig != nil {
		pipeline.AddStage(osbuild.NewInsightsClientConfigStage(p.OSCustomizations.InsightsClientConfig))
	}

	if p.OSCustomizations.AuthConfig != nil {
		pipeline.AddStage(osbuild.NewAuthconfigStage(p.OSCustomizations.AuthConfig))
	}

	if p.OSCustomizations.PwQuality != nil {
		pipeline.AddStage(osbuild.NewPwqualityConfStage(p.OSCustomizations.PwQuality))
	}

	if p.OSCustomizations.Subscription != nil {
		subStage, subDirs, subFiles, subServices, err := subscriptionService(*p.OSCustomizations.Subscription, &subscriptionServiceOptions{InsightsOnBoot: p.OSTreeRef != ""})
		if err != nil {
			panic(err)
		}
		pipeline.AddStage(subStage)
		p.OSCustomizations.Directories = append(p.OSCustomizations.Directories, subDirs...)
		p.addInlineDataAndStages(&pipeline, subFiles)
		p.OSCustomizations.EnabledServices = append(p.OSCustomizations.EnabledServices, subServices...)
	}

	if p.OSCustomizations.RHSMConfig != nil {
		pipeline.AddStage(osbuild.NewRHSMStage(osbuild.NewRHSMStageOptions(p.OSCustomizations.RHSMConfig)))
	}

	if p.OSCustomizations.WAAgentConfig != nil {
		pipeline.AddStage(osbuild.NewWAAgentConfStage(p.OSCustomizations.WAAgentConfig))
	}

	if p.OSCustomizations.UdevRules != nil {
		pipeline.AddStage(osbuild.NewUdevRulesStage(p.OSCustomizations.UdevRules))
	}

	if pt := p.PartitionTable; pt != nil {
		rootUUID, kernelOptions, err := osbuild.GenImageKernelOptions(p.PartitionTable, p.OSCustomizations.MountUnits)
		if err != nil {
			panic(err)
		}
		kernelOptions = append(kernelOptions, p.OSCustomizations.KernelOptionsAppend...)

		if p.OSCustomizations.FIPS {
			kernelOptions = append(kernelOptions, osbuild.GenFIPSKernelOptions(p.PartitionTable)...)
			pipeline.AddStage(osbuild.NewDracutStage(&osbuild.DracutStageOptions{
				Kernel:     []string{p.kernelVer},
				AddModules: []string{"fips"},
			}))
		}

		fsCfgStages, err := filesystemConfigStages(pt, p.OSCustomizations.MountUnits)
		if err != nil {
			panic(err)
		}
		pipeline.AddStages(fsCfgStages...)

		var bootloader *osbuild.Stage
		switch p.platform.GetArch() {
		case arch.ARCH_S390X:
			bootloader = osbuild.NewZiplStage(new(osbuild.ZiplStageOptions))
		default:
			if p.OSCustomizations.NoBLS {
				// BLS entries not supported: use grub2.legacy
				id := "76a22bf4-f153-4541-b6c7-0332c0dfaeac"
				product := osbuild.GRUB2Product{
					Name:    p.OSProduct,
					Version: p.OSVersion,
					Nick:    p.OSNick,
				}

				_, err := rpmmd.GetVerStrFromPackageSpecList(p.packageSpecs, "dracut-config-rescue")
				hasRescue := err == nil
				bootloader = osbuild.NewGrub2LegacyStage(
					osbuild.NewGrub2LegacyStageOptions(
						p.OSCustomizations.Grub2Config,
						p.PartitionTable,
						kernelOptions,
						p.platform.GetBIOSPlatform(),
						p.platform.GetUEFIVendor(),
						osbuild.MakeGrub2MenuEntries(id, p.kernelVer, product, hasRescue),
					),
				)
			} else {
				options := osbuild.NewGrub2StageOptions(pt,
					strings.Join(kernelOptions, " "),
					p.kernelVer,
					p.platform.GetUEFIVendor() != "",
					p.platform.GetBIOSPlatform(),
					p.platform.GetUEFIVendor(), false)
				if cfg := p.OSCustomizations.Grub2Config; cfg != nil {
					// TODO: don't store Grub2Config in OSPipeline, making the overrides unnecessary
					// grub2.Config.Default is owned and set by `NewGrub2StageOptionsUnified`
					// and thus we need to preserve it
					if options.Config != nil {
						cfg.Default = options.Config.Default
					}

					options.Config = cfg
				}
				if p.OSCustomizations.KernelOptionsBootloader {
					options.WriteCmdLine = nil
					if options.UEFI != nil {
						options.UEFI.Unified = false
					}
				}
				bootloader = osbuild.NewGRUB2Stage(options)
			}
		}

		pipeline.AddStage(bootloader)

		if !p.OSCustomizations.KernelOptionsBootloader || p.platform.GetArch() == arch.ARCH_S390X {
			pipeline = prependKernelCmdlineStage(pipeline, rootUUID, kernelOptions)
		}
	}

	if p.OSCustomizations.RHSMFacts != nil {
		rhsmFacts := osbuild.RHSMFacts{
			ApiType: p.OSCustomizations.RHSMFacts.APIType.String(),
		}

		if p.OSCustomizations.RHSMFacts.OpenSCAPProfileID != "" {
			rhsmFacts.OpenSCAPProfileID = p.OSCustomizations.RHSMFacts.OpenSCAPProfileID
		}

		if p.OSCustomizations.RHSMFacts.CompliancePolicyID != uuid.Nil {
			rhsmFacts.CompliancePolicyID = p.OSCustomizations.RHSMFacts.CompliancePolicyID.String()
		}

		pipeline.AddStage(osbuild.NewRHSMFactsStage(&osbuild.RHSMFactsStageOptions{
			Facts: rhsmFacts,
		}))
	}

	if p.OSTreeRef != "" {
		pipeline.AddStage(osbuild.NewSystemdJournaldStage(
			&osbuild.SystemdJournaldStageOptions{
				Filename: "10-persistent.conf",
				Config: osbuild.SystemdJournaldConfigDropin{
					Journal: osbuild.SystemdJournaldConfigJournalSection{
						Storage: osbuild.StoragePresistent,
					},
				},
			}))
	}

	// write modularity related configuration files
	if len(p.moduleSpecs) > 0 {
		pipeline.AddStages(osbuild.GenDNFModuleConfigStages(p.moduleSpecs)...)

		var failsafeFiles []*fsnode.File

		// the failsafe file is a blob of YAML returned directly from the depsolver,
		// we write them as 'normal files' without a special stage
		for _, module := range p.moduleSpecs {
			moduleFailsafeFile, err := fsnode.NewFile(module.FailsafeFile.Path, nil, nil, nil, []byte(module.FailsafeFile.Data))

			if err != nil {
				panic("failed to create module failsafe file")
			}

			failsafeFiles = append(failsafeFiles, moduleFailsafeFile)
		}

		failsafeDir, err := fsnode.NewDirectory("/var/lib/dnf/modulefailsafe", nil, nil, nil, true)
		if err != nil {
			panic("failed to create module failsafe directory")
		}

		pipeline.AddStages(osbuild.GenDirectoryNodesStages([]*fsnode.Directory{failsafeDir})...)
		p.addInlineDataAndStages(&pipeline, failsafeFiles)
	}

	// First create custom directories, because some of the custom files may depend on them
	if len(p.OSCustomizations.Directories) > 0 {
		pipeline.AddStages(osbuild.GenDirectoryNodesStages(p.OSCustomizations.Directories)...)
	}

	// Custom files (from the blueprint) are often used to create systemd
	// units, so let's make sure they get created before the systemd stage that
	// will probably want to enable them
	if len(p.OSCustomizations.Files) > 0 {
		p.addInlineDataAndStages(&pipeline, p.OSCustomizations.Files)
	}

	enabledServices := []string{}
	disabledServices := []string{}
	maskedServices := []string{}
	enabledServices = append(enabledServices, p.OSCustomizations.EnabledServices...)
	disabledServices = append(disabledServices, p.OSCustomizations.DisabledServices...)
	maskedServices = append(maskedServices, p.OSCustomizations.MaskedServices...)
	if p.Environment != nil {
		enabledServices = append(enabledServices, p.Environment.GetServices()...)
	}
	if p.Workload != nil {
		enabledServices = append(enabledServices, p.Workload.GetServices()...)
		disabledServices = append(disabledServices, p.Workload.GetDisabledServices()...)
	}
	if len(enabledServices) != 0 ||
		len(disabledServices) != 0 ||
		len(maskedServices) != 0 || p.OSCustomizations.DefaultTarget != "" {
		pipeline.AddStage(osbuild.NewSystemdStage(&osbuild.SystemdStageOptions{
			EnabledServices:  enabledServices,
			DisabledServices: disabledServices,
			MaskedServices:   maskedServices,
			DefaultTarget:    p.OSCustomizations.DefaultTarget,
		}))
	}
	if len(p.OSCustomizations.ShellInit) > 0 {
		pipeline.AddStage(osbuild.GenShellInitStage(p.OSCustomizations.ShellInit))
	}

	if p.OSCustomizations.WSLConfig != nil {
		pipeline.AddStage(osbuild.NewWSLConfStage(p.OSCustomizations.WSLConfig))
	}

	if p.OSCustomizations.FIPS {
		pipeline.AddStages(osbuild.GenFIPSStages()...)
		p.addInlineDataAndStages(&pipeline, osbuild.GenFIPSFiles())
	}

	// NOTE: We need to run the OpenSCAP stages as the last stage before SELinux
	// since the remediation may change file permissions and other aspects of the
	// hardened image
	if remediationConfig := p.OSCustomizations.OpenSCAPRemediationConfig; remediationConfig != nil {
		if remediationConfig.TailoringConfig != nil {
			tailoringStageOpts := osbuild.NewOscapAutotailorStageOptions(remediationConfig)
			pipeline.AddStage(osbuild.NewOscapAutotailorStage(tailoringStageOpts))
		}
		remediationStageOpts := osbuild.NewOscapRemediationStageOptions(oscap.DataDir, remediationConfig)
		pipeline.AddStage(osbuild.NewOscapRemediationStage(remediationStageOpts))
	}

	if len(p.OSCustomizations.Presets) != 0 {
		pipeline.AddStage(osbuild.NewSystemdPresetStage(&osbuild.SystemdPresetStageOptions{
			Presets: p.OSCustomizations.Presets,
		}))
	}

	if len(p.OSCustomizations.CACerts) > 0 {
		for _, cc := range p.OSCustomizations.CACerts {
			files, err := osbuild.NewCAFileNodes(cc)
			if err != nil {
				panic(err.Error())
			}

			if len(files) > 0 {
				p.addInlineDataAndStages(&pipeline, files)
			}
		}
		pipeline.AddStage(osbuild.NewUpdateCATrustStage())
	}

	if p.OSCustomizations.MachineIdUninitialized {
		pipeline.AddStage(osbuild.NewMachineIdStage(&osbuild.MachineIdStageOptions{
			FirstBoot: osbuild.MachineIdFirstBootYes,
		}))
	}

	if p.OSCustomizations.SElinux != "" {
		pipeline.AddStage(osbuild.NewSELinuxStage(&osbuild.SELinuxStageOptions{
			FileContexts:     fmt.Sprintf("etc/selinux/%s/contexts/files/file_contexts", p.OSCustomizations.SElinux),
			ForceAutorelabel: p.OSCustomizations.SELinuxForceRelabel,
		}))
	}

	if p.OSTreeRef != "" {
		pipeline.AddStage(osbuild.NewOSTreePrepTreeStage(&osbuild.OSTreePrepTreeStageOptions{
			EtcGroupMembers: []string{
				// NOTE: We may want to make this configurable.
				"wheel", "docker",
			},
		}))
		if p.Bootupd {
			pipeline.AddStage(osbuild.NewBootupdGenMetadataStage())
		}
		if cfg := p.BootcConfig; cfg != nil {
			pipeline.AddStage(osbuild.NewBootcInstallConfigStage(
				osbuild.GenBootcInstallOptions(cfg.Filename, cfg.RootFilesystemType),
			))
		}
	} else {
		if p.Bootupd {
			panic("bootupd is only compatible with ostree-based images, this is a programming error")
		}
		if p.BootcConfig != nil {
			panic("bootc config is only compatible with ostree-based images, this is a programming error")
		}
	}

	return pipeline
}

func prependKernelCmdlineStage(pipeline osbuild.Pipeline, rootUUID string, kernelOptions []string) osbuild.Pipeline {
	kernelStage := osbuild.NewKernelCmdlineStage(osbuild.NewKernelCmdlineStageOptions(rootUUID, strings.Join(kernelOptions, " ")))
	pipeline.Stages = append([]*osbuild.Stage{kernelStage}, pipeline.Stages...)
	return pipeline
}

func usersFirstBootOptions(users []users.User) *osbuild.FirstBootStageOptions {
	cmds := make([]string, 0, 3*len(users)+2)
	// workaround for creating authorized_keys file for user
	// need to special case the root user, which has its home in a different place
	varhome := filepath.Join("/var", "home")
	roothome := filepath.Join("/var", "roothome")

	for _, user := range users {
		if user.Key != nil {
			var home string

			if user.Name == "root" {
				home = roothome
			} else {
				home = filepath.Join(varhome, user.Name)
			}

			sshdir := filepath.Join(home, ".ssh")

			cmds = append(cmds, fmt.Sprintf("mkdir -p %s", sshdir))
			cmds = append(cmds, fmt.Sprintf("sh -c 'echo %q >> %q'", *user.Key, filepath.Join(sshdir, "authorized_keys")))
			cmds = append(cmds, fmt.Sprintf("chown %s:%s -Rc %s", user.Name, user.Name, sshdir))
		}
	}
	cmds = append(cmds, fmt.Sprintf("restorecon -rvF %s", varhome))
	cmds = append(cmds, fmt.Sprintf("restorecon -rvF %s", roothome))

	options := &osbuild.FirstBootStageOptions{
		Commands:       cmds,
		WaitForNetwork: false,
	}

	return options
}

func (p *OS) Platform() platform.Platform {
	return p.platform
}

func (p *OS) getInline() []string {
	return p.inlineData
}

// addInlineDataAndStages generates stages for creating files and adds them to
// the pipeline. It also adds their data to the inlineData for the pipeline so
// that the appropriate sources are created.
func (p *OS) addInlineDataAndStages(pipeline *osbuild.Pipeline, files []*fsnode.File) {
	pipeline.AddStages(osbuild.GenFileNodesStages(files)...)

	for _, file := range files {
		p.inlineData = append(p.inlineData, string(file.Data()))
	}
}

func (p *OS) fileRefs() []string {
	var fileRefs []string

	for _, file := range p.OSCustomizations.Files {
		if file.Ref() != "" {
			fileRefs = append(fileRefs, file.Ref())
		}
	}

	return fileRefs
}
