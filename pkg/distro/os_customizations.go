package distro

import (
	"fmt"

	"github.com/osbuild/images/pkg/blueprint"
	"github.com/osbuild/images/pkg/container"
	"github.com/osbuild/images/pkg/customizations/fsnode"
	"github.com/osbuild/images/pkg/customizations/oscap"
	"github.com/osbuild/images/pkg/customizations/subscription"
	"github.com/osbuild/images/pkg/customizations/users"
	"github.com/osbuild/images/pkg/manifest"
	"github.com/osbuild/images/pkg/osbuild"
	"github.com/osbuild/images/pkg/rpmmd"
)

func OsCustomizations(t *ImageTypeConfig, osPackageSet rpmmd.PackageSet, options ImageOptions, containers []container.SourceSpec, c *blueprint.Customizations) (manifest.OSCustomizations, error) {

	imageConfig := t.DefaultImageConfig

	osc := manifest.OSCustomizations{}

	if t.Bootable || t.RpmOstree {
		// XXX: the default kernel name sould come from
		// DistroConfig first, it is currently hardcoded in
		// blueprints as a fallback
		osc.KernelName = c.GetKernel().Name

		var kernelOptions []string
		if len(t.KernelOptions) > 0 {
			kernelOptions = append(kernelOptions, t.KernelOptions...)
		}
		if bpKernel := c.GetKernel(); bpKernel.Append != "" {
			kernelOptions = append(kernelOptions, bpKernel.Append)
		}
		osc.KernelOptionsAppend = kernelOptions
		if imageConfig.KernelOptionsBootloader != nil {
			osc.KernelOptionsBootloader = *imageConfig.KernelOptionsBootloader
		}
	}

	osc.FIPS = c.GetFIPS()

	osc.BasePackages = osPackageSet.Include
	osc.ExcludeBasePackages = osPackageSet.Exclude
	osc.ExtraBaseRepos = osPackageSet.Repositories

	osc.Containers = containers

	osc.GPGKeyFiles = imageConfig.GPGKeyFiles
	if rpm := c.GetRPM(); rpm != nil && rpm.ImportKeys != nil {
		osc.GPGKeyFiles = append(osc.GPGKeyFiles, rpm.ImportKeys.Files...)
	}

	if imageConfig.ExcludeDocs != nil {
		osc.ExcludeDocs = *imageConfig.ExcludeDocs
	}

	if !t.BootISO {
		// don't put users and groups in the payload of an installer
		// add them via kickstart instead
		osc.Groups = users.GroupsFromBP(c.GetGroups())
		osc.Users = users.UsersFromBP(c.GetUsers())
	}

	osc.EnabledServices = imageConfig.EnabledServices
	osc.DisabledServices = imageConfig.DisabledServices
	osc.MaskedServices = imageConfig.MaskedServices
	if imageConfig.DefaultTarget != nil {
		osc.DefaultTarget = *imageConfig.DefaultTarget
	}

	osc.Firewall = imageConfig.Firewall
	if fw := c.GetFirewall(); fw != nil {
		options := osbuild.FirewallStageOptions{
			Ports: fw.Ports,
		}

		if fw.Services != nil {
			options.EnabledServices = fw.Services.Enabled
			options.DisabledServices = fw.Services.Disabled
		}
		if fw.Zones != nil {
			for _, z := range fw.Zones {
				options.Zones = append(options.Zones, osbuild.FirewallZone{
					Name:    *z.Name,
					Sources: z.Sources,
				})
			}
		}
		osc.Firewall = &options
	}

	language, keyboard := c.GetPrimaryLocale()
	if language != nil {
		osc.Language = *language
	} else if imageConfig.Locale != nil {
		osc.Language = *imageConfig.Locale
	}
	if keyboard != nil {
		osc.Keyboard = keyboard
	} else if imageConfig.Keyboard != nil {
		osc.Keyboard = &imageConfig.Keyboard.Keymap
		if imageConfig.Keyboard.X11Keymap != nil {
			osc.X11KeymapLayouts = imageConfig.Keyboard.X11Keymap.Layouts
		}
	}

	if hostname := c.GetHostname(); hostname != nil {
		osc.Hostname = *hostname
	} else if imageConfig.Hostname != nil {
		osc.Hostname = *imageConfig.Hostname
	}

	timezone, ntpServers := c.GetTimezoneSettings()
	if timezone != nil {
		osc.Timezone = *timezone
	} else if imageConfig.Timezone != nil {
		osc.Timezone = *imageConfig.Timezone
	}

	if len(ntpServers) > 0 {
		for _, server := range ntpServers {
			osc.NTPServers = append(osc.NTPServers, osbuild.ChronyConfigServer{Hostname: server})
		}
	} else if imageConfig.TimeSynchronization != nil {
		osc.NTPServers = imageConfig.TimeSynchronization.Servers
		osc.LeapSecTZ = imageConfig.TimeSynchronization.LeapsecTz
	}

	// Relabel the tree, unless the `NoSElinux` flag is explicitly set to `true`
	if imageConfig.NoSElinux == nil || imageConfig.NoSElinux != nil && !*imageConfig.NoSElinux {
		osc.SElinux = "targeted"
		osc.SELinuxForceRelabel = imageConfig.SELinuxForceRelabel
	}

	if t.IsRHEL && options.Facts != nil {
		osc.RHSMFacts = options.Facts
	}

	var err error
	osc.Directories, err = blueprint.DirectoryCustomizationsToFsNodeDirectories(c.GetDirectories())
	if err != nil {
		// In theory this should never happen, because the blueprint directory customizations
		// should have been validated before this point.
		panic(fmt.Sprintf("failed to convert directory customizations to fs node directories: %v", err))
	}

	osc.Files, err = blueprint.FileCustomizationsToFsNodeFiles(c.GetFiles())
	if err != nil {
		// In theory this should never happen, because the blueprint file customizations
		// should have been validated before this point.
		panic(fmt.Sprintf("failed to convert file customizations to fs node files: %v", err))
	}

	// OSTree commits do not include data in `/var` since that is tied to the
	// deployment, rather than the commit. Therefore the containers need to be
	// stored in a different location, like `/usr/share`, and the container
	// storage engine configured accordingly.
	if t.RpmOstree && len(containers) > 0 {
		storagePath := "/usr/share/containers/storage"
		osc.ContainersStorage = &storagePath
	}

	if containerStorage := c.GetContainerStorage(); containerStorage != nil {
		osc.ContainersStorage = containerStorage.StoragePath
	}

	// set yum repos first, so it doesn't get overridden by
	// imageConfig.YUMRepos
	osc.YUMRepos = imageConfig.YUMRepos

	customRepos, err := c.GetRepositories()
	if err != nil {
		// This shouldn't happen and since the repos
		// should have already been validated
		panic(fmt.Sprintf("failed to get custom repos: %v", err))
	}

	// This function returns a map of filename and corresponding yum repos
	// and a list of fs node files for the inline gpg keys so we can save
	// them to disk. This step also swaps the inline gpg key with the path
	// to the file in the os file tree
	yumRepos, gpgKeyFiles, err := blueprint.RepoCustomizationsToRepoConfigAndGPGKeyFiles(customRepos)
	if err != nil {
		panic(fmt.Sprintf("failed to convert inline gpgkeys to fs node files: %v", err))
	}

	// add the gpg key files to the list of files to be added to the tree
	if len(gpgKeyFiles) > 0 {
		osc.Files = append(osc.Files, gpgKeyFiles...)
	}

	for filename, repos := range yumRepos {
		osc.YUMRepos = append(osc.YUMRepos, osbuild.NewYumReposStageOptions(filename, repos))
	}

	if oscapConfig := c.GetOpenSCAP(); oscapConfig != nil {
		if t.RpmOstree {
			panic("unexpected oscap options for ostree image type")
		}

		oscapDataNode, err := fsnode.NewDirectory(oscap.DataDir, nil, nil, nil, true)
		if err != nil {
			panic(fmt.Sprintf("unexpected error creating required OpenSCAP directory: %s", oscap.DataDir))
		}
		osc.Directories = append(osc.Directories, oscapDataNode)

		remediationConfig, err := oscap.NewConfigs(*oscapConfig, imageConfig.DefaultOSCAPDatastream)
		if err != nil {
			panic(fmt.Errorf("error creating OpenSCAP configs: %w", err))
		}

		osc.OpenSCAPRemediationConfig = remediationConfig
	}

	var subscriptionStatus subscription.RHSMStatus
	if options.Subscription != nil {
		subscriptionStatus = subscription.RHSMConfigWithSubscription
		if options.Subscription.Proxy != "" {
			osc.InsightsClientConfig = &osbuild.InsightsClientConfigStageOptions{Proxy: options.Subscription.Proxy}
		}
	} else {
		subscriptionStatus = subscription.RHSMConfigNoSubscription
	}
	if rhsmConfig, exists := imageConfig.RHSMConfig[subscriptionStatus]; exists {
		osc.RHSMConfig = rhsmConfig
	}

	if bpRhsmConfig := subscription.RHSMConfigFromBP(c.GetRHSM()); bpRhsmConfig != nil {
		osc.RHSMConfig = osc.RHSMConfig.Update(bpRhsmConfig)
	}

	osc.ShellInit = imageConfig.ShellInit
	osc.Grub2Config = imageConfig.Grub2Config
	osc.Sysconfig = imageConfig.SysconfigStageOptions()
	osc.SystemdLogind = imageConfig.SystemdLogind
	osc.CloudInit = imageConfig.CloudInit
	osc.Modprobe = imageConfig.Modprobe
	osc.DracutConf = imageConfig.DracutConf
	osc.SystemdUnit = imageConfig.SystemdUnit
	osc.Authselect = imageConfig.Authselect
	osc.SELinuxConfig = imageConfig.SELinuxConfig
	osc.Tuned = imageConfig.Tuned
	osc.Tmpfilesd = imageConfig.Tmpfilesd
	osc.PamLimitsConf = imageConfig.PamLimitsConf
	osc.Sysctld = imageConfig.Sysctld
	osc.DNFConfig = imageConfig.DNFConfig
	osc.DNFAutomaticConfig = imageConfig.DNFAutomaticConfig
	osc.YUMConfig = imageConfig.YumConfig
	osc.SshdConfig = imageConfig.SshdConfig
	osc.AuthConfig = imageConfig.Authconfig
	osc.PwQuality = imageConfig.PwQuality
	osc.Subscription = options.Subscription
	osc.WAAgentConfig = imageConfig.WAAgentConfig
	osc.UdevRules = imageConfig.UdevRules
	osc.GCPGuestAgentConfig = imageConfig.GCPGuestAgentConfig
	osc.WSLConfig = imageConfig.WSLConfStageOptions()

	osc.Files = append(osc.Files, imageConfig.Files...)
	osc.Directories = append(osc.Directories, imageConfig.Directories...)

	if imageConfig.NoBLS != nil {
		osc.NoBLS = *imageConfig.NoBLS
	}

	ca, err := c.GetCACerts()
	if err != nil {
		panic(fmt.Sprintf("unexpected error checking CA certs: %v", err))
	}
	if ca != nil {
		osc.CACerts = ca.PEMCerts
	}

	if imageConfig.InstallWeakDeps != nil {
		osc.InstallWeakDeps = *imageConfig.InstallWeakDeps
	}

	if imageConfig.MachineIdUninitialized != nil {
		osc.MachineIdUninitialized = *imageConfig.MachineIdUninitialized
	}

	if imageConfig.MountUnits != nil {
		osc.MountUnits = *imageConfig.MountUnits
	}

	return osc, nil
}
