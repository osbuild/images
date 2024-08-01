package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/osbuild/images/pkg/osbuild"

	"github.com/osbuild/images/internal/otkdisk"
)

type Input struct {
	Tree otkdisk.Data `json:"tree"`
}

func run(r io.Reader, w io.Writer) error {
	var inp Input
	if err := json.NewDecoder(r).Decode(&inp); err != nil {
		return err
	}

	opts, err := osbuild.NewFSTabStageOptions(inp.Tree.Const.Internal.PartitionTable)
	if err != nil {
		return fmt.Errorf("cannot make partition stages: %w", err)
	}
	stage := osbuild.NewFSTabStage(opts)

	out := map[string]interface{}{
		"tree": stage,
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
