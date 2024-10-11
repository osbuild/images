#!/bin/sh

set -eux

# Go version must be consistent with image-builder which uses UBI
# container that is typically few months behind
GO_VERSION=1.21.13
GO_BINARY=$(go env GOPATH)/bin/go$GO_VERSION

# this is the official way to get a different version of golang
# see https://go.dev/doc/manage-install
go install golang.org/dl/go$GO_VERSION@latest
$GO_BINARY download

# Ensure that go.mod and go.sum are up to date.
$GO_BINARY mod tidy

# Ensure the code is formatted correctly.
$GO_BINARY fmt ./...

./test/scripts/generate-gitlab-ci ./.gitlab-ci.yml
