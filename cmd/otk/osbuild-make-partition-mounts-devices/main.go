package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/osbuild/images/internal/otkdisk"
	"github.com/osbuild/images/pkg/osbuild"
)

type Input struct {
	Tree otkdisk.Data `json:"tree"`
}

type Output struct {
	RootMountName string                    `json:"root_mount_name"`
	Mounts        []osbuild.Mount           `json:"mounts"`
	Devices       map[string]osbuild.Device `json:"devices"`
}

func run(r io.Reader, w io.Writer) error {
	var inp Input
	if err := json.NewDecoder(r).Decode(&inp); err != nil {
		return err
	}
	if err := inp.Tree.Validate(); err != nil {
		return fmt.Errorf("cannot validate input data: %w", err)
	}

	rootMntName, mounts, devices, err := osbuild.GenMountsDevicesFromPT(inp.Tree.Const.Filename, inp.Tree.Const.Internal.PartitionTable)
	if err != nil {
		return err
	}

	out := map[string]interface{}{
		"tree": Output{
			RootMountName: rootMntName,
			Mounts:        mounts,
			Devices:       devices,
		},
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
