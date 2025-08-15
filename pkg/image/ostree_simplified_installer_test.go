package image_test

import (
	"testing"

	"github.com/osbuild/images/internal/testdisk"
	"github.com/osbuild/images/pkg/arch"
	"github.com/osbuild/images/pkg/image"
	"github.com/osbuild/images/pkg/manifest"
	"github.com/osbuild/images/pkg/ostree"
	"github.com/osbuild/images/pkg/platform"
	"github.com/stretchr/testify/assert"
)

func TestSimplifiedInstallerDracut(t *testing.T) {
	commit := ostree.SourceSpec{}
	ostreeDiskImage := image.NewOSTreeDiskImageFromCommit(commit)
	ostreeDiskImage.PartitionTable = testdisk.MakeFakePartitionTable("/")
	ostreeDiskImage.Platform = &platform.Data{Arch: arch.ARCH_X86_64}
	img := image.NewOSTreeSimplifiedInstaller(ostreeDiskImage, "")
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
	img.Platform = testPlatform
	mfs := instantiateAndSerialize(t, img, packageSets, nil, commitSpec)
	modules, addModules, drivers, addDrivers := findDracutStageOptions(t, manifest.OSBuildManifest(mfs), "coi-tree")
	assert.NotNil(t, modules)
	assert.Nil(t, addModules)
	assert.Nil(t, drivers)
	assert.NotNil(t, addDrivers)

	assert.Subset(t, modules, testModules)
	assert.Subset(t, addDrivers, testDrivers)
}
