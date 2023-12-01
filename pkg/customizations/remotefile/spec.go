package remotefile

import "github.com/osbuild/images/internal/worker/clienterrors"

type Spec struct {
	URL             string
	Content         []byte
	ResolutionError *clienterrors.Error
}
