package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/osbuild/images/pkg/arch"
	"github.com/osbuild/images/pkg/osbuildmonitor"
)

// XXX: merge back into images/pkg/osbuild/osbuild-exec.go or
// into osbuildmonitor
func runOSBuild(manifest []byte, store, outputDirectory string, exports, extraEnv []string) error {
	rp, wp, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("cannot create pipe for osbuild: %w", err)
	}
	defer rp.Close()
	defer wp.Close()

	cmd := exec.Command(
		"osbuild",
		"--store", store,
		"--output-directory", outputDirectory,
		"--monitor=JSONSeqMonitor",
		"--monitor-fd=3",
		"-",
	)
	for _, export := range exports {
		cmd.Args = append(cmd.Args, "--export", export)
	}

	cmd.Env = append(os.Environ(), extraEnv...)
	cmd.Stdin = bytes.NewBuffer(manifest)
	cmd.Stderr = os.Stderr
	// we could use "--json" here and would get the build-result
	// exported here
	cmd.Stdout = nil
	cmd.ExtraFiles = []*os.File{wp}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting osbuild: %v", err)
	}
	wp.Close()

	scanner := osbuildmonitor.NewStatusScanner(rp)
	for {
		status, err := scanner.Status()
		if err != nil {
			return err
		}
		if status == nil {
			break
		}
		// XXX: add progress bar
		fmt.Printf("[%s] %s\n", status.Timestamp.Format("2006-01-02 15:04:05"), status.Trace)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("error running osbuild: %w", err)
	}

	return nil
}

func buildImage(out io.Writer, distroName, imgTypeStr, outputFilename string) error {
	// cross arch building is not possible, we would have to download
	// a pre-populated buildroot (tar,container) with rpm for that
	archStr := arch.Current().String()
	filterResult, err := getOneImage(distroName, imgTypeStr, archStr)
	if err != nil {
		return err
	}
	imgType := filterResult.ImgType

	var mf bytes.Buffer
	opts := &genManifestOptions{
		OutputFilename: outputFilename,
	}
	if err := outputManifest(&mf, distroName, imgTypeStr, archStr, opts); err != nil {
		return err
	}

	osbuildStoreDir := ".store"
	outputDir := "."
	buildName := fmt.Sprintf("%s-%s-%s", distroName, imgTypeStr, archStr)
	jobOutputDir := filepath.Join(outputDir, buildName)
	return runOSBuild(mf.Bytes(), osbuildStoreDir, jobOutputDir, imgType.Exports(), nil)
}
