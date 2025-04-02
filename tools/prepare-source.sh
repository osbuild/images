#!/bin/sh

set -eux

# Figure out the latest and greatest Go version
GO_LATEST=$(curl -s https://endoflife.date/api/go.json | jq -r '.[0].latest')

# Go version must be consistent with image-builder which uses UBI
# container that is typically few months behind
GO_VERSION=1.22.9

# Pin Go and toolchain versions at a reasonable versions
go get "go@$GO_VERSION" "toolchain@$GO_LATEST"

# Ensure the code is formatted correctly.
go fmt ./...

# Update go.mod and go.sum
go mod tidy

# Generate CI
./test/scripts/generate-gitlab-ci ./.gitlab-ci.yml

