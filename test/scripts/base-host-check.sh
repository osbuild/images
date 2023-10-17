#!/usr/bin/env bash
set -euxo pipefail

if ! sudo systemctl is-system-running --wait; then
    # system is not fully operational
    # (try to) list units so we can troubleshoot any failures
    systemctl list-units

    # exit with failure; we don't care about the exact exit code from the
    # failed condition
    exit 1
fi

rpm -qa
cat /etc/os-release
uname -a
uptime
