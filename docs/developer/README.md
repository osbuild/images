# Hacking on osbuild/images

## Local development environment

To build most binaries defined in `cmd` and run tests you will need to install `gpgme-devel`.
To generate manifests, you will need to install the `osbuild-depsolve-dnf` package.
To build images, you will also need to install `osbuild` and its sub-packages.

The full list of dependencies is:
- `gpgme-devel`
- `osbuild`
- `osbuild-depsolve-dnf`
- `osbuild-luks2`
- `osbuild-lvm2`
- `osbuild-ostree`
- `osbuild-selinux`

## Topics

- [Useful cmds](./cmds.md) for development and testing.
- [Manifest generation code](./code-manifest-generation.md)
