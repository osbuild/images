#!/usr/bin/env bash
set -euo pipefail

echo "❓ Checking system status"
if ! sudo systemctl is-system-running --wait; then

    echo "❌ Listing units and exiting with failure"
    # system is not fully operational
    # (try to) list units so we can troubleshoot any failures
    systemctl list-units

    # exit with failure; we don't care about the exact exit code from the
    # failed condition
    exit 1
fi

echo "📦 Listing packages"
rpm -qa

echo "ℹ️ os-release"
cat /etc/os-release

echo "ℹ️ system information"
uname -a

echo "🕰️ uptime"
uptime
