package image_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osbuild/images/internal/testdisk"
	"github.com/osbuild/images/pkg/arch"
	"github.com/osbuild/images/pkg/image"
	"github.com/osbuild/images/pkg/manifest"
	"github.com/osbuild/images/pkg/ostree"
	"github.com/osbuild/images/pkg/platform"
)

func TestSimplifiedInstallerDracut(t *testing.T) {
	commit := ostree.SourceSpec{}
	platform := &platform.Data{Arch: arch.ARCH_X86_64}
	ostreeDiskImage := image.NewOSTreeDiskImageFromCommit(platform, "filename", commit)
	ostreeDiskImage.PartitionTable = testdisk.MakeFakePartitionTable("/")
	img := image.NewOSTreeSimplifiedInstaller(testPlatform, "filename", ostreeDiskImage, "")
	img.Product = product
	img.OSVersion = osversion
	img.ISOLabel = isolabel

	testModules := []string{"test-module"}
	testDrivers := []string{"test-driver"}

	img.AdditionalDracutModules = testModules
	img.AdditionalDrivers = testDrivers

	commitSpec := map[string][]ostree.CommitSpec{
		"ostree-deployment": {
			{
				Ref: "test/ostree/3",
				URL: "http://localhost:8080/repo",
			},
		},
	}

	packageSets := mockPackageSets()
	packageSets["coi-tree"] = packageSets["os"]

	assert.NotNil(t, img)
	mfs := instantiateAndSerialize(t, img, packageSets, nil, commitSpec)
	modules, addModules, drivers, addDrivers := findDracutStageOptions(t, manifest.OSBuildManifest(mfs), "coi-tree")
	assert.NotNil(t, modules)
	assert.Nil(t, addModules)
	assert.Nil(t, drivers)
	assert.NotNil(t, addDrivers)

	assert.Subset(t, modules, testModules)
	assert.Subset(t, addDrivers, testDrivers)
}
