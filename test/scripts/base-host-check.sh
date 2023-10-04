#!/usr/bin/env bash
set -euxo pipefail

systemctl is-system-running --wait
rpm -qa
cat /etc/os-release
uname -a
uptime
