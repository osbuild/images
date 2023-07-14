# osbuild/images testing information

`./test/configs/` contains configuration files for building images for testing. The files are used by the following tools:

- `./cmd/build` takes a config file as argument to build an image.  For example:
```
sudo go run ./cmd/build -output ./buildtest -rpmmd /tmp/rpmmd -distro fedora-38 -image qcow2 -config test/configs/embed-containers.json
```
will build a Fedora 38 qcow2 image using the configuration specified in the file `embed-containers.json`

- `./cmd/gen-manifests` generates manifests based on the configs specified in `./test/config-map.json`. The config map maps configuration files to image types and also sets a default configuration for any image type that's not specified.

The config map is also used in CI to dynamically generate test builds using the `./test/cases/generate-build-config` scripts.
