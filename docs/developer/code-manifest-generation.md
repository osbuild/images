# Manifest generation

This document explains how manifests are generated in code. It is useful for
understanding how (and where) changes should be made in the pipeline generation
code to have the desired effect.

## Overview

Manifests are generated in two general stages: _Instantiation_ and _Serialization_.
- Instantiation: Creates an object that implements the
  [`manifest.Manifest`][godoc-manifest-manifest] interface.
  - Creating this requires a number of steps. See [Manifest
    Instantiation](#manifest-instantiation).
  - An instantiated `Manifest` contains:
    - **Source** specifications for content: package names, containers, ostree
      commits.
    - An array of [`manifest.Pipeline`][godoc-manifest-pipeline] objects, each
      with necessary customizations and serialization methods for producing
      stage-sequences.
- Serialization: Creates the sequence of stages based on each pipeline and
  produces [`manifest.OSBuildManifest`][godoc-manifest-osbuildmanifest], which
  is a `[]byte` array with custom un/marshalling methods.
  - This stage requires the content specifications resolved from the manifest
    source specifications (package specs, container specs, ostree commit
    specs). See [Resolving Content](#resolving-content).

## Manifest Instantiation

## Resolving Content

[`Manifest`][godoc-manifest-manifest] objects should provide source
specifications for the content they need. Each **source specification** should be
resolved to a **content specification** and passed to the serialization function to
create the final manifest

All source and content specifications have type `map[string][]<spec>` (where
`<spec>` is the base type for the source or content specification). The map key
is the name of the pipeline that requires the content and multiple specs can be
assigned to each pipeline.

Currently there are three methods for three types of content that needs to be
resolved.

### Package sets

**Source specification**: The source specification for packages is the
[`rpmmd.PackageSet`][godoc-rpmmd-packageset]. Each package set defines a list of
package names to include, a list to exclude, and a set of RPM repositories to
use to resolve and retrieve the packages.

_Note:_ The package source specification is special in that it defines an array
of package _sets_, which means each element in the array specifies multiple
packages. We sometimes refer to this array as a _package set chain_. Chains of
package sets are depsolved in order as part of the same call to `dnf-json`. The
result of a depsolve of each package set in the chain is merged with the
subsequent set and the result is a single array of package specs.

**Content specification**: The content specification for packages is the
[`rpmmd.PackageSpec`][godoc-rpmmd-packagespec]. Each package spec is a fully resolved
description of an RPM, with metadata, a checksum, and a URL from which to
retrieve the package.

**Resolving**: Resolving **package sets** to **package specs** is done using
the [`dnfjson.Solver.Depsolve()`][godoc-dnfjson-solver-depsolve] function. This
call resolves the dependencies of an array of package sets and returns all the
packages that were specified, their dependencies, and the metadata for each
package.

### Containers

**Source specification**: The source specification for containers is the
[`container.SourceSpec`][godoc-container-sourcespec]. The main component is the
`Source`, which is a full ref to a container in a registry.

**Content specification**: The content specification for containers is the
[`container.Spec`][godoc-container-spec]. Each container spec is a fully
resolved description of a container.

**Resolving**: Resolving **container source specs** to **container specs** is
done using the [`container.Resolver`][godoc-container-resolver] type. Container
source specs are added to the resolver with the
[`Resolver.Add()`][godoc-container-resolver-add] method and all results are
retrieved with the [`Resolver.Finish()`][godoc-container-resolver-finish]
method.


----

[godoc-manifest-manifest]: https://pkg.go.dev/github.com/osbuild/images@main/pkg/manifest#Manifest
[godoc-manifest-pipeline]: https://pkg.go.dev/github.com/osbuild/images@main/pkg/manifest#Pipeline
[godoc-manifest-osbuildmanifest]: https://pkg.go.dev/github.com/osbuild/images@main/pkg/manifest#OSBuildManifest
[godoc-rpmmd-packageset]: https://pkg.go.dev/github.com/osbuild/images@main/pkg/rpmmd#PackageSet
[godoc-rpmmd-packagespec]: https://pkg.go.dev/github.com/osbuild/images@main/pkg/rpmmd#PackageSpec
[godoc-dnfjson-solver-depsolve]: https://pkg.go.dev/github.com/osbuild/images@main/internal/dnfjson#Solver.Depsolve
[godoc-container-sourcespec]: https://pkg.go.dev/github.com/osbuild/images@main/pkg/container#SourceSpec
[godoc-container-spec]: https://pkg.go.dev/github.com/osbuild/images@main/pkg/container#Spec
[godoc-container-resolver]: https://pkg.go.dev/github.com/osbuild/images@main/pkg/container#Resolver
[godoc-container-resolver-add]: https://pkg.go.dev/github.com/osbuild/images@main/pkg/container#Resolver.Add
[godoc-container-resolver-finish]: https://pkg.go.dev/github.com/osbuild/images@main/pkg/container#Resolver.Finish
