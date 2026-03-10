package container

import "github.com/osbuild/images/pkg/customizations/fsnode"

// NetworkBackend is the type of network backend used by Podman.
type NetworkBackend string

const (
	NetworkBackendCNI     NetworkBackend = "cni"
	NetworkBackendNetavark NetworkBackend = "netavark"
)

// GenDefaultNetworkBackendFile creates an fsnode.File that writes the given
// network backend name to /var/lib/containers/storage/defaultNetworkBackend.
//
// Certain versions of Podman fall back to 'cni' when they find existing
// container images in the system storage, assuming a migration from an older
// version. Writing this file prevents that behavior and forces Podman to use
// the specified backend.
func GenDefaultNetworkBackendFile(backend NetworkBackend) (*fsnode.File, error) {
	file, err := fsnode.NewFile("/var/lib/containers/storage/defaultNetworkBackend", nil, nil, nil, []byte(backend))
	if err != nil {
		return nil, err
	}
	return file, nil
}
