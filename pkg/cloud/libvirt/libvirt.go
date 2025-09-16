package libvirt

import (
	"fmt"
	"github.com/osbuild/images/pkg/cloud"
	"io"
	"os"
	"os/exec"
)

type libvirtUploader struct {
	connection string
	pool       string
	volume     string
}

func NewUploader(connection string, pool string, volume string) (cloud.Uploader, error) {
	return &libvirtUploader{
		connection: connection,
		pool:       pool,
		volume:     volume,
	}, nil
}

func (lu *libvirtUploader) CanShowProgress() bool {
	return false
}

func (lu *libvirtUploader) Check(status io.Writer) error {
	cmd := exec.Command("virsh", "help")
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("Can't execute virsh:\n%v", err)
	}
	return nil
}

func (lu *libvirtUploader) UploadAndRegister(r io.Reader, f *os.File, status io.Writer) (err error) {
	fmt.Fprintf(status, "Uploading to libvirt...\n")

	err = lu.Check(io.Discard)
	if err != nil {
		return err
	}
	err = lu.VolCreate()
	if err != nil {
		return err
	}
	fmt.Fprintf(status, "Starting to upload %s\n", f.Name())
	fmt.Fprintf(status, "Progressbar is not supported.\n")
	fmt.Fprintf(status, "This can take a while...\n")

	err = lu.VolUpload(f)
	if err != nil {
		return err
	}
	fmt.Fprintf(status, "File %s uploaded to %s\n", f.Name(), lu.connection)
	return nil
}

func (lu *libvirtUploader) VolCreate() (err error) {
	cmd := exec.Command(
		"virsh",
		"--connect", lu.connection,
		"vol-create-as",
		"--pool", lu.pool,
		lu.volume,
		"1M",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("virsh vol-create-as failed: %v", err)
	}
	return nil
}

func (lu *libvirtUploader) VolUpload(f *os.File) (err error) {
	cmd := exec.Command(
		"virsh",
		"--connect", lu.connection,
		"vol-upload",
		"--pool", lu.pool,
		lu.volume,
		"--sparse",
		"--file", f.Name(),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("virsh vol-upload failed: %v", err)
	}
	return nil
}
