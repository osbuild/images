package main

import (
	"os"
	"path"

	"github.com/osbuild/images/pkg/spec"
)

func main() {
	dir := os.DirFS(".")
	path := path.Clean(os.Args[1])
	merged, err := spec.MergeConfig(dir, path)
	if err != nil {
		panic(err)
	}

	os.Stdout.Write(merged)
}
