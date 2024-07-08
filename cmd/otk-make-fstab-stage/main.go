package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/osbuild/images/pkg/osbuild"

	"github.com/osbuild/images/internal/otkdisk"
)

type Input = otkdisk.Data

func makeFstabStages(inp Input) (stages *osbuild.Stage, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("cannot generate image prepare stages: %v", r)
		}
	}()

	pt := inp.Const.Internal.PartitionTable
	fstabStage := osbuild.NewFSTabStage(osbuild.NewFSTabStageOptions(pt))
	return fstabStage, nil
}

func run(r io.Reader, w io.Writer) error {
	var inp Input
	if err := json.NewDecoder(r).Decode(&inp); err != nil {
		return err
	}

	stage, err := makeFstabStages(inp)
	if err != nil {
		return fmt.Errorf("cannot make partition stages: %w", err)
	}

	outputJson, err := json.MarshalIndent(stage, "", "  ")
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
