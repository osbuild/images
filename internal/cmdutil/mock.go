package cmdutil

import (
	"crypto/sha256"
	"fmt"

	"github.com/osbuild/images/pkg/ostree"
)

func MockOSTreeResolve(commitSource ostree.SourceSpec) ostree.CommitSpec {
	checksum := fmt.Sprintf("%x", sha256.Sum256([]byte(commitSource.URL+commitSource.Ref)))
	spec := ostree.CommitSpec{
		Ref:      commitSource.Ref,
		URL:      commitSource.URL,
		Checksum: checksum,
	}
	if commitSource.RHSM {
		spec.Secrets = "org.osbuild.rhsm.consumer"
	}
	return spec
}
