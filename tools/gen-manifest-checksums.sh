#!/usr/bin/bash
#
# Generates mock manifests (i.e. without real resolved content) for all test
# configurations and computes the checksum for each file. The checksumss are stored
# in test/data/manifest-checksums.txt and should be updated whenever a manifest
# changes. This makes it visible when a change affects a manifest without
# needing to store real manifests in the repository.

set -euo pipefail

tmpdir="$(mktemp -d)"
cleanup() {
    rm -r "${tmpdir}"
}
trap cleanup EXIT

export OSBUILD_TESTING_RNG_SEED=0
# NOTE: fedora-41 riscv has no test repositories so we need to skip it.
# NOTE: silence stdout as it gets way too noisy in the GitHub action log (until
# gen-manifests gets a verbosity or progress option).
go run ./cmd/gen-manifests \
    --packages=false --containers=false --commits=false \
    --metadata=false \
    --arches "x86_64,aarch64,ppc64le,s390x" \
    --output "${tmpdir}" \
    > /dev/null


# NOTE: 'osbuild --inspect' is generally a better way to calculate a manifest
# fingerprint, because it ignores things like pipeline names, source URLs, and
# generally things that don't affect the build output.
# For mocked manifests though we want those things to be visible changes, so we
# calculate the checksum of the file directly. Also it's faster.
checksums_file="./test/data/manifest-checksums.txt"
(cd "${tmpdir}" && sha1sum -- *) | sort > "${checksums_file}"

echo "Checksums saved to ${checksums_file}"
