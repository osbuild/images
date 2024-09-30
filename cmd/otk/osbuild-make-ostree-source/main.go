package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/osbuild/images/pkg/osbuild"
	"github.com/osbuild/images/pkg/ostree"
)

// TODO: move structs to common package with resolver external

type Input struct {
	Tree InputTree `json:"tree"`
}

type InputTree struct {
	Const InputConst `json:"const"`
}

type InputConst struct {
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

type Output struct {
	OSTreeSource osbuild.OSTreeSource `json:"org.osbuild.ostree"`
}

func run(r io.Reader, w io.Writer) error {
	var inp Input
	if err := json.NewDecoder(r).Decode(&inp); err != nil {
		return err
	}

	ostreeSource := osbuild.NewOSTreeSource()
	ostreeSource.AddItem(ostree.CommitSpec{
		Ref:      inp.Tree.Const.Ref,
		URL:      inp.Tree.Const.URL,
		Secrets:  inp.Tree.Const.Secrets,
		Checksum: inp.Tree.Const.Checksum,
	})

	output := Output{
		OSTreeSource: *ostreeSource,
	}
	out := map[string]interface{}{
		"tree": output,
	}
	outputJson, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Errorf("cannot marshal response: %w", err)
	}
	fmt.Fprintf(w, "%s\n", outputJson)
	return nil
}

func main() {
	if err := run(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v", err.Error())
		os.Exit(1)
	}
}
