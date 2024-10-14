Images
======

This repository is, primarily, a Go library for generating osbuild manifests
([more details here](./docs/developer/code-manifest-generation.md)).
It also has some libraries for uploading artifacts to cloud platforms and Koji.
The binaries implemented in `cmd/` are for development and testing purposes and not part of the library.

## Project

 * **Website**: <https://www.osbuild.org>
 * **Bug Tracker**: <https://github.com/osbuild/images/issues>
* **Discussions**: https://github.com/orgs/osbuild/discussions
 * **Matrix (chat)**: [Image Builder channel on Fedora Chat](https://matrix.to/#/#image-builder:fedoraproject.org?web-instance[element.io]=chat.fedoraproject.org)
 * **Changelog**: <https://github.com/osbuild/images/releases>

### Principles
1. The image definitions API is internal and can therefore be broken. The blueprint API is the stable API.
2. Nonsensical manifests should not compile (at the Golang level).
3. OSBuild units (stages, sources, inputs, mounts, devices) should be directly mapped into Go objects.
4. Image definitions don’t test distributions that are end-of-life. Respective code-paths should be dropped.
5. Image definitions need to support the oldest supported target distribution.

### Contributing

Please refer to the [developer guide](https://www.osbuild.org/docs/developer-guide/index) to learn about our workflow, code style and more.

See also the [local developer documentation](./docs/developer) for useful information about working with this specific project.

The build-requirements of the Go library for Fedora and rpm-based distributions are:

- `go`
- `gpgme-devel`

(see also [`Containerfile`](Containerfile) )

Other dependencies only needed in some cases are:

- `btrfs-progs-devel`, `device-mapper-devel`  
  build dependencies for the unit tests and projects that import `pkg/container`, which even in that case can be skipped using exclude_graphdriver_btrfs and exclude_graphdriver_devicemapper (see bootc-image-builder).
- `krb5-devel`  
  build dependency for the unit tests and projects that import `pkg/upload/koji`
- `osbuild-depsolve-dnf`  
  runtime dependency for the unit tests and projects that import `pkg/dnfjson`.
  or to run `cmd/gen-manifests` and `cmd/build`
- `osbuild` (and subpackages)  
  runtime dependencies for `cmd/build`.


### Repository:

 - **web**:   <https://github.com/osbuild/images>
 - **https**: `https://github.com/osbuild/images.git`
 - **ssh**:   `git@github.com:osbuild/images.git`

### Pull request gating

Each pull request against `images` starts a series of automated
tests. Tests run via GitHub Actions and GitLab CI. Each push to the pull request
will launch theses tests automatically.

### License:

 - **Apache-2.0**
 - See LICENSE file for details.
