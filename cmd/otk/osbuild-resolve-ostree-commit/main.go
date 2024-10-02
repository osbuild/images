package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/osbuild/images/internal/cmdutil"
	"github.com/osbuild/images/pkg/ostree"
)

// All otk external inputs are nested under a top-level "tree"
type Tree struct {
	Tree Input `json:"tree"`
}

// Input represents the user-provided inputs that will be used to resolve an
// ostree commit ID.
type Input struct {
	// URL of the repo where the commit can be fetched.
	URL string `json:"url"`

	// Ref to resolve.
	Ref string `json:"ref"`

	// Whether to use RHSM secrets when resolving and fetching the commit.
	RHSM bool `json:"rhsm,omitempty"`
}

// Output contains everything needed to write a manifest that requires pulling
// an ostree commit.
type Output struct {
	Const OutputConst `json:"const"`
}

type OutputConst struct {
	// Ref of the commit (can be empty).
	Ref string `json:"ref,omitempty"`

	// URL of the repo where the commit can be fetched.
	URL string `json:"url"`

	// Secrets type to use when pulling the ostree commit content
	// (e.g. org.osbuild.rhsm.consumer).
	Secrets string `json:"secrets,omitempty"`

	// Checksum of the commit.
	Checksum string `json:"checksum"`
}

// for mocking in testing
var osLookupEnv = os.LookupEnv

func underTest() bool {
	testVar, found := osLookupEnv("OTK_UNDER_TEST")
	return found && testVar == "1"
}

func run(r io.Reader, w io.Writer) error {
	var inputTree Tree
	if err := json.NewDecoder(r).Decode(&inputTree); err != nil {
		return err
	}

	sourceSpec := ostree.SourceSpec(inputTree.Tree)

	var commitSpec ostree.CommitSpec
	if !underTest() {
		var err error
		commitSpec, err = ostree.Resolve(sourceSpec)
		if err != nil {
			return fmt.Errorf("failed to resolve ostree commit: %w", err)
		}
	} else {
		commitSpec = cmdutil.MockOSTreeResolve(sourceSpec)
	}

	output := map[string]Output{
		"tree": {
			Const: OutputConst{
				Ref:      commitSpec.Ref,
				URL:      commitSpec.URL,
				Secrets:  commitSpec.Secrets,
				Checksum: commitSpec.Checksum,
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
