#!/bin/bash
set -euo pipefail

# Get OS data.
source /etc/os-release

# Provision the software under test.
/usr/libexec/osbuild-composer-test/provision.sh

# Test the images
/usr/libexec/osbuild-composer-test/libvirt_test.sh qcow2
/usr/libexec/osbuild-composer-test/libvirt_test.sh openstack
/usr/libexec/osbuild-composer-test/libvirt_test.sh vhd

# RHEL 8.4 and Centos Stream 8 images also supports uefi, check that
if [[ "${ID}-${VERSION_ID}" == "rhel-8.4" || "${ID}-${VERSION_ID}" == "centos-8" ]]; then
  echo "🐄 Booting qcow2 image in UEFI mode on RHEL/Centos Stream"
  /usr/libexec/osbuild-composer-test/libvirt_test.sh qcow2 uefi
fi
