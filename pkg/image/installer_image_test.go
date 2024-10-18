package image_test

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/osbuild/images/pkg/container"
	"github.com/osbuild/images/pkg/customizations/anaconda"
	"github.com/osbuild/images/pkg/customizations/kickstart"
	"github.com/osbuild/images/pkg/image"
	"github.com/osbuild/images/pkg/manifest"
	"github.com/osbuild/images/pkg/osbuild"
	"github.com/osbuild/images/pkg/ostree"
	"github.com/osbuild/images/pkg/platform"
	"github.com/osbuild/images/pkg/rpmmd"
	"github.com/osbuild/images/pkg/runner"
	"github.com/stretchr/testify/assert"
)

func mockPackageSets() map[string][]rpmmd.PackageSpec {
	return map[string][]rpmmd.PackageSpec{
		"build": {
			{
				Name:     "coreutils",
				Checksum: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			},
		},
		"os": {
			{
				Name:     "kernel",
				Checksum: "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
			},
		},
		"anaconda-tree": {
			{
				Name:     "kernel",
				Checksum: "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
			},
		},
	}
}

func mockContainerSpecs() map[string][]container.Spec {
	return map[string][]container.Spec{
		"bootiso-tree": {
			{
				Source:  "repo.example.com/container",
				Digest:  "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				ImageID: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			},
		},
	}
}

func mockOSTreeCommitSpecs() map[string][]ostree.CommitSpec {
	return map[string][]ostree.CommitSpec{
		"bootiso-tree": {
			{
				Ref: "test/ostree/3",
				URL: "http://localhost:8080/repo",
			},
		},
	}
}

var testPlatform = &platform.X86{
	BasePlatform: platform.BasePlatform{
		ImageFormat: platform.FORMAT_ISO,
	},
	BIOS:       true,
	UEFIVendor: "test",
}

const (
	product   = "Fedora"
	osversion = "40"
	isolabel  = "Fedora-40-Workstation-x86_64"
)

func TestContainerInstallerUnsetKSOptions(t *testing.T) {
	img := image.NewAnacondaContainerInstaller(container.SourceSpec{}, "")
	img.Product = product
	img.OSVersion = osversion
	img.ISOLabel = isolabel

	assert.NotNil(t, img)
	img.Platform = testPlatform
	mfs := instantiateAndSerialize(t, img, mockPackageSets(), mockContainerSpecs(), nil)
	assert.Contains(t, mfs, fmt.Sprintf(`"inst.ks=hd:LABEL=%s:/osbuild.ks"`, isolabel))
}

func TestContainerInstallerUnsetKSPath(t *testing.T) {
	img := image.NewAnacondaContainerInstaller(container.SourceSpec{}, "")
	img.Product = product
	img.OSVersion = osversion
	img.ISOLabel = isolabel

	assert.NotNil(t, img)
	img.Platform = testPlatform
	// set empty kickstart options (no path)
	img.Kickstart = &kickstart.Options{}

	mfs := instantiateAndSerialize(t, img, mockPackageSets(), mockContainerSpecs(), nil)
	assert.Contains(t, mfs, fmt.Sprintf(`"inst.ks=hd:LABEL=%s:/osbuild.ks"`, isolabel))
}

func TestContainerInstallerSetKSPath(t *testing.T) {
	img := image.NewAnacondaContainerInstaller(container.SourceSpec{}, "")
	img.Product = product
	img.OSVersion = osversion
	img.ISOLabel = isolabel

	assert.NotNil(t, img)
	img.Platform = testPlatform
	img.Kickstart = &kickstart.Options{
		Path: "/test.ks",
	}

	mfs := instantiateAndSerialize(t, img, mockPackageSets(), mockContainerSpecs(), nil)
	assert.Contains(t, mfs, fmt.Sprintf(`"inst.ks=hd:LABEL=%s:/test.ks"`, isolabel))
	assert.NotContains(t, mfs, "osbuild.ks") // no mention of the default value anywhere
}

func TestOSTreeInstallerUnsetKSPath(t *testing.T) {
	img := image.NewAnacondaOSTreeInstaller(ostree.SourceSpec{})
	img.Product = product
	img.OSVersion = osversion
	img.ISOLabel = isolabel

	assert.NotNil(t, img)
	img.Platform = testPlatform
	img.Kickstart = &kickstart.Options{
		// the ostree options must be non-nil
		OSTree: &kickstart.OSTree{},
	}

	mfs := instantiateAndSerialize(t, img, mockPackageSets(), nil, mockOSTreeCommitSpecs())
	assert.Contains(t, mfs, fmt.Sprintf(`"inst.ks=hd:LABEL=%s:/osbuild.ks"`, isolabel))
}

func TestOSTreeInstallerSetKSPath(t *testing.T) {
	img := image.NewAnacondaOSTreeInstaller(ostree.SourceSpec{})
	img.Product = product
	img.OSVersion = osversion
	img.ISOLabel = isolabel

	assert.NotNil(t, img)
	img.Platform = testPlatform
	img.Kickstart = &kickstart.Options{
		// the ostree options must be non-nil
		OSTree: &kickstart.OSTree{},
		Path:   "/test.ks",
	}

	mfs := instantiateAndSerialize(t, img, mockPackageSets(), nil, mockOSTreeCommitSpecs())
	assert.Contains(t, mfs, fmt.Sprintf(`"inst.ks=hd:LABEL=%s:/test.ks"`, isolabel))
	assert.NotContains(t, mfs, "osbuild.ks") // no mention of the default value anywhere
}

func TestTarInstallerUnsetKSOptions(t *testing.T) {
	img := image.NewAnacondaTarInstaller()
	img.Product = product
	img.OSVersion = osversion
	img.ISOLabel = isolabel

	assert.NotNil(t, img)
	img.Platform = testPlatform

	mfs := instantiateAndSerialize(t, img, mockPackageSets(), nil, nil)
	// the tar installer doesn't set a custom kickstart path unless the
	// unattended option is enabled, so the inst.ks option isn't set and
	// interactive-defaults.ks is used
	assert.Contains(t, mfs, fmt.Sprintf(`"inst.stage2=hd:LABEL=%s"`, isolabel))
	assert.Contains(t, mfs, fmt.Sprintf("%q", osbuild.KickstartPathInteractiveDefaults))
	assert.NotContains(t, mfs, "osbuild.ks") // no mention of the default (custom) value
}

func TestTarInstallerUnsetKSPath(t *testing.T) {
	img := image.NewAnacondaTarInstaller()
	img.Product = product
	img.OSVersion = osversion
	img.ISOLabel = isolabel

	assert.NotNil(t, img)
	img.Platform = testPlatform
	img.Kickstart = &kickstart.Options{}

	mfs := instantiateAndSerialize(t, img, mockPackageSets(), nil, nil)
	// the tar installer doesn't set a custom kickstart path unless the
	// unattended option is enabled, so the inst.ks option isn't set and
	// interactive-defaults.ks is used
	assert.Contains(t, mfs, fmt.Sprintf(`"inst.stage2=hd:LABEL=%s"`, isolabel))
	assert.Contains(t, mfs, fmt.Sprintf("%q", osbuild.KickstartPathInteractiveDefaults))
	assert.NotContains(t, mfs, "osbuild.ks") // no mention of the default (custom) value

	// enable unattended and retest
	img.Kickstart.Unattended = true
	mfs = instantiateAndSerialize(t, img, mockPackageSets(), nil, nil)
	assert.Contains(t, mfs, fmt.Sprintf(`"inst.ks=hd:LABEL=%s:/osbuild.ks"`, isolabel))
	assert.NotContains(t, mfs, osbuild.KickstartPathInteractiveDefaults)
}

func TestTarInstallerSetKSPath(t *testing.T) {
	img := image.NewAnacondaTarInstaller()
	img.Product = product
	img.OSVersion = osversion
	img.ISOLabel = isolabel

	assert.NotNil(t, img)
	img.Platform = testPlatform
	img.Kickstart = &kickstart.Options{
		Path: "/test.ks",
	}

	// enable unattended to use the custom kickstart path instead of interactive-defaults
	img.Kickstart.Unattended = true
	mfs := instantiateAndSerialize(t, img, mockPackageSets(), nil, nil)
	assert.Contains(t, mfs, fmt.Sprintf(`"inst.ks=hd:LABEL=%s:/test.ks"`, isolabel))
	assert.NotContains(t, mfs, "osbuild.ks") // no mention of the default value anywhere
}

func instantiateAndSerialize(t *testing.T, img image.ImageKind, packages map[string][]rpmmd.PackageSpec, containers map[string][]container.Spec, commits map[string][]ostree.CommitSpec) string {
	source := rand.NewSource(int64(0))
	// math/rand is good enough in this case
	/* #nosec G404 */
	rng := rand.New(source)

	mf := manifest.New(manifest.DISTRO_FEDORA)
	_, err := img.InstantiateManifest(mf, nil, &runner.CentOS{Version: 9}, rng)
	assert.NoError(t, err)

	mfs, err := mf.Serialize(packages, containers, commits, nil)
	assert.NoError(t, err)

	return string(mfs)
}

// NOTE(akoutsou):
// The following tests assert that the serialization of installer image kinds
// panics when the ISO-related properties aren't set (product name, product
// version, and ISO label). The panics happen in the stage validation, but we
// might want to catch them earlier (perhaps make them mandatory in the image
// kind or pipeline constructors) in the future.

func TestContainerInstallerPanics(t *testing.T) {
	assert := assert.New(t)
	img := image.NewAnacondaContainerInstaller(container.SourceSpec{}, "")
	img.Platform = testPlatform
	assert.PanicsWithError("org.osbuild.grub2.iso: product.name option is required", func() { instantiateAndSerialize(t, img, mockPackageSets(), mockContainerSpecs(), nil) })
	img.Product = product
	assert.PanicsWithError("org.osbuild.grub2.iso: product.version option is required", func() { instantiateAndSerialize(t, img, mockPackageSets(), mockContainerSpecs(), nil) })
	img.OSVersion = osversion
	assert.PanicsWithError("org.osbuild.grub2.iso: isolabel option is required", func() { instantiateAndSerialize(t, img, mockPackageSets(), mockContainerSpecs(), nil) })
}

func TestOSTreeInstallerPanics(t *testing.T) {
	assert := assert.New(t)
	img := image.NewAnacondaOSTreeInstaller(ostree.SourceSpec{})
	img.Platform = testPlatform
	img.Kickstart = &kickstart.Options{
		// the ostree options must be non-nil
		OSTree: &kickstart.OSTree{},
	}

	assert.PanicsWithError("org.osbuild.grub2.iso: product.name option is required",
		func() { instantiateAndSerialize(t, img, mockPackageSets(), nil, mockOSTreeCommitSpecs()) })

	img.Product = product
	assert.PanicsWithError("org.osbuild.grub2.iso: product.version option is required",
		func() { instantiateAndSerialize(t, img, mockPackageSets(), nil, mockOSTreeCommitSpecs()) })

	img.OSVersion = osversion
	assert.PanicsWithError("org.osbuild.grub2.iso: isolabel option is required",
		func() { instantiateAndSerialize(t, img, mockPackageSets(), nil, mockOSTreeCommitSpecs()) })
}

func TestTarInstallerPanics(t *testing.T) {
	assert := assert.New(t)
	img := image.NewAnacondaTarInstaller()
	img.Platform = testPlatform

	assert.PanicsWithError("org.osbuild.grub2.iso: product.name option is required",
		func() { instantiateAndSerialize(t, img, mockPackageSets(), nil, nil) })

	img.Product = product
	assert.PanicsWithError("org.osbuild.grub2.iso: product.version option is required",
		func() { instantiateAndSerialize(t, img, mockPackageSets(), nil, nil) })

	img.OSVersion = osversion
	assert.PanicsWithError("org.osbuild.grub2.iso: isolabel option is required",
		func() { instantiateAndSerialize(t, img, mockPackageSets(), nil, nil) })
}

func findAnacondaStageModules(t *testing.T, mf manifest.OSBuildManifest, legacyOptions bool) []interface{} {
	pipeline := findPipelineFromOsbuildManifest(t, mf, "anaconda-tree")
	assert.NotNil(t, pipeline)
	stage := findStageFromOsbuildPipeline(t, pipeline, "org.osbuild.anaconda")
	assert.NotNil(t, stage)
	anacondaStageOptions := stage["options"].(map[string]interface{})
	assert.NotNil(t, anacondaStageOptions)

	// NOTE: remove this condition and the legacyOptions function argument when
	// we remove support for RHEL 8.
	modulesKey := "activatable-modules"
	if legacyOptions {
		modulesKey = "kickstart-modules"
	}
	modules := anacondaStageOptions[modulesKey].([]interface{})
	assert.NotNil(t, modules)
	return modules
}

type testCase struct {
	enable   []string
	disable  []string
	expected []string
}

var moduleTestCases = map[string]testCase{
	"empty-args": {
		expected: []string{
			anaconda.ModulePayloads,
			anaconda.ModuleNetwork,
			anaconda.ModuleStorage,
		},
	},
	"no-op": {
		enable: []string{
			anaconda.ModulePayloads,
			anaconda.ModuleNetwork,
			anaconda.ModuleStorage,
		},
		expected: []string{
			anaconda.ModulePayloads,
			anaconda.ModuleNetwork,
			anaconda.ModuleStorage,
		},
	},
	"enable-users": {
		enable: []string{
			anaconda.ModuleUsers,
		},
		expected: []string{
			anaconda.ModulePayloads,
			anaconda.ModuleNetwork,
			anaconda.ModuleStorage,
			anaconda.ModuleUsers,
		},
	},
	"disable-storage": {
		disable: []string{
			anaconda.ModuleStorage,
		},
		expected: []string{
			anaconda.ModulePayloads,
			anaconda.ModuleNetwork,
		},
	},
	"enable-users-disable-storage": {
		enable: []string{
			anaconda.ModuleUsers,
		},
		disable: []string{
			anaconda.ModuleStorage,
		},
		expected: []string{
			anaconda.ModulePayloads,
			anaconda.ModuleNetwork,
			anaconda.ModuleUsers,
		},
	},
}

func TestContainerInstallerModules(t *testing.T) {
	for name := range moduleTestCases {
		tc := moduleTestCases[name]
		// Run each test case twice: once with activatable-modules and once with kickstart-modules.
		// Remove this when we drop support for RHEL 8.
		for _, legacy := range []bool{true, false} {
			t.Run(name, func(t *testing.T) {
				img := image.NewAnacondaContainerInstaller(container.SourceSpec{}, "")
				img.Product = product
				img.OSVersion = osversion
				img.ISOLabel = isolabel

				img.UseLegacyAnacondaConfig = legacy
				img.AdditionalAnacondaModules = tc.enable
				img.DisabledAnacondaModules = tc.disable

				assert.NotNil(t, img)
				img.Platform = testPlatform
				mfs := instantiateAndSerialize(t, img, mockPackageSets(), mockContainerSpecs(), nil)
				modules := findAnacondaStageModules(t, manifest.OSBuildManifest(mfs), legacy)
				assert.NotNil(t, modules)
				assert.ElementsMatch(t, modules, tc.expected)
			})
		}
	}
}

func TestOSTreeInstallerModules(t *testing.T) {
	for name := range moduleTestCases {
		tc := moduleTestCases[name]
		// Run each test case twice: once with activatable-modules and once with kickstart-modules.
		// Remove this when we drop support for RHEL 8.
		for _, legacy := range []bool{true, false} {
			t.Run(name, func(t *testing.T) {
				img := image.NewAnacondaOSTreeInstaller(ostree.SourceSpec{})
				img.Product = product
				img.OSVersion = osversion
				img.ISOLabel = isolabel
				img.Kickstart = &kickstart.Options{
					// the ostree options must be non-nil
					OSTree: &kickstart.OSTree{},
				}

				img.UseLegacyAnacondaConfig = legacy
				img.AdditionalAnacondaModules = tc.enable
				img.DisabledAnacondaModules = tc.disable

				assert.NotNil(t, img)
				img.Platform = testPlatform
				mfs := instantiateAndSerialize(t, img, mockPackageSets(), nil, mockOSTreeCommitSpecs())
				modules := findAnacondaStageModules(t, manifest.OSBuildManifest(mfs), legacy)
				assert.NotNil(t, modules)
				assert.ElementsMatch(t, modules, tc.expected)
			})
		}
	}
}

func TestTarInstallerModules(t *testing.T) {
	for name := range moduleTestCases {
		tc := moduleTestCases[name]
		// Run each test case twice: once with activatable-modules and once with kickstart-modules.
		// Remove this when we drop support for RHEL 8.
		for _, legacy := range []bool{true, false} {
			t.Run(name, func(t *testing.T) {
				img := image.NewAnacondaTarInstaller()
				img.Product = product
				img.OSVersion = osversion
				img.ISOLabel = isolabel

				img.UseLegacyAnacondaConfig = legacy
				img.AdditionalAnacondaModules = tc.enable
				img.DisabledAnacondaModules = tc.disable

				assert.NotNil(t, img)
				img.Platform = testPlatform
				mfs := instantiateAndSerialize(t, img, mockPackageSets(), nil, nil)
				modules := findAnacondaStageModules(t, manifest.OSBuildManifest(mfs), legacy)
				assert.NotNil(t, modules)
				assert.ElementsMatch(t, modules, tc.expected)
			})
		}
	}
}
