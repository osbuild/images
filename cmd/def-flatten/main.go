package main

import (
	"os"
	"path"

	"github.com/osbuild/images/pkg/definition"
	"gopkg.in/yaml.v3"
)

func main() {
	dir := os.DirFS(".")
	path := path.Clean(os.Args[1])
	merged, err := definition.MergeConfig(dir, path)
	if err != nil {
		panic(err)
	}

	err = yaml.NewEncoder(os.Stdout).Encode(merged)
	if err != nil {
		panic(err)
	}
}
