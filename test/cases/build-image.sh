#!/usr/bin/env bash
#
# Build an image and store the hash of the manifest in S3 if successful.

set -euxo pipefail

distro="${1}"
imgtype="${2}"
config="${3}"

echo "Installing dependencies"
sudo dnf install -y go gpgme-devel gcc osbuild osbuild-luks2 osbuild-lvm2 osbuild-ostree osbuild-selinux

echo "Building image ${distro}/${imgtype} using config ${config}"
cat "${config}"  # print the config for logging
sudo go run ./cmd/build -output ./build -distro "${distro}" -image "${imgtype}" -config "${config}"

echo "Build finished!!"

echo "Registering successful build in S3"

config_name=$(jq -r .name "${config}")
manifest=(./build/*/manifest.json)  # there should only be one
manifest_hash=$(sha256sum "${manifest[0]}" | awk '{ print $1 }')

arch=$(uname -m)
osbuild_ver=$(osbuild --version)

# TODO: Include osbuild commit hash
filename="${manifest_hash}.json"
cat << EOF > "${filename}"
{
  "distro": "${distro}",
  "arch": "${arch}",
  "image-type": "${imgtype}",
  "config": "${config_name}",
  "manifest-checksum": "${manifest_hash}",
  "obuild-version": "${osbuild_ver}",
  "commit": "${CI_COMMIT_SHA}"
}
EOF

s3url="s3://image-builder-ci-artifacts/images/builds/${distro}/${arch}/${filename}"

source /etc/os-release
# s3cmd is in epel, add if it's not present
if [[ $ID == rhel || $ID == centos ]] && ! rpm -q epel-release; then
    curl -Ls --retry 5 --output /tmp/epel.rpm \
        https://dl.fedoraproject.org/pub/epel/epel-release-latest-"${VERSION_ID%.*}".noarch.rpm
    sudo rpm -Uvh /tmp/epel.rpm
fi
sudo dnf -y install s3cmd

echo "Uploading ${filename} to ${s3url}"
AWS_SECRET_ACCESS_KEY="$V2_AWS_SECRET_ACCESS_KEY" \
AWS_ACCESS_KEY_ID="$V2_AWS_ACCESS_KEY_ID" \
s3cmd --acl-private put "${filename}" "${s3url}"
