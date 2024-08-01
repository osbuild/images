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

func makeImagePrepareStages(inp Input, filename string) (stages []*osbuild.Stage, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("cannot generate image prepare stages: %v", r)
		}
	}()

	// rhel7 uses PTSgdisk, if we ever need to support this, make this
	// configurable
	partTool := osbuild.PTSfdisk
	stages = osbuild.GenImagePrepareStages(inp.Tree.Const.Internal.PartitionTable, inp.Tree.Const.Filename, partTool)
	return stages, nil
}

func run(r io.Reader, w io.Writer) error {
	var inp Input
	if err := json.NewDecoder(r).Decode(&inp); err != nil {
		return err
	}

	stages, err := makeImagePrepareStages(inp, inp.Tree.Const.Filename)
	if err != nil {
		return fmt.Errorf("cannot make partition stages: %w", err)
	}

	stagesInTree := map[string]interface{}{
		"tree": stages,
	}
	outputJson, err := json.MarshalIndent(stagesInTree, "", "  ")
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
