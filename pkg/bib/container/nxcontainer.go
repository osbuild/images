package container

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

// Never add anything to this code that actually starts the the container
// or execs code from it

// NxContainer is a container non-executing container that runs *no* code
// from the container and only creates/mounts it.
type NxContainer struct {
	id   string
	root string
	arch string
}

func NewNxContainer(ref string) (*NxContainer, error) {
	output, err := exec.Command("podman", "create", ref).Output()
	if err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("creating %s container failed: %w\nstderr:\n%s", ref, e, e.Stderr)
		}
		return nil, fmt.Errorf("creating %s container failed with generic error: %w", ref, err)
	}

	c := &NxContainer{
		id: strings.TrimSpace(string(output)),
	}
	// not all containers set {{.Architecture}} so fallback
	c.arch, err = findContainerArchInspect(c.id, ref)
	if err != nil {
		return nil, err
	}

	/* #nosec G204 */
	output, err = exec.Command("podman", "mount", c.id).Output()
	if err != nil {
		if err, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("mounting %s container failed: %w\nstderr:\n%s", ref, err, err.Stderr)
		}
		return nil, fmt.Errorf("mounting %s container failed with generic error: %w", ref, err)
	}
	c.root = strings.TrimSpace(string(output))

	return c, err
}

func (c *NxContainer) Stop() error {
	/* #nosec G204 */
	if output, err := exec.Command("podman", "umount", c.id).CombinedOutput(); err != nil {
		return fmt.Errorf("umount %s nxcontainer failed: %w\noutput:\n%s", c.id, err, output)
	}

	/* #nosec G204 */
	if output, err := exec.Command("podman", "rm", c.id).CombinedOutput(); err != nil {
		return fmt.Errorf("rm %s nxcontainer failed: %w\noutput:\n%s", c.id, err, output)
	}
	return nil
}

// Root returns the root directory of the nxcontainer as available on the host.
func (c *NxContainer) Root() string {
	return c.root
}

// Arch returns the architecture of the container
func (c *NxContainer) Arch() string {
	return c.arch
}

// Reads a file from the container
func (c *NxContainer) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(filepath.Join(c.root, path))
}

// DefaultRootfsType returns the default rootfs type (e.g. "ext4") as
// specified by the bootc container install configuration. An empty
// string is valid and means the container sets no default.
func (c *NxContainer) DefaultRootfsType() (string, error) {
	// TODO:
	//  add --rootdir or similar to bootc install print-configuration
	// (c.f. lib/src/install/config.rs)
	// so that we can drop this reimplementation
	var bootcConfig struct {
		Install struct {
			RootFsType string `toml:"root-fs-type"`
			Filesystem struct {
				Root struct {
					Type string `toml:"type"`
				} `toml:"root"`
			} `toml:"filesystem"`
		} `toml:"install"`
	}

	// this is extremly simple but we get the desired "merging"
	// behavior by just not clearing the previous values, so
	// if "root-fs-type" is set on both "/etc" and "/run" the
	// run version will just override what we had before which
	// is what we want
	bases := []string{"/usr/lib", "/usr/local/lib", "/etc", "/run"}
	for _, base := range bases {
		frags, err := filepath.Glob(filepath.Join(c.root, base, "bootc/install/*.toml"))
		if err != nil {
			return "", err
		}
		sort.Strings(frags)
		for _, frag := range frags {
			if _, err := toml.DecodeFile(frag, &bootcConfig); err != nil {
				return "", fmt.Errorf("failed to decode bootc configuration %v: %w", frag, err)
			}
		}
	}

	// filesystem.root.type is the preferred way instead of the old root-fs-type top-level key.
	// See https://github.com/containers/bootc/commit/558cd4b1d242467e0ffec77fb02b35166469dcc7
	fsType := bootcConfig.Install.Filesystem.Root.Type
	if fsType == "" {
		fsType = bootcConfig.Install.RootFsType
	}
	// Note that these are the only filesystems that the "images" library
	// knows how to handle, i.e. how to construct the required osbuild
	// stages for.
	// TODO: move this into a helper in "images" so that there is only
	// a single place that needs updating when we add e.g. btrfs or
	// bcachefs
	supportedFS := []string{"ext4", "xfs", "btrfs"}

	if fsType == "" {
		return "", nil
	}
	if !slices.Contains(supportedFS, fsType) {
		return "", fmt.Errorf("unsupported root filesystem type: %s, supported: %s", fsType, strings.Join(supportedFS, ", "))
	}

	return fsType, nil
}
