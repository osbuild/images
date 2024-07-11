package manifest

import (
	"bytes"
	"fmt"
	"os/exec"

	"github.com/osbuild/images/pkg/blueprint"
	"github.com/osbuild/images/pkg/container"
	"github.com/osbuild/images/pkg/ostree"
	"github.com/osbuild/images/pkg/rpmmd"
)

type OTKManifest struct {
	// entrypoint yaml file for this otk manifest
	entrypoint string

	blueprint blueprint.Blueprint
}

func NewOTK(path string, bp blueprint.Blueprint) *OTKManifest {
	return &OTKManifest{entrypoint: path, blueprint: bp}
}

func (m *OTKManifest) addPipeline(p Pipeline) {}

func (m *OTKManifest) GetPackageSetChains() map[string][]rpmmd.PackageSet {
	return nil
}

func (m *OTKManifest) GetContainerSourceSpecs() map[string][]container.SourceSpec {
	return nil
}

func (m *OTKManifest) GetOSTreeSourceSpecs() map[string][]ostree.SourceSpec {
	return nil
}

func (m *OTKManifest) Serialize(_ map[string][]rpmmd.PackageSpec, _ map[string][]container.Spec, _ map[string][]ostree.CommitSpec, _ map[string][]rpmmd.RepoConfig) (OSBuildManifest, error) {
	cmd := exec.Command("otk", "compile", "--target=osbuild", m.entrypoint)
	var stdoutBuffer bytes.Buffer
	var stderrBuffer bytes.Buffer
	cmd.Stdout = &stdoutBuffer
	cmd.Stderr = &stderrBuffer
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("compiling omnifest: %v %s", err, stderrBuffer.String())

	}
	return OSBuildManifest(stdoutBuffer.Bytes()), nil
}

func (m *OTKManifest) GetCheckpoints() []string {
	return nil
}

func (m *OTKManifest) GetExports() []string {
	return nil
}
