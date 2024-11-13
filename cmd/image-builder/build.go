package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/osbuild/images/pkg/arch"
	"github.com/osbuild/images/pkg/osbuild"
)

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
	// XXX: support stremaing via statusWriter
	_, err = osbuild.RunOSBuild(mf.Bytes(), osbuildStoreDir, jobOutputDir, imgType.Exports(), nil, nil, false, os.Stderr)
	return err
}
