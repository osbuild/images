package bootc

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/osbuild/blueprint/pkg/blueprint"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/arch"
	bibcontainer "github.com/osbuild/images/pkg/bib/container"
	"github.com/osbuild/images/pkg/bib/osinfo"
	"github.com/osbuild/images/pkg/container"
	"github.com/osbuild/images/pkg/customizations/anaconda"
	"github.com/osbuild/images/pkg/customizations/kickstart"
	"github.com/osbuild/images/pkg/customizations/users"
	"github.com/osbuild/images/pkg/disk"
	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/distro/defs"
	"github.com/osbuild/images/pkg/dnfjson"
	"github.com/osbuild/images/pkg/image"
	"github.com/osbuild/images/pkg/manifest"
	"github.com/osbuild/images/pkg/osbuild"
	"github.com/osbuild/images/pkg/platform"
	"github.com/osbuild/images/pkg/policies"
	"github.com/osbuild/images/pkg/rpmmd"
	"github.com/osbuild/images/pkg/runner"
)

var _ = distro.CustomDepsolverDistro(&BootcDistro{})

type BootcDistro struct {
	imgref          string
	buildImgref     string
	sourceInfo      *osinfo.Info
	buildSourceInfo *osinfo.Info

	name          string
	defaultFs     string
	releasever    string
	rootfsMinSize uint64

	arches map[string]distro.Arch
}

var _ = distro.Arch(&BootcArch{})

type BootcArch struct {
	distro *BootcDistro
	arch   arch.Arch

	imageTypes map[string]distro.ImageType
}

var _ = distro.ImageType(&BootcImageType{})

type BootcImageType struct {
	arch *BootcArch

	name   string
	export string
	// file extension
	ext string
	// image is an iso
	iso bool
}

func (d *BootcDistro) SetBuildContainer(imgref string) (err error) {
	if imgref == "" {
		return nil
	}

	cnt, err := bibcontainer.New(imgref)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, cnt.Stop())
	}()

	info, err := osinfo.Load(cnt.Root())
	if err != nil {
		return err
	}
	d.buildImgref = imgref
	d.buildSourceInfo = info
	return nil
}

func (d *BootcDistro) SetDefaultFs(defaultFs string) error {
	if defaultFs == "" {
		return nil
	}

	d.defaultFs = defaultFs
	return nil
}

func (d *BootcDistro) DefaultFs() string {
	return d.defaultFs
}

func (d *BootcDistro) Name() string {
	return d.name
}

func (d *BootcDistro) Codename() string {
	return ""
}

func (d *BootcDistro) Releasever() string {
	return d.releasever
}

func (d *BootcDistro) OsVersion() string {
	return d.releasever
}

func (d *BootcDistro) Product() string {
	return d.name
}

func (d *BootcDistro) ModulePlatformID() string {
	return ""
}

func (d *BootcDistro) OSTreeRef() string {
	return ""
}

func (d *BootcDistro) Depsolver(rpmCacheRoot string, archi arch.Arch) (solver *dnfjson.Solver, cleanup func(), err error) {
	cnt, err := bibcontainer.New(d.buildImgref)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, cnt.Stop())
		}
	}()

	cleanup = func() {
		cnt.Stop()
	}
	if err := cnt.InitDNF(); err != nil {
		return nil, nil, err
	}
	solver, err = cnt.NewContainerSolver(rpmCacheRoot, archi, d.buildSourceInfo)
	if err != nil {
		return nil, nil, err
	}

	return solver, cleanup, nil
}

func (d *BootcDistro) ListArches() []string {
	archs := make([]string, 0, len(d.arches))
	for name := range d.arches {
		archs = append(archs, name)
	}
	sort.Strings(archs)
	return archs
}

func (d *BootcDistro) GetArch(arch string) (distro.Arch, error) {
	a, exists := d.arches[arch]
	if !exists {
		return nil, errors.New("invalid arch: " + arch)
	}
	return a, nil
}

func (d *BootcDistro) addArches(arches ...*BootcArch) {
	if d.arches == nil {
		d.arches = map[string]distro.Arch{}
	}

	for _, a := range arches {
		a.distro = d
		d.arches[a.Name()] = a
	}
}

func (a *BootcArch) Name() string {
	return a.arch.String()
}

func (a *BootcArch) Distro() distro.Distro {
	return a.distro
}

func (a *BootcArch) ListImageTypes() []string {
	formats := make([]string, 0, len(a.imageTypes))
	for name := range a.imageTypes {
		formats = append(formats, name)
	}
	sort.Strings(formats)
	return formats
}

func (a *BootcArch) GetImageType(imageType string) (distro.ImageType, error) {
	t, exists := a.imageTypes[imageType]
	if !exists {
		return nil, errors.New("invalid image type: " + imageType)
	}

	return t, nil
}

func (a *BootcArch) addImageTypes(imageTypes ...BootcImageType) {
	if a.imageTypes == nil {
		a.imageTypes = map[string]distro.ImageType{}
	}
	for idx := range imageTypes {
		it := imageTypes[idx]
		it.arch = a
		a.imageTypes[it.Name()] = &it
	}
}

func (t *BootcImageType) Name() string {
	return t.name
}

func (t *BootcImageType) Aliases() []string {
	return nil
}

func (t *BootcImageType) Arch() distro.Arch {
	return t.arch
}

func (t *BootcImageType) Filename() string {
	if t.iso {
		return "install.iso"
	}
	return fmt.Sprintf("disk.%s", t.ext)
}

func (t *BootcImageType) MIMEType() string {
	return "application/x-test"
}

func (t *BootcImageType) OSTreeRef() string {
	return ""
}

func (t *BootcImageType) ISOLabel() (string, error) {
	return "", nil
}

func (t *BootcImageType) Size(size uint64) uint64 {
	if size == 0 {
		size = 1073741824
	}
	return size
}

func (t *BootcImageType) PartitionType() disk.PartitionTableType {
	return disk.PT_NONE
}

func (t *BootcImageType) BasePartitionTable() (*disk.PartitionTable, error) {
	return nil, nil
}

func (t *BootcImageType) BootMode() platform.BootMode {
	return platform.BOOT_HYBRID
}

func (t *BootcImageType) BuildPipelines() []string {
	return []string{"build"}
}

func (t *BootcImageType) PayloadPipelines() []string {
	return []string{""}
}

func (t *BootcImageType) PayloadPackageSets() []string {
	return nil
}

func (t *BootcImageType) Exports() []string {
	return []string{t.export}
}

func (t *BootcImageType) SupportedBlueprintOptions() []string {
	return []string{
		"customizations.directories",
		"customizations.disk",
		"customizations.files",
		"customizations.filesystem",
		"customizations.group",
		"customizations.kernel",
		"customizations.user",
	}
}
func (t *BootcImageType) RequiredBlueprintOptions() []string {
	return nil
}

func (t *BootcImageType) manifestForDisk(bp *blueprint.Blueprint, options distro.ImageOptions, repos []rpmmd.RepoConfig, seedp *int64) (*manifest.Manifest, []string, error) {
	if t.arch.distro.imgref == "" {
		return nil, nil, fmt.Errorf("internal error: no base image defined")
	}

	containerSource := container.SourceSpec{
		Source: t.arch.distro.imgref,
		Name:   t.arch.distro.imgref,
		Local:  true,
	}
	buildContainerSource := container.SourceSpec{
		Source: t.arch.distro.buildImgref,
		Name:   t.arch.distro.buildImgref,
		Local:  true,
	}

	var customizations *blueprint.Customizations
	if bp != nil {
		customizations = bp.Customizations
	}

	archi := common.Must(arch.FromString(t.arch.Name()))
	platform := &platform.Data{
		Arch:        archi,
		UEFIVendor:  t.arch.distro.sourceInfo.UEFIVendor,
		QCOW2Compat: "1.1",
	}
	switch archi {
	case arch.ARCH_X86_64:
		platform.BIOSPlatform = "i386-pc"
	case arch.ARCH_PPC64LE:
		platform.BIOSPlatform = "powerpc-ieee1275"
	case arch.ARCH_S390X:
		platform.ZiplSupport = true
	}
	// For the bootc-disk image, the filename is the basename and
	// the extension is added automatically for each disk format
	filename := "disk"

	img := image.NewBootcDiskImage(platform, filename, containerSource, buildContainerSource)
	img.OSCustomizations.Users = users.UsersFromBP(customizations.GetUsers())
	img.OSCustomizations.Groups = users.GroupsFromBP(customizations.GetGroups())
	img.OSCustomizations.SELinux = t.arch.distro.sourceInfo.SELinuxPolicy
	img.OSCustomizations.BuildSELinux = img.OSCustomizations.SELinux
	if t.arch.distro.buildSourceInfo != nil {
		img.OSCustomizations.BuildSELinux = t.arch.distro.buildSourceInfo.SELinuxPolicy
	}

	img.OSCustomizations.KernelOptionsAppend = []string{
		"rw",
		// TODO: Drop this as we expect kargs to come from the container image,
		// xref https://github.com/CentOS/centos-bootc-layered/blob/main/cloud/usr/lib/bootc/install/05-cloud-kargs.toml
		"console=tty0",
		"console=ttyS0",
	}

	if kopts := customizations.GetKernel(); kopts != nil && kopts.Append != "" {
		img.OSCustomizations.KernelOptionsAppend = append(img.OSCustomizations.KernelOptionsAppend, kopts.Append)
	}

	rootfsMinSize := max(t.arch.distro.rootfsMinSize, options.Size)
	rng := createRand()
	pt, err := t.genPartitionTable(customizations, rootfsMinSize, rng)
	if err != nil {
		return nil, nil, err
	}
	img.PartitionTable = pt

	// Check Directory/File Customizations are valid
	dc := customizations.GetDirectories()
	fc := customizations.GetFiles()
	if err := blueprint.ValidateDirFileCustomizations(dc, fc); err != nil {
		return nil, nil, err
	}
	if err := blueprint.CheckDirectoryCustomizationsPolicy(dc, policies.OstreeCustomDirectoriesPolicies); err != nil {
		return nil, nil, err
	}
	if err := blueprint.CheckFileCustomizationsPolicy(fc, policies.OstreeCustomFilesPolicies); err != nil {
		return nil, nil, err
	}
	img.OSCustomizations.Files, err = blueprint.FileCustomizationsToFsNodeFiles(fc)
	if err != nil {
		return nil, nil, err
	}
	img.OSCustomizations.Directories, err = blueprint.DirectoryCustomizationsToFsNodeDirectories(dc)
	if err != nil {
		return nil, nil, err
	}

	mf := manifest.New()
	mf.Distro = manifest.DISTRO_FEDORA
	runner := &runner.Linux{}

	if err := img.InstantiateManifestFromContainers(&mf, []container.SourceSpec{containerSource}, runner, rng); err != nil {
		return nil, nil, err
	}

	return &mf, nil, nil
}

func needsRHELLoraxTemplates(si osinfo.OSRelease) bool {
	return si.ID == "rhel" || slices.Contains(si.IDLike, "rhel") || si.VersionID == "eln"
}
func getDistroAndRunner(osRelease osinfo.OSRelease) (manifest.Distro, runner.Runner, error) {
	switch osRelease.ID {
	case "fedora":
		version, err := strconv.ParseUint(osRelease.VersionID, 10, 64)
		if err != nil {
			return manifest.DISTRO_NULL, nil, fmt.Errorf("cannot parse Fedora version (%s): %w", osRelease.VersionID, err)
		}

		return manifest.DISTRO_FEDORA, &runner.Fedora{
			Version: version,
		}, nil
	case "centos":
		version, err := strconv.ParseUint(osRelease.VersionID, 10, 64)
		if err != nil {
			return manifest.DISTRO_NULL, nil, fmt.Errorf("cannot parse CentOS version (%s): %w", osRelease.VersionID, err)
		}
		r := &runner.CentOS{
			Version: version,
		}
		switch version {
		case 9:
			return manifest.DISTRO_EL9, r, nil
		case 10:
			return manifest.DISTRO_EL10, r, nil
		default:
			logrus.Warnf("Unknown CentOS version %d, using default distro for manifest generation", version)
			return manifest.DISTRO_NULL, r, nil
		}

	case "rhel":
		versionParts := strings.Split(osRelease.VersionID, ".")
		if len(versionParts) != 2 {
			return manifest.DISTRO_NULL, nil, fmt.Errorf("invalid RHEL version format: %s", osRelease.VersionID)
		}
		major, err := strconv.ParseUint(versionParts[0], 10, 64)
		if err != nil {
			return manifest.DISTRO_NULL, nil, fmt.Errorf("cannot parse RHEL major version (%s): %w", versionParts[0], err)
		}
		minor, err := strconv.ParseUint(versionParts[1], 10, 64)
		if err != nil {
			return manifest.DISTRO_NULL, nil, fmt.Errorf("cannot parse RHEL minor version (%s): %w", versionParts[1], err)
		}
		r := &runner.RHEL{
			Major: major,
			Minor: minor,
		}
		switch major {
		case 9:
			return manifest.DISTRO_EL9, r, nil
		case 10:
			return manifest.DISTRO_EL10, r, nil
		default:
			logrus.Warnf("Unknown RHEL version %d, using default distro for manifest generation", major)
			return manifest.DISTRO_NULL, r, nil
		}
	}

	logrus.Warnf("Unknown distro %s, using default runner", osRelease.ID)
	return manifest.DISTRO_NULL, &runner.Linux{}, nil
}

func labelForISO(os *osinfo.OSRelease, arch *arch.Arch) string {
	switch os.ID {
	case "fedora":
		return fmt.Sprintf("Fedora-S-dvd-%s-%s", arch, os.VersionID)
	case "centos":
		labelTemplate := "CentOS-Stream-%s-BaseOS-%s"
		if os.VersionID == "8" {
			labelTemplate = "CentOS-Stream-%s-%s-dvd"
		}
		return fmt.Sprintf(labelTemplate, os.VersionID, arch)
	case "rhel":
		version := strings.ReplaceAll(os.VersionID, ".", "-")
		return fmt.Sprintf("RHEL-%s-BaseOS-%s", version, arch)
	default:
		return fmt.Sprintf("Container-Installer-%s", arch)
	}
}

func (t *BootcImageType) manifestForISO(bp *blueprint.Blueprint, options distro.ImageOptions, repos []rpmmd.RepoConfig, seedp *int64) (*manifest.Manifest, []string, error) {
	if t.arch.distro.imgref == "" {
		return nil, nil, fmt.Errorf("pipeline: no base image defined")
	}

	containerSource := container.SourceSpec{
		Source: t.arch.distro.imgref,
		Name:   t.arch.distro.imgref,
		Local:  true,
	}

	// XXX: duplicated
	archi := common.Must(arch.FromString(t.arch.Name()))
	platform := &platform.Data{
		Arch:        archi,
		UEFIVendor:  t.arch.distro.sourceInfo.UEFIVendor,
		QCOW2Compat: "1.1",
	}
	switch archi {
	case arch.ARCH_X86_64:
		platform.BIOSPlatform = "i386-pc"
	case arch.ARCH_PPC64LE:
		platform.BIOSPlatform = "powerpc-ieee1275"
	case arch.ARCH_S390X:
		platform.ZiplSupport = true
	}

	// The ref is not needed and will be removed from the ctor later
	// in time
	img := image.NewAnacondaContainerInstaller(platform, "install.iso", containerSource, "")
	img.ContainerRemoveSignatures = true
	img.RootfsCompression = "zstd"

	img.InstallerCustomizations.Product = t.arch.distro.sourceInfo.OSRelease.Name
	img.InstallerCustomizations.OSVersion = t.arch.distro.sourceInfo.OSRelease.VersionID

	nameVer := fmt.Sprintf("%s-%v", t.arch.distro.sourceInfo.OSRelease.ID, t.arch.distro.sourceInfo.OSRelease.VersionID)
	id, err := distro.ParseID(nameVer)
	if err != nil {
		return nil, nil, err
	}
	dy, err := defs.NewDistroYAML(nameVer)
	if err != nil {
		return nil, nil, err
	}
	di := dy.ImageTypes()["image-installer"]
	img.ExtraBasePackages = rpmmd.PackageSet{
		Include: di.PackageSets(*id, t.arch.Name())["installer"].Include,
	}
	// XXX: use dy.getISOLabelFunc()
	img.InstallerCustomizations.ISOLabel = labelForISO(&t.arch.distro.sourceInfo.OSRelease, &archi)

	var customizations *blueprint.Customizations
	if bp != nil {
		customizations = bp.Customizations
	}
	img.InstallerCustomizations.FIPS = customizations.GetFIPS()
	img.Kickstart, err = kickstart.New(customizations)
	if err != nil {
		return nil, nil, err
	}
	img.Kickstart.Path = osbuild.KickstartPathOSBuild
	if kopts := customizations.GetKernel(); kopts != nil && kopts.Append != "" {
		img.Kickstart.KernelOptionsAppend = append(img.Kickstart.KernelOptionsAppend, kopts.Append)
	}
	img.Kickstart.NetworkOnBoot = true

	instCust, err := customizations.GetInstaller()
	if err != nil {
		return nil, nil, err
	}
	if instCust != nil && instCust.Modules != nil {
		img.InstallerCustomizations.EnabledAnacondaModules = append(img.InstallerCustomizations.EnabledAnacondaModules, instCust.Modules.Enable...)
		img.InstallerCustomizations.DisabledAnacondaModules = append(img.InstallerCustomizations.DisabledAnacondaModules, instCust.Modules.Disable...)
	}
	img.InstallerCustomizations.EnabledAnacondaModules = append(img.InstallerCustomizations.EnabledAnacondaModules,
		// XXX: ???
		"org.fedoraproject.Anaconda.Modules.Network",
		"org.fedoraproject.Anaconda.Modules.Payloads",
		"org.fedoraproject.Anaconda.Modules.Runtime",
		"org.fedoraproject.Anaconda.Modules.Storage",
		anaconda.ModuleUsers,
		anaconda.ModuleServices,
		anaconda.ModuleSecurity,
	)

	img.Kickstart.OSTree = &kickstart.OSTree{
		OSName: "default",
	}
	img.InstallerCustomizations.UseRHELLoraxTemplates = needsRHELLoraxTemplates(t.arch.distro.sourceInfo.OSRelease)
	// see https://github.com/osbuild/bootc-image-builder/issues/733
	img.InstallerCustomizations.ISORootfsType = manifest.SquashfsRootfs

	installRootfsType, err := disk.NewFSType(t.arch.distro.defaultFs)
	if err != nil {
		return nil, nil, err
	}
	img.InstallRootfsType = installRootfsType

	mf := manifest.New()

	foundDistro, foundRunner, err := getDistroAndRunner(t.arch.distro.sourceInfo.OSRelease)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to infer distro and runner: %w", err)
	}
	mf.Distro = foundDistro

	rng := createRand()
	_, err = img.InstantiateManifest(&mf, nil, foundRunner, rng)
	return &mf, nil, err
}

func (t *BootcImageType) Manifest(bp *blueprint.Blueprint, options distro.ImageOptions, repos []rpmmd.RepoConfig, seedp *int64) (*manifest.Manifest, []string, error) {
	if t.iso {
		return t.manifestForISO(bp, options, repos, seedp)
	}
	return t.manifestForDisk(bp, options, repos, seedp)
}

// newBootcDistro returns a new instance of BootcDistro
// from the given url
func NewBootcDistro(imgref string) (bd *BootcDistro, err error) {
	cnt, err := bibcontainer.New(imgref)
	if err != nil {
		return nil, err
	}
	defer func() {
		err = errors.Join(err, cnt.Stop())
	}()

	info, err := osinfo.Load(cnt.Root())
	if err != nil {
		return nil, err
	}

	// XXX: provide a way to set defaultfs (needed for bib)
	defaultFs, err := cnt.DefaultRootfsType()
	if err != nil {
		return nil, err
	}
	cntSize, err := getContainerSize(imgref)
	if err != nil {
		return nil, fmt.Errorf("cannot get container size: %w", err)
	}

	nameVer := fmt.Sprintf("bootc-%s-%s", info.OSRelease.ID, info.OSRelease.VersionID)
	bd = &BootcDistro{
		name:          nameVer,
		releasever:    info.OSRelease.VersionID,
		defaultFs:     defaultFs,
		rootfsMinSize: cntSize * containerSizeToDiskSizeMultiplier,

		imgref:     imgref,
		sourceInfo: info,
		// default buildref/info to regular container, this can
		// be overriden with SetBuildContainer()
		buildImgref:     imgref,
		buildSourceInfo: info,
	}

	for _, archStr := range []string{"x86_64", "aarch64", "ppc64le", "s390x", "riscv64"} {
		ba := &BootcArch{
			arch: common.Must(arch.FromString(archStr)),
		}
		// TODO: add iso image types, see bootc-image-builder
		//
		// Note that the file extension is hardcoded in
		// pkg/image/bootc_disk.go, we have no way to access
		// it here so we need to duplicate it
		// XXX: find a way to avoid this duplication
		ba.addImageTypes(
			BootcImageType{
				name:   "ami",
				export: "image",
				ext:    "raw",
			},
			BootcImageType{
				name:   "qcow2",
				export: "qcow2",
				ext:    "qcow2",
			},
			BootcImageType{
				name:   "raw",
				export: "image",
				ext:    "raw",
			},
			BootcImageType{
				name:   "vmdk",
				export: "vmdk",
				ext:    "vmdk",
			},
			BootcImageType{
				name:   "vhd",
				export: "bpc",
				ext:    "vhd",
			},
			BootcImageType{
				name:   "gce",
				export: "gce",
				ext:    "tar.gz",
			},
			// Image types that build ISOs
			BootcImageType{
				name:   "anaconda-iso",
				export: "bootiso",
				iso:    true,
			},
			BootcImageType{
				name:   "iso",
				export: "bootiso",
				iso:    true,
			},
		)
		bd.addArches(ba)
	}

	return bd, nil
}

func DistroFactory(idStr string) distro.Distro {
	l := strings.SplitN(idStr, ":", 2)
	if l[0] != "bootc" {
		return nil
	}
	imgRef := l[1]

	return common.Must(NewBootcDistro(imgRef))
}
