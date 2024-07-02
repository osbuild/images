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
  is a `[]byte` array with custom unmarshalling/marshalling methods.
  - This stage requires the content specifications resolved from the manifest
    source specifications (package specs, container specs, ostree commit
    specs). See [Resolving Content](#resolving-content).

The `makeManifest()` function in `cmd/build/main.go` is a straightforward
implementation of the sequence of actions required to generate a manifest
described below.

## Manifest Instantiation

Instantiating a manifest involves generating an array of
[Pipelines][godoc-manifest-pipeline] that will produce an image. Each pipeline
supports different options that will affect the stages and stage options it
will generate.

Typically, manifest instantiation happens inside the
[`ImageType.Manifest()`][godoc-distro-imagetype] function, which each distro
implements separately. The function is responsible for:
- Validating blueprint customizations for the selected image type.
- Collecting static package sets for the distro and image type.
- Collecting container source specs from the blueprint customizations.
- Calling the image function for the image type, which creates an
  [`image.ImageKind`][godoc-image-imagekind] object, and
- Instantiating the manifest using `ImageKind.InstantiateManifest()`.

### The OS pipeline

The [OS][godoc-manifest-os] pipeline is the biggest and most central
pipeline. It is responsible for generating the stages that will create the main
OS tree of the resulting image. Almost all customizations that control specific
options for a bootable image are defined on this pipeline.

### The Build pipeline

The [Build][godoc-manifest-build] pipeline is used in every manifest to define
a build root for all the following pipelines to run in. It is always added
first and is almost never customized beyond package selection.

### Package Selection

The OS pipeline's private `getPackageSetChains()` method (which is called by
[`Manifest.GetPackageSetChains()`][godoc-manifest-manifest-getpackagesetchains])
is a good example of dynamic package selection based on enabled features and
customizations. Packages are selected based on features that will be required
on the running system. For example, if NTP servers are defined for the image,
the `chrony` package is selected.

The build pipeline is special in that extra packages are added to it
dynamically based on the requirements of subsequent pipelines. Pipelines can
define build packages that they require in the `getBuildPackages()` private
method. These packages will be merged with the static packages that are defined
for the build root. For example, if a container will be embedded in the OS
pipeline, the `skopeo` package is added to the build root to copy the container
into the OS container store.

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

### OSTree Commits

**Source specification**: The source specification for ostree commits is the
[`ostree.SourceSpec`][godoc-ostree-sourcespec]. It contains a URL to an ostree
repository and a ref to resolve.

**Content specification**: The content specification for ostree commits is the
[`ostree.CommitSpec`][godoc-ostree-commitspec]. Each ostree commit spec
contains the URL and ref from the source spec as well as the checksum of the
commit.

**Resolving**: Resolving **ostree source specs** to **ostree commit specs** is
done using the [`ostree.Resolve`][godoc-ostree-resolve] function. Resolving an
ostree source spec mainly involves resolving the content of the file at
`<URL>/refs/heads/<REF>`.

## Manifest Serialization

When a manifest is serialized by calling its
[`Manifest.Serialize()`][godoc-manifest-manifest-serialize], it runs the
private `serialize()` method on each pipeline in its array. Each pipeline in
turn creates an array of stages with the appropriate options based on the
customizations and options set on the pipeline.

The final JSON representation of the manifest that can be used with osbuild can
be created by using the standard library [`json.Marshal()`][godoc-json-marshal]
function.


----

[godoc-manifest-manifest]: https://pkg.go.dev/github.com/osbuild/images@main/pkg/manifest#Manifest
[godoc-manifest-pipeline]: https://pkg.go.dev/github.com/osbuild/images@main/pkg/manifest#Pipeline
[godoc-distro-imagetype]: https://pkg.go.dev/github.com/osbuild/images@main/pkg/distro/ImageType
[godoc-manifest-osbuildmanifest]: https://pkg.go.dev/github.com/osbuild/images@main/pkg/manifest#OSBuildManifest
[godoc-rpmmd-packageset]: https://pkg.go.dev/github.com/osbuild/images@main/pkg/rpmmd#PackageSet
[godoc-rpmmd-packagespec]: https://pkg.go.dev/github.com/osbuild/images@main/pkg/rpmmd#PackageSpec
[godoc-dnfjson-solver-depsolve]: https://pkg.go.dev/github.com/osbuild/images@main/internal/dnfjson#Solver.Depsolve
[godoc-container-sourcespec]: https://pkg.go.dev/github.com/osbuild/images@main/pkg/container#SourceSpec
[godoc-container-spec]: https://pkg.go.dev/github.com/osbuild/images@main/pkg/container#Spec
[godoc-container-resolver]: https://pkg.go.dev/github.com/osbuild/images@main/pkg/container#Resolver
[godoc-container-resolver-add]: https://pkg.go.dev/github.com/osbuild/images@main/pkg/container#Resolver.Add
[godoc-container-resolver-finish]: https://pkg.go.dev/github.com/osbuild/images@main/pkg/container#Resolver.Finish
[godoc-ostree-sourcespec]: https://pkg.go.dev/github.com/osbuild/images@main/pkg/ostree#SourceSpec
[godoc-ostree-commitspec]: https://pkg.go.dev/github.com/osbuild/images@main/pkg/ostree#CommitSpec
[godoc-ostree-resolve]: https://pkg.go.dev/github.com/osbuild/images@main/pkg/ostree#Resolve
[godoc-manifest-os]: https://pkg.go.dev/github.com/osbuild/images@main/pkg/manifest#OS
[godoc-manifest-build]: https://pkg.go.dev/github.com/osbuild/images@main/pkg/manifest#Build
[godoc-manifest-manifest-getpackagesetchains]: https://pkg.go.dev/github.com/osbuild/images@main/pkg/manifest#Manifest.GetPackageSetChains
[godoc-manifest-manifest-serialize]: https://pkg.go.dev/github.com/osbuild/images@main/pkg/manifest#Manifest.Serialize
[godoc-json-marshal]: https://pkg.go.dev/encoding/json#Marshal
[godoc-image-imagekind]: https://pkg.go.dev/github.com/osbuild/images@main/pkg/image#ImageKind
