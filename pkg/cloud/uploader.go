package cloud

import (
	"io"
	"os"
)

// Uploader is an interface that is returned from the actual
// cloud implementation. The uploader will be parameterized
// by the actual cloud implemntation, e.g.
//
//	awscloud.NewUploader(region, bucket, image) Uploader
//
// which is outside the scope of this interface.
type Uploader interface {
	// Check can be called before the actual upload to ensure
	// all permissions are correct
	Check(status io.Writer) error

	// UploadAndRegister will upload the given image from
	// the reader and write status message to the given
	// status writer.
	// To implement progress a proxy reader can be used.
	UploadAndRegister(r io.Reader, f *os.File, status io.Writer) error

	// Can this uploader show a progressbar while uploading the image?
	CanShowProgress() bool
}
