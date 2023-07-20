#!/usr/bin/env bash
#
# Push a container to the CI registry

set -euo pipefail

archive="${1}"
name="${2}"


sudo dnf install -y podman
podman login -u "${CI_REGISTRY_USER}" -p "${CI_JOB_TOKEN}" "${CI_REGISTRY}"

# pull into local registry
container_name="${CI_REGISTRY}/${CI_PROJECT_PATH}/${name}"
image_id=$(podman pull "oci-archive:${archive}")
echo "Tagging ${image_id} -> ${container_name}"
podman tag "${image_id}" "${container_name}"
echo "Pushing ${container_name}"
podman push "${container_name}"
