package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/osbuild/images/internal/otkostree"
	"github.com/osbuild/images/pkg/manifestgen/manifestmock"
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

	// MTLS information. Will be ignored if RHSM is set.
	MTLS *struct {
		CA         string `json:"ca"`
		ClientCert string `json:"client_cert"`
		ClientKey  string `json:"client_key"`
	} `json:"mtls,omitempty"`

	// HTTP proxy to use when fetching the ref.
	Proxy string `json:"proxy,omitempty"`
}

// Output contains everything needed to write a manifest that requires pulling
// an ostree commit.
type Output struct {
	Const otkostree.ResolvedConst `json:"const"`
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

	sourceSpec := ostree.SourceSpec{
		URL:   inputTree.Tree.URL,
		Ref:   inputTree.Tree.Ref,
		RHSM:  inputTree.Tree.RHSM,
		Proxy: inputTree.Tree.Proxy,
	}

	if inputTree.Tree.MTLS != nil {
		sourceSpec.MTLS = &ostree.MTLS{}
		sourceSpec.MTLS.CA = inputTree.Tree.MTLS.CA
		sourceSpec.MTLS.ClientCert = inputTree.Tree.MTLS.ClientCert
		sourceSpec.MTLS.ClientKey = inputTree.Tree.MTLS.ClientKey
	}

	var commitSpec ostree.CommitSpec
	if !underTest() {
		var err error
		commitSpec, err = ostree.Resolve(sourceSpec)
		if err != nil {
			return fmt.Errorf("failed to resolve ostree commit: %w", err)
		}
	} else {
		commitSpec = manifestmock.OSTreeResolve(sourceSpec)
	}

	output := map[string]Output{
		"tree": {
			Const: otkostree.ResolvedConst{
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
