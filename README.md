Images
======

### Project

 * **Website**: <https://www.osbuild.org>
 * **Bug Tracker**: <https://github.com/osbuild/images/issues>
 * **IRC**: #osbuild on [Libera.Chat](https://libera.chat/)
 * **Changelog**: <https://github.com/osbuild/images/releases>

#### Contributing

Please refer to the [developer guide](https://www.osbuild.org/guides/developer-guide/developer-guide.html) to learn about our workflow, code style and more.

The build-requirements for Fedora and rpm-based distributions are:
- `gpgme-devel`, `btrfs-progs-devel`, `device-mapper-devel`

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
