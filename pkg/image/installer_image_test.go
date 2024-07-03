package image_test

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/osbuild/images/pkg/container"
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
}

func instantiateAndSerialize(t *testing.T, img image.ImageKind, packages map[string][]rpmmd.PackageSpec, containers map[string][]container.Spec, commits map[string][]ostree.CommitSpec) string {
	source := rand.NewSource(int64(0))
	// math/rand is good enough in this case
	/* #nosec G404 */
	rng := rand.New(source)

	mf := manifest.New()
	_, err := img.InstantiateManifest(&mf, nil, &runner.CentOS{Version: 9}, rng)
	assert.NoError(t, err)

	mfs, err := mf.Serialize(packages, containers, commits, nil)
	assert.NoError(t, err)

	return string(mfs)
}
