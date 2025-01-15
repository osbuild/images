# Image definituons
This directory contains definitions of image types that this library can build.

> Currently, only packages are defined in YAML. The rest is still in Go.

The supporting library for these files is in ../pkg/definition.

## Definition searching
When a user wants to build an ami for rhel 10.2 on x86_64, the library searches for the most suitable definition:

- firstly, it tries to open `rhel/10.2/x86_64/ami.yaml`
- if it doesn't exist, it tries to open `rhel/10.2/generic/ami.yaml`
- if it doesn't exist, it tries to open `rhel/X.Y/x86_64/ami.yaml` or `rhel/X.Y/generic/ami.yaml`, with X.Y being the closest older version to 10.2 (10.1 will be prefered over 10.0)
- if it doesn't exist, the build fails

## Definition format
There are two top-level fields: `from` and `def`.

### `from`
An array of relative paths to include. This works recursively, so an included file will also process its includes. The merging rules are explained in the invidual fields under `def`.

The includes are processed using [DFS postordering](https://en.wikipedia.org/wiki/Depth-first_search#Vertex_orderings).

### `def`
`def` is the core of the definition file. It contains following keys:

#### `packages`
Map of package sets installed in the image type. The keys are currently defined in Go code. The values are structs, each one represents inputs for one DNF transaction.

When merging package sets, the list of includes and excludes are simply appended together.

The struct has following keys:

##### `include`
List of packages that must be in the depsolved transaction.

##### `exclude`
List of packages that must not be in the depsolved transaction.
