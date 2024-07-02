package distro

import (
	"github.com/osbuild/images/pkg/blueprint"
	"github.com/osbuild/images/pkg/customizations/subscription"
	"github.com/osbuild/images/pkg/disk"
	"github.com/osbuild/images/pkg/manifest"
	"github.com/osbuild/images/pkg/ostree"
	"github.com/osbuild/images/pkg/rhsm/facts"
	"github.com/osbuild/images/pkg/rpmmd"
)

type BootMode uint64

const (
	BOOT_NONE BootMode = iota
	BOOT_LEGACY
	BOOT_UEFI
	BOOT_HYBRID
	UnsupportedCustomizationError = "unsupported blueprint customizations found for image type %q: (allowed: %s)"
	NoCustomizationsAllowedError  = "image type %q does not support customizations"
)

func (m BootMode) String() string {
	switch m {
	case BOOT_NONE:
		return "none"
	case BOOT_LEGACY:
		return "legacy"
	case BOOT_UEFI:
		return "uefi"
	case BOOT_HYBRID:
		return "hybrid"
	default:
		panic("invalid boot mode")
	}
}

// A Distro represents composer's notion of what a given distribution is.
type Distro interface {
	// Returns the name of the distro.
	Name() string

	// Returns the codename of the distro.
	Codename() string

	// Returns the release version of the distro. This is used in repo
	// files on the host system and required for the subscription support.
	Releasever() string

	// Returns the OS version of the distro, which may contain minor versions
	// if the distro supports them. This is used in various places where the
	// minor version of the distro is needed to determine the correct
	// configuration.
	OsVersion() string

	// Returns the module platform id of the distro. This is used by DNF
	// for modularity support.
	ModulePlatformID() string

	// Returns the product name of the distro.
	Product() string

	// Returns the ostree reference template
	OSTreeRef() string

	// Returns a sorted list of the names of the architectures this distro
	// supports.
	ListArches() []string

	// Returns an object representing the given architecture as support
	// by this distro.
	GetArch(arch string) (Arch, error)
}

// An Arch represents a given distribution's support for a given architecture.
type Arch interface {
	// Returns the name of the architecture.
	Name() string

	// Returns a sorted list of the names of the image types this architecture
	// supports.
	ListImageTypes() []string

	// Returns an object representing a given image format for this architecture,
	// on this distro.
	GetImageType(imageType string) (ImageType, error)

	// Returns the parent distro
	Distro() Distro
}

// An ImageType represents a given distribution's support for a given Image Type
// for a given architecture.
type ImageType interface {
	// Returns the name of the image type.
	Name() string

	// Returns the parent architecture
	Arch() Arch

	// Returns the canonical filename for the image type.
	Filename() string

	// Retrns the MIME-type for the image type.
	MIMEType() string

	// Returns the default OSTree ref for the image type.
	OSTreeRef() string

	// Returns the ISO Label for the image type. Returns an error if the image
	// type is not an ISO.
	ISOLabel() (string, error)

	// Returns the proper image size for a given output format. If the input size
	// is 0 the default value for the format will be returned.
	Size(size uint64) uint64

	// Returns the corresponding partion type ("gpt", "dos") or "" the image type
	// has no partition table. Only support for RHEL 8.5+
	PartitionType() string

	// Returns the corresponding boot mode ("legacy", "uefi", "hybrid") or "none"
	BootMode() BootMode

	// Returns the names of the pipelines that set up the build environment (buildroot).
	BuildPipelines() []string

	// Returns the names of the pipelines that create the image.
	PayloadPipelines() []string

	// Returns the package set names safe to install custom packages via custom repositories.
	PayloadPackageSets() []string

	// Returns named arrays of package set names which should be depsolved in a chain.
	PackageSetsChains() map[string][]string

	// Returns the names of the stages that will produce the build output.
	Exports() []string

	// Returns an osbuild manifest, containing the sources and pipeline necessary
	// to build an image, given output format with all packages and customizations
	// specified in the given blueprint; it also returns any warnings (e.g.
	// deprecation notices) generated by the manifest.
	// The packageSpecSets must be labelled in the same way as the originating PackageSets.
	Manifest(bp *blueprint.Blueprint, options ImageOptions, repos []rpmmd.RepoConfig, seed int64) (manifest.Manifest, []string, error)
}

// The ImageOptions specify options for a specific image build
type ImageOptions struct {
	Size             uint64                     `json:"size"`
	OSTree           *ostree.ImageOptions       `json:"ostree,omitempty"`
	Subscription     *subscription.ImageOptions `json:"subscription,omitempty"`
	Facts            *facts.ImageOptions        `json:"facts,omitempty"`
	PartitioningMode disk.PartitioningMode      `json:"partitioning-mode,omitempty"`
}

type BasePartitionTableMap map[string]disk.PartitionTable

// Fallbacks: When a new method is added to an interface to provide to provide
// information that isn't available for older implementations, the older
// methods should return a fallback/default value by calling the appropriate
// function from below.
// Example: Exports() simply returns "assembler" for older image type
// implementations that didn't produce v1 manifests that have named pipelines.
func BuildPipelinesFallback() []string {
	return []string{"build"}
}

func PayloadPipelinesFallback() []string {
	return []string{"os", "assembler"}
}

func ExportsFallback() []string {
	return []string{"assembler"}
}

func PayloadPackageSets() []string {
	return []string{}
}
