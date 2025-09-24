//go:build !cgo

package libvirt

import (
	"fmt"
	"io"

	"github.com/osbuild/images/pkg/cloud"
)

var _ = cloud.Uploader(&libvirtUploader{})

type libvirtUploader struct {
}

func NewUploader(connection string, pool string, volume string) (cloud.Uploader, error) {
	return nil, fmt.Errorf("cannot use libvirt: build without cgo")
}

func (lu *libvirtUploader) Check(status io.Writer) error {
	return fmt.Errorf("cannot use libvirt: build without cgo")
}

func (lu *libvirtUploader) UploadAndRegister(r io.Reader, uploadSize uint64, status io.Writer) (err error) {
	return fmt.Errorf("cannot use libvirt: build without cgo")
}
