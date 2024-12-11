# Image specs
This directory contains specifications of image types that this library can build.

Currently, only packages are defined here.

The supporting library for these files is in ../pkg/specs.

## Definition searching
When user wants to build an ami for rhel 10.2 on x86_64, the library searches for the most suitable definition:

- firstly, it tries to open `rhel-10.2/x86_64/ami.yaml`
- if it doesn't exist, it tries to open `rhel-10.2/generic/ami.yaml`
- if it doesn't exist, it tries to open `rhel-X.Y/x86_64/ami.yaml` or `rhel-X.Y/generic/ami.yaml`, with X.Y being the closest older version to 10.2 (10.1 will be prefered over 10.0)
- if it doesn't exist, the build fails

## Definition format

There are two top-level fields: `includes` and `spec`.

### `includes`
An array of relative paths to include. This works recursively, so an included file will also process its includes.

Currently, the merging rules are simple:
- included lists are being appened to
- other value type are just being overriden

The includes are processed using [DFS postordering](https://en.wikipedia.org/wiki/Depth-first_search#Vertex_orderings). At least I think.

Example:
```console
$ cat base.yaml
spec:
  key: new
  array:
    - a
$ cat derived.yaml
includes:
  - base.yaml
spec:
  key: new
  array:
    - b
$ go run ./cmd/spec-flatten ./derived.yaml
spec:
    array:
        - a
        - b
    key: new
```
### `specs`
`specs` is the meat of the definition file. It's a struct with two keys: `packages` and `exclude_packages`.

#### `packages`
`packages` is a list of packages to include in the os pipeline.

#### `exclude_packages`
`exclude_packages` is a list of packages to exclude from the os pipeline.

