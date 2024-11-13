package main

import (
	"io"
	"log"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/osbuild/images/pkg/arch"
)

var osStdout io.Writer = os.Stdout

func cmdListImages(cmd *cobra.Command, args []string) error {
	filter, err := cmd.Flags().GetStringArray("filter")
	if err != nil {
		return err
	}
	format, err := cmd.Flags().GetString("format")
	if err != nil {
		return err
	}

	return listImages(osStdout, format, filter)
}

func cmdManifest(cmd *cobra.Command, args []string) error {
	// support prefixes to make it easy to copy/paste from list-images
	distroName := strings.TrimPrefix(args[0], "distro:")
	imgType := strings.TrimPrefix(args[1], "type:")
	var archStr string
	if len(args) > 2 {
		archStr = strings.TrimPrefix(args[2], "arch:")
	} else {
		archStr = arch.Current().String()
	}

	return outputManifest(osStdout, distroName, imgType, archStr, nil)
}

func cmdBuild(cmd *cobra.Command, args []string) error {
	// support prefixes to make it easy to copy/paste from list-images
	distroName := strings.TrimPrefix(args[0], "distro:")
	imgType := strings.TrimPrefix(args[1], "type:")
	outputFilename, err := cmd.Flags().GetString("filename")
	if err != nil {
		return err
	}

	return buildImage(osStdout, distroName, imgType, outputFilename)
}

func run() error {
	// images logs a bunch of stuff to Debug/Info that we we do not
	// want to show
	logrus.SetLevel(logrus.WarnLevel)

	rootCmd := &cobra.Command{
		Use:   "image-builder",
		Short: "Build operating system images from a given blueprint",
		Long: `Build operating system images from a given blueprint

Image-builder builds operating system images for a range of predefined
operating sytsems like centos and RHEL with easy customizations support.`,
	}

	// XXX: this will list 802 images right now, we need a sensible
	// default here, maybe without --filter just list all available
	// distro names?
	listImagesCmd := &cobra.Command{
		Use:          "list-images",
		Short:        "List buildable images, use --filter to limit further",
		RunE:         cmdListImages,
		SilenceUsage: true,
	}
	listImagesCmd.Flags().StringArray("filter", nil, "Filter distributions by a specific criteria")
	listImagesCmd.Flags().String("format", "", "Output in a specific format (text,json)")
	rootCmd.AddCommand(listImagesCmd)

	manifestCmd := &cobra.Command{
		Use:          "manifest <distro> <image-type> [<arch>]",
		Short:        "Build manifest for the given distro/image-type, e.g. centos-9 qcow2",
		RunE:         cmdManifest,
		SilenceUsage: true,
		// XXX: show error with available types if only one arg given
		Args:   cobra.MinimumNArgs(2),
		Hidden: true,
	}
	rootCmd.AddCommand(manifestCmd)

	buildCmd := &cobra.Command{
		Use:          "build <distro> <image-type>",
		Short:        "Build the given distro/image-type, e.g. centos-9 qcow2",
		RunE:         cmdBuild,
		SilenceUsage: true,
		// XXX: show error with available types if only one arg given
		Args: cobra.ExactArgs(2),
	}
	// XXX: add this for "manifest" too in a nice way
	buildCmd.Flags().String("filename", "", "Output as a specific filename")
	// XXX: add --output=text,json and streaming
	rootCmd.AddCommand(buildCmd)

	return rootCmd.Execute()
}

func main() {
	if err := run(); err != nil {
		log.Fatalf("error: %s", err)
	}
}
