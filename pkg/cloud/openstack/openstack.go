//go:build cgo

package openstack

import (
	"fmt"
	"io"
	"os"

	"github.com/osbuild/images/pkg/cloud"

	"github.com/gophercloud/gophercloud"
	ostack "github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/imagedata"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"
)

var _ = cloud.Uploader(&openstackUploader{})

type openstackUploader struct {
	image           string
	diskFormat      string
	containerFormat string
}

type UploaderOptions struct {
	DiskFormat      string
	ContainerFormat string
}

func NewUploader(image string, opts *UploaderOptions) (cloud.Uploader, error) {
	return &openstackUploader{
		image:           image,
		diskFormat:      opts.DiskFormat,
		containerFormat: opts.ContainerFormat,
	}, nil
}

func (ou *openstackUploader) Check(status io.Writer) error {
	return nil
}

func (ou *openstackUploader) UploadAndRegister(r io.Reader, uploadSize uint64, status io.Writer) (err error) {
	fmt.Fprintf(status, "Uploading to OpenStack...\n")

	opts, err := ostack.AuthOptionsFromEnv()
	if err != nil {
		return fmt.Errorf("Failed to read OpenStack ENV variables. Please source the OpenStack RC file: %w", err)
	}

	// This is needed otherwise we get the following error when authenticating:
	//	   You must provide exactly one of DomainID or DomainName to
	//	   authenticate by Username
	// Even with an RC file that works perfectly fine with `openstack token issue`
	if opts.DomainName == "" {
		opts.DomainName = os.Getenv("OS_USER_DOMAIN_NAME")
	}

	provider, err := ostack.AuthenticatedClient(opts)
	if err != nil {
		return fmt.Errorf("Failed to authenticate to OpenStack: %w", err)
	}

	client, err := ostack.NewImageServiceV2(provider, gophercloud.EndpointOpts{})
	if err != nil {
		return fmt.Errorf("Failed to initialize the client: %w", err)
	}

	createOpts := images.CreateOpts{
		Name:            ou.image,
		DiskFormat:      ou.diskFormat,
		ContainerFormat: ou.containerFormat,
	}
	img, err := images.Create(client, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Failed to create the image metadata: %w", err)
	}

	err = imagedata.Upload(client, img.ID, r).ExtractErr()
	if err != nil {
		return fmt.Errorf("Failed to upload the image: %w", err)
	}

	// This would glitch the progressbar, but once it gets fixed, we would
	// like to print this message
	// fmt.Printf("Created image: %s (ID: %s)\n", img.Name, img.ID)

	return nil
}
