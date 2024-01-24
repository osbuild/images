package distro_test_common

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osbuild/images/internal/reporegistry"
	"github.com/osbuild/images/pkg/blueprint"
	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/ostree"
)

const RandomTestSeed = 0

func isOSTree(imgType distro.ImageType) bool {
	return imgType.OSTreeRef() != ""
}

func isUbi(imgType distro.ImageType) bool {
	return imgType.Name() == "wsl"
}

var knownKernels = []string{"kernel", "kernel-debug", "kernel-rt"}

// Returns the number of known kernels in the package list
func kernelCount(imgType distro.ImageType, bp blueprint.Blueprint) int {
	ostreeOptions := &ostree.ImageOptions{
		URL: "https://example.com", // required by some image types
	}
	manifest, _, err := imgType.Manifest(&bp, distro.ImageOptions{OSTree: ostreeOptions}, nil, 0)
	if err != nil {
		panic(err)
	}
	sets := manifest.GetPackageSetChains()

	// Use a map to count unique kernels in a package set. If the same kernel
	// name appears twice, it will only be installed once, so we only count it
	// once.
	kernels := make(map[string]bool)
	for _, name := range []string{
		// payload package set names
		"os", "ostree-tree", "anaconda-tree",
		"packages", "installer",
	} {
		for _, pset := range sets[name] {
			for _, pkg := range pset.Include {
				for _, kernel := range knownKernels {
					if kernel == pkg {
						kernels[kernel] = true
					}
				}
			}
			if len(kernels) > 0 {
				// BUG: some RHEL image types contain both 'packages'
				// and 'installer' even though only 'installer' is used
				// this counts the kernel package twice. None of these
				// sets should appear more than once, so return the count
				// for the first package set that has at least one kernel.
				return len(kernels)
			}
		}
	}
	return len(kernels)
}

func TestDistro_KernelOption(t *testing.T, d distro.Distro) {
	skipList := map[string]bool{
		// Ostree installers and raw images download a payload to embed or
		// deploy.  The kernel is part of the payload so it doesn't appear in
		// the image type's package lists.
		"edge-ami":                  true,
		"edge-installer":            true,
		"edge-raw-image":            true,
		"edge-simplified-installer": true,
		"edge-vsphere":              true,
		"iot-installer":             true,
		"iot-qcow2-image":           true,
		"iot-raw-image":             true,
		"iot-simplified-installer":  true,

		// the tar image type is a minimal image type which is not expected to
		// be usable without a blueprint (see commit 83a63aaf172f556f6176e6099ffaa2b5357b58f5).
		"tar": true,

		// containers don't have kernels
		"container": true,

		// image installer on Fedora doesn't support kernel customizations
		// on RHEL we support kernel name
		// TODO: Remove when we unify the allowed options
		"image-installer": true,
		"live-installer":  true,
	}

	{ // empty blueprint: all image types should just have the default kernel
		for _, archName := range d.ListArches() {
			arch, err := d.GetArch(archName)
			assert.NoError(t, err)
			for _, typeName := range arch.ListImageTypes() {
				if skipList[typeName] {
					continue
				}
				imgType, err := arch.GetImageType(typeName)
				assert.NoError(t, err)
				nk := kernelCount(imgType, blueprint.Blueprint{})

				if isUbi(imgType) {
					if nk != 0 {
						assert.Fail(t, fmt.Sprintf("%s Kernel count", d.Name()),
							"Image type %s (arch %s) specifies %d Kernel packages", typeName, archName, nk)
					}
				} else if nk != 1 {
					assert.Fail(t, fmt.Sprintf("%s Kernel count", d.Name()),
						"Image type %s (arch %s) specifies %d Kernel packages", typeName, archName, nk)
				}
			}
		}
	}

	{ // kernel in blueprint: the specified kernel replaces the default
		for _, kernelName := range []string{"kernel", "kernel-debug"} {
			bp := blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					Kernel: &blueprint.KernelCustomization{
						Name: kernelName,
					},
				},
			}
			for _, archName := range d.ListArches() {
				arch, err := d.GetArch(archName)
				assert.NoError(t, err)
				for _, typeName := range arch.ListImageTypes() {
					if typeName != "image-installer" {
						continue
					}
					if typeName != "live-installer" {
						continue
					}
					if skipList[typeName] {
						continue
					}
					imgType, err := arch.GetImageType(typeName)
					assert.NoError(t, err)
					nk := kernelCount(imgType, bp)

					// ostree image types should have only one kernel
					// ubi image types should have no kernels
					// other image types should have at least 1
					if nk < 1 || (nk != 1 && isOSTree(imgType)) || nk == 0 && isUbi(imgType) {
						assert.Fail(t, fmt.Sprintf("%s Kernel count", d.Name()),
							"Image type %s (arch %s) specifies %d Kernel packages", typeName, archName, nk)
					}
				}
			}
		}
	}
}

func TestDistro_OSTreeOptions(t *testing.T, d distro.Distro) {
	// test that ostree parameters are properly resolved by image functions that should support them
	typesWithParent := map[string]bool{ // image types that support specifying a parent commit
		"edge-commit":            true,
		"edge-container":         true,
		"iot-commit":             true,
		"iot-container":          true,
		"iot-bootable-container": true,
	}

	typesWithPayload := map[string]bool{
		"edge-vsphere":              true,
		"edge-ami":                  true,
		"edge-installer":            true,
		"edge-raw-image":            true,
		"edge-simplified-installer": true,
		"iot-ami":                   true,
		"iot-installer":             true,
		"iot-qcow2-image":           true,
		"iot-raw-image":             true,
		"iot-simplified-installer":  true,
	}

	assert := assert.New(t)

	{ // empty options: payload ref should equal default
		for _, archName := range d.ListArches() {
			arch, err := d.GetArch(archName)
			assert.NoError(err)
			for _, typeName := range arch.ListImageTypes() {
				bp := &blueprint.Blueprint{}
				if strings.HasSuffix(typeName, "simplified-installer") {
					// simplified installers require installation device
					bp = &blueprint.Blueprint{
						Customizations: &blueprint.Customizations{
							InstallationDevice: "/dev/sda42",
						},
					}
				}
				imgType, err := arch.GetImageType(typeName)
				assert.NoError(err)

				ostreeOptions := ostree.ImageOptions{}
				if typesWithPayload[typeName] {
					// payload types require URL
					ostreeOptions.URL = "https://example.com/repo"
				}
				options := distro.ImageOptions{OSTree: &ostreeOptions}

				m, _, err := imgType.Manifest(bp, options, nil, 0)
				assert.NoError(err)

				nrefs := 0
				// If a manifest returns an ostree source spec, the ref should
				// match the default.
				for _, commits := range m.GetOSTreeSourceSpecs() {
					for _, commit := range commits {
						assert.Equal(options.OSTree.URL, commit.URL, "url does not match expected for image type %q\n", typeName)
						assert.Equal(imgType.OSTreeRef(), commit.Ref, "ref does not match expected for image type %q\n", typeName)
						nrefs++
					}
				}
				nexpected := 0
				if typesWithPayload[typeName] {
					// image types with payload should return a ref
					nexpected = 1
				}
				assert.Equal(nexpected, nrefs, "incorrect ref count for image type %q\n", typeName)
			}
		}
	}

	{ // ImageRef set: should be returned as payload ref - no parent for commits and containers
		for _, archName := range d.ListArches() {
			arch, err := d.GetArch(archName)
			assert.NoError(err)
			for _, typeName := range arch.ListImageTypes() {
				bp := &blueprint.Blueprint{}
				if strings.HasSuffix(typeName, "simplified-installer") {
					// simplified installers require installation device
					bp = &blueprint.Blueprint{
						Customizations: &blueprint.Customizations{
							InstallationDevice: "/dev/sda42",
						},
					}
				}

				imgType, err := arch.GetImageType(typeName)
				assert.NoError(err)

				ostreeOptions := ostree.ImageOptions{
					ImageRef: "test/x86_64/01",
				}
				if typesWithPayload[typeName] {
					// payload types require URL
					ostreeOptions.URL = "https://example.com/repo"
				}
				options := distro.ImageOptions{OSTree: &ostreeOptions}
				m, _, err := imgType.Manifest(bp, options, nil, 0)
				assert.NoError(err)

				nrefs := 0
				// if a manifest returns an ostree source spec, the ref should
				// match the default
				for _, commits := range m.GetOSTreeSourceSpecs() {
					for _, commit := range commits {
						assert.Equal(options.OSTree.URL, commit.URL, "url does not match expected for image type %q\n", typeName)
						assert.Equal(options.OSTree.ImageRef, commit.Ref, "ref does not match expected for image type %q\n", typeName)
						nrefs++
					}
				}
				nexpected := 0
				if typesWithPayload[typeName] {
					// image types with payload should return a ref
					nexpected = 1
				}
				assert.Equal(nexpected, nrefs, "incorrect ref count for image type %q\n", typeName)
			}
		}
	}

	{ // URL always specified: should add a parent to image types that support it and the ref should match the option
		for _, archName := range d.ListArches() {
			arch, err := d.GetArch(archName)
			assert.NoError(err)
			for _, typeName := range arch.ListImageTypes() {
				bp := &blueprint.Blueprint{}
				if strings.HasSuffix(typeName, "simplified-installer") {
					// simplified installers require installation device
					bp = &blueprint.Blueprint{
						Customizations: &blueprint.Customizations{
							InstallationDevice: "/dev/sda42",
						},
					}
				}

				imgType, err := arch.GetImageType(typeName)
				assert.NoError(err)

				ostreeOptions := ostree.ImageOptions{
					ImageRef: "test/x86_64/01",
					URL:      "https://example.com/repo",
				}
				options := distro.ImageOptions{OSTree: &ostreeOptions}
				m, _, err := imgType.Manifest(bp, options, nil, 0)
				assert.NoError(err)

				nrefs := 0
				for _, commits := range m.GetOSTreeSourceSpecs() {
					for _, commit := range commits {
						assert.Equal(options.OSTree.URL, commit.URL, "url does not match expected for image type %q\n", typeName)
						assert.Equal(options.OSTree.ImageRef, commit.Ref, "ref does not match expected for image type %q\n", typeName)
						nrefs++
					}
				}
				nexpected := 0
				if typesWithPayload[typeName] || typesWithParent[typeName] {
					// image types with payload or parent should return a ref
					nexpected = 1
				}
				assert.Equal(nexpected, nrefs, "incorrect ref count for image type %q\n", typeName)
			}
		}
	}

	{ // URL and parent ref always specified: payload ref should be default - parent ref should match option
		for _, archName := range d.ListArches() {
			arch, err := d.GetArch(archName)
			assert.NoError(err)
			for _, typeName := range arch.ListImageTypes() {
				bp := &blueprint.Blueprint{}
				if strings.HasSuffix(typeName, "simplified-installer") {
					// simplified installers require installation device
					bp = &blueprint.Blueprint{
						Customizations: &blueprint.Customizations{
							InstallationDevice: "/dev/sda42",
						},
					}
				}

				imgType, err := arch.GetImageType(typeName)
				assert.NoError(err)

				ostreeOptions := ostree.ImageOptions{
					ParentRef: "test/x86_64/01",
					URL:       "https://example.com/repo",
				}
				options := distro.ImageOptions{OSTree: &ostreeOptions}
				m, _, err := imgType.Manifest(bp, options, nil, 0)
				assert.NoError(err)

				nrefs := 0
				for _, commits := range m.GetOSTreeSourceSpecs() {
					for _, commit := range commits {
						assert.Equal(options.OSTree.URL, commit.URL, "url does not match expected for image type %q\n", typeName)
						if typesWithPayload[typeName] {
							// payload ref should fall back to default
							assert.Equal(imgType.OSTreeRef(), commit.Ref, "ref does not match expected for image type %q\n", typeName)
						} else if typesWithParent[typeName] {
							// parent ref should match option
							assert.Equal(options.OSTree.ParentRef, commit.Ref, "ref does not match expected for image type %q\n", typeName)
						} else {
							// image type requires ostree commit but isn't specified: this shouldn't happen
							panic(fmt.Sprintf("image type %q requires ostree commit but is not covered by test", typeName))
						}
						nrefs++
					}
				}
				nexpected := 0
				if typesWithPayload[typeName] || typesWithParent[typeName] {
					// image types with payload or parent should return a ref
					nexpected = 1
				}
				assert.Equal(nexpected, nrefs, "incorrect ref count for image type %q\n", typeName)
			}
		}
	}

	{ // All options set: all refs should match the corresponding option
		for _, archName := range d.ListArches() {
			arch, err := d.GetArch(archName)
			assert.NoError(err)
			for _, typeName := range arch.ListImageTypes() {
				bp := &blueprint.Blueprint{}
				if strings.HasSuffix(typeName, "simplified-installer") {
					// simplified installers require installation device
					bp = &blueprint.Blueprint{
						Customizations: &blueprint.Customizations{
							InstallationDevice: "/dev/sda42",
						},
					}
				}

				imgType, err := arch.GetImageType(typeName)
				assert.NoError(err)

				ostreeOptions := ostree.ImageOptions{
					ImageRef:  "test/x86_64/01",
					ParentRef: "test/x86_64/02",
					URL:       "https://example.com/repo",
				}
				options := distro.ImageOptions{OSTree: &ostreeOptions}
				m, _, err := imgType.Manifest(bp, options, nil, 0)
				assert.NoError(err)

				nrefs := 0
				for _, commits := range m.GetOSTreeSourceSpecs() {
					for _, commit := range commits {
						assert.Equal(options.OSTree.URL, commit.URL, "url does not match expected for image type %q\n", typeName)
						if typesWithPayload[typeName] {
							// payload ref should match image ref
							assert.Equal(options.OSTree.ImageRef, commit.Ref, "ref does not match expected for image type %q\n", typeName)
						} else if typesWithParent[typeName] {
							// parent ref should match option
							assert.Equal(options.OSTree.ParentRef, commit.Ref, "ref does not match expected for image type %q\n", typeName)
						} else {
							// image type requires ostree commit but isn't specified: this shouldn't happen
							panic(fmt.Sprintf("image type %q requires ostree commit but is not covered by test", typeName))
						}
						nrefs++
					}
				}
				nexpected := 0
				if typesWithPayload[typeName] || typesWithParent[typeName] {
					// image types with payload or parent should return a ref
					nexpected = 1
				}
				assert.Equal(nexpected, nrefs, "incorrect ref count for image type %q\n", typeName)
			}
		}
	}

	{ // Parent set without URL: always causes error
		for _, archName := range d.ListArches() {
			arch, err := d.GetArch(archName)
			assert.NoError(err)
			for _, typeName := range arch.ListImageTypes() {
				bp := &blueprint.Blueprint{}
				if strings.HasSuffix(typeName, "simplified-installer") {
					// simplified installers require installation device
					bp = &blueprint.Blueprint{
						Customizations: &blueprint.Customizations{
							InstallationDevice: "/dev/sda42",
						},
					}
				}

				imgType, err := arch.GetImageType(typeName)
				assert.NoError(err)

				ostreeOptions := ostree.ImageOptions{
					ParentRef: "test/x86_64/02",
				}
				options := distro.ImageOptions{OSTree: &ostreeOptions}
				_, _, err = imgType.Manifest(bp, options, nil, 0)
				assert.Error(err)
			}
		}
	}
}

// ListTestedDistros returns a list of distro names that are explicitly tested
func ListTestedDistros(t *testing.T) []string {
	testRepoRegistry, err := reporegistry.NewTestedDefault()
	require.Nil(t, err)
	return testRepoRegistry.ListDistros()
}
