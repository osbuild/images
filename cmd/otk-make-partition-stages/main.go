package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/osbuild/images/pkg/osbuild"

	"github.com/osbuild/images/internal/otk"
)

type Input struct {
	Internal      InputInternal      `json:"internal"`
	Modifications InputModifications `json:"modifications"`
}

type InputInternal = otk.PartitionInternal

type InputModifications struct {
	Filename string `json:"filename"`
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
	stages = osbuild.GenImagePrepareStages(inp.Internal.PartitionTable, filename, partTool)
	return stages, nil
}

func run(r io.Reader, w io.Writer) error {
	var inp Input
	if err := json.NewDecoder(r).Decode(&inp); err != nil {
		return err
	}

	fname := "disk.img"
	if inp.Modifications.Filename != "" {
		fname = inp.Modifications.Filename
	}
	stages, err := makeImagePrepareStages(inp, fname)
	if err != nil {
		return fmt.Errorf("cannot make partition stages: %w", err)
	}

	outputJson, err := json.MarshalIndent(stages, "", "  ")
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
