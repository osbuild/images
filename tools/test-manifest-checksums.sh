#!/usr/bin/bash
#
# Runs gen-manifests N times and checks that each run produces identical
# manifest JSON files. Compares consecutive runs as soon as the later run
# finishes and exits on the first mismatch.

set -euo pipefail

N="${1:-10}"
BASE="/tmp/test-manifest-checksums"
BIN="/tmp/test-manifest-checksums-bin"

export OSBUILD_TESTING_RNG_SEED=0
export IMAGE_BUILDER_EXPERIMENTAL=gen-manifest-mock-bpfile-uris

cd "$(git rev-parse --show-toplevel)"

echo "Building gen-manifests"
go build -v -o "${BIN}" ./cmd/gen-manifests

rm -rf "${BASE}"
mkdir -p "${BASE}"

gen_flags=(
    --packages=false --containers=false --commits=false --flatpaks=false
    --metadata=false
    --fake-bootc=true
    --arches "x86_64,aarch64,ppc64le,s390x"
)

for i in $(seq 1 "${N}"); do
    out="${BASE}/${i}"
    mkdir -p "${out}"
    stderr="$(mktemp)"
    start=$(date +%s.%N)
    if ! "${BIN}" "${gen_flags[@]}" --output "${out}" > /dev/null 2> "${stderr}"; then
        cat "${stderr}"
        rm -f "${stderr}"
        exit 1
    fi
    rm -f "${stderr}"
    elapsed=$(awk -v s="${start}" -v e="$(date +%s.%N)" 'BEGIN { printf "%.2f", e - s }')
    printf 'Run %s/%s -> %s (%ss)\n' "${i}" "${N}" "${out}" "${elapsed}"

    if (( i >= 2 )); then
        prev="${BASE}/$((i - 1))"
        echo "Comparing run $((i - 1)) with run ${i}"
        if ! diff -qr "${prev}" "${out}" > /dev/null; then
            echo "Manifest output differs between run $((i - 1)) and run ${i}" >&2
            diff -ru "${prev}" "${out}" >&2 || true
            echo "Left outputs in ${BASE}" >&2
            exit 1
        fi
        rm -rf "${prev}"
    fi
done

echo "All ${N} runs produced identical manifests."
rm -rf "${BASE}"
