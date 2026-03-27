package osbuild

import (
	"bytes"
	"fmt"
	"os/exec"
)

var osbuildStoreCmd = "/usr/libexec/osbuild-store"

func RunOSBuildStore(manifest []byte, sourceStore, tgtStore string) (string, error) {
	var stdoutBuffer bytes.Buffer
	// nolint: gosec
	cmd := exec.Command(
		osbuildStoreCmd,
		"export-sources",
		"--source-store", sourceStore,
		"--target-store", tgtStore,
		"-",
	)
	cmd.Stdin = bytes.NewBuffer(manifest)
	cmd.Stdout = &stdoutBuffer

	err := cmd.Start()
	if err != nil {
		return "", fmt.Errorf("error starting osbuild-store: %v", err)
	}
	err = cmd.Wait()
	if err != nil {
		return stdoutBuffer.String(), fmt.Errorf("error waiting for osbuild-store: %v", err)
	}
	return stdoutBuffer.String(), nil
}
