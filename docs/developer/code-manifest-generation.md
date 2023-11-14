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


[godoc-manifest-manifest]: https://pkg.go.dev/github.com/osbuild/images@main/pkg/manifest#Manifest
[godoc-manifest-pipeline]: https://pkg.go.dev/github.com/osbuild/images@main/pkg/manifest#Pipeline
[godoc-manifest-osbuildmanifest]: https://pkg.go.dev/github.com/osbuild/images@main/pkg/manifest#OSBuildManifest
