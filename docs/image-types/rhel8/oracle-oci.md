# Oracle's oci image

This image type is meant to be used on Oracle Cloud Infrastructure. It is derived from the KVM guest image type.

## Format
The image format is `qcow2`. There are not special Oracle-specifics/metadata in it.

## Missing packages
- oracle guest agent - this image does not include the oracle guest agent, which is a collection of agents to collect, report, and allow
disk usage, os updates, metrics, and more, under the web console. It is mainly excluded because we don't have the source
code and we can't build and include it.

## Architecture
This image type is working with `x86_64` instances.
Oracle has ARM compute instances and this image type wasn't tested with it yet.
