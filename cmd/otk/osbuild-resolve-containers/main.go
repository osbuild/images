package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/osbuild/images/pkg/blueprint"
	"github.com/osbuild/images/pkg/container"
)

// All otk external inputs are nested under a top-level "tree"
type Tree struct {
	Tree Input `json:"tree"`
}

// Input represents the user-provided inputs that will be used to resolve a
// container image.
type Input struct {
	// The architecture of the container images to resolve.
	Arch string `json:"arch"`

	// List of container refs to resolve.
	Containers []blueprint.Container `json:"containers"`
}

// Output contains everything needed to write a manifest that requires pulling
// container images.
type Output struct {
	Const OutputConst `json:"const"`
}

type OutputConst struct {
	Containers []ContainerInfo `json:"containers"`
}

type ContainerInfo struct {
	Source string `json:"source"`

	// Digest of the manifest at the source.
	Digest string `json:"digest"`

	// Container image identifier.
	ImageID string `json:"imageid"`

	// Name to use inside the image.
	LocalName string `json:"local-name"`

	// Digest of the list manifest at the source
	ListDigest string `json:"list-digest,omitempty"`

	// The architecture of the image
	Arch string `json:"arch"`

	TLSVerify *bool `json:"tls-verify,omitempty"`
}

func run(r io.Reader, w io.Writer) error {
	var inputTree Tree
	if err := json.NewDecoder(r).Decode(&inputTree); err != nil {
		return err
	}

	resolver := container.NewResolver(inputTree.Tree.Arch)

	for _, bpSpec := range inputTree.Tree.Containers {
		srcSpec := container.SourceSpec{
			Source:    bpSpec.Source,
			Name:      bpSpec.Name,
			TLSVerify: bpSpec.TLSVerify,
		}
		resolver.Add(srcSpec)
	}

	containerSpecs, err := resolver.Finish()
	if err != nil {
		return err
	}
	containerInfos := make([]ContainerInfo, len(containerSpecs))
	for idx := range containerSpecs {
		spec := containerSpecs[idx]
		containerInfos[idx] = ContainerInfo{
			Source:     spec.Source,
			Digest:     spec.Digest,
			ImageID:    spec.ImageID,
			LocalName:  spec.LocalName,
			ListDigest: spec.ListDigest,
			Arch:       spec.Arch.String(),
			TLSVerify:  spec.TLSVerify,
		}
	}

	output := map[string]Output{
		"tree": {
			Const: OutputConst{
				Containers: containerInfos,
			},
		},
	}
	outputJson, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("cannot marshal response: %w", err)
	}
	fmt.Fprintf(w, "%s\n", outputJson)
	return nil
}

func main() {
	if err := run(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err.Error())
		os.Exit(1)
	}
}
