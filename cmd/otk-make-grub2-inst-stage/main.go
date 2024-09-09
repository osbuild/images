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
	Tree Tree `json:"tree"`
}

type Tree struct {
	Platform   string       `json:"platform"`
	Filesystem otkdisk.Data `json:"filesystem"`
}

func run(r io.Reader, w io.Writer) error {
	var inp Input
	if err := json.NewDecoder(r).Decode(&inp); err != nil {
		return err
	}

	opts := osbuild.NewGrub2InstStageOption(inp.Tree.Filesystem.Const.Filename, inp.Tree.Filesystem.Const.Internal.PartitionTable, inp.Tree.Platform)
	stage := osbuild.NewGrub2InstStage(opts)

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
