#!/usr/bin/env bash
# vim: sw=4:et
set -euo pipefail

running_wait() {
    # simple implementation of 'systemctl is-system-running --wait' for older
    # versions of systemd that don't support the option (EL8)
    #
    # From SYSTEMCTL(1)
    #   If --wait is in use, states initializing or starting will not be
    #   reported, instead the command will block until a later state (such as
    #   running or degraded) is reached.
    while true; do
        state=$(systemctl is-system-running)
        echo "${state}"

        # keep iterating on initializing and starting
        case "${state}" in
        "initializing" | "starting")
            sleep 3
            continue
            ;;

        # the only good state
        "running")
            return 0
            ;;

        # fail on anything else
        *)
            return 1
        esac
    done
}

get_oscap_score() {
    config_file="$1"
    baseline_score=0.8
    echo "🔒 Running oscap scanner"
    # NOTE: sudo works here without password because we test this only on ami
    # initialised with cloud-init, which sets sudo NOPASSWD for the user
    profile=$(jq -r .blueprint.customizations.openscap.profile_id "${config_file}")
    datastream=$(jq -r .blueprint.customizations.openscap.datastream "${config_file}")
    sudo oscap xccdf eval \
        --results results.xml \
        --profile "${profile}_osbuild_tailoring" \
        --tailoring-file "/oscap_data/tailoring.xml" \
        "${datastream}" || true # oscap returns exit code 2 for any failed rules

    echo "📄 Saving results"
    sudo chown "$UID" results.xml

    echo "📗 Checking oscap score"
    hardened_score=$(xmlstarlet sel -N x="http://checklists.nist.gov/xccdf/1.2" -t -v "//x:score" results.xml)
    echo "Hardened score: ${hardened_score}%"

    echo "📗 Checking for failed rules"
    high_severity=$(xmlstarlet sel -N x="http://checklists.nist.gov/xccdf/1.2" -t -v "//x:rule-result[@severity='high']" results.xml)
    severity_count=$(echo "${high_severity}" | grep -c "fail" || true)
    echo "Severity count: ${severity_count}"

    echo "🎏 Checking for test result"
    echo "Baseline score: ${baseline_score}%"
    echo "Hardened score: ${hardened_score}%"

    # compare floating point numbers
    if (( hardened_score < baseline_score )); then
        echo "❌ Failed"
        echo "Hardened image score (${hardened_score}) did not improve baseline score (${baseline_score})"
        exit 1
    fi

    if (( severity_count > 0 )); then
        echo "❌ Failed"
        echo "One or more oscap rules with high severity failed"
        # add a line to print the failed rules
        echo "${high_severity}" | grep -B 5 "fail"
        exit 1
    fi
}

check_ca_cert() {
    serial=$(jq -r '.blueprint.customizations.cacerts.pem_certs[0]' "${config}" | openssl x509 -noout -serial | cut -d= -f 2- | tr '[:upper:]' '[:lower:]')
    cn=$(jq -r '.blueprint.customizations.cacerts.pem_certs[0]' "${config}" | openssl x509 -noout -subject | sed -E 's/.*CN ?= ?//')

    echo "📗 Checking CA cert anchor file serial '${serial}'"
    if ! [ -e "/etc/pki/ca-trust/source/anchors/${serial}.pem" ]; then
        echo "Anchor CA file does not exist, directory contents:"
        find /etc/pki/ca-trust/source/anchors
        exit 1
    fi

    echo "📗 Checking extracted CA cert file named '${cn}'"
    if ! grep -q "${cn}" /etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem; then
        echo "Extracted CA cert not found in the bundle, tls-ca-bundle.pem contents:"
        grep '^#' /etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem
        exit 1
    fi
}

check_modularity() {
    # Verify modules that are enabled on a system, if any.
    modules_expected=$(jq -rc '.expect_modularity.modules[]' "${config}")
    modules_enabled=$(dnf module list --enabled 2>&1 | tail -n+4 | head -n -2)

    # Go over the expected modules and check if each of them is installed
    echo "$modules_expected" | while read module_expected; do
        name=""
        version=""
    done
}

echo "❓ Checking system status"
if ! running_wait; then

    echo "❌ Listing units"
    # system is not fully operational
    # (try to) list units so we can troubleshoot any failures
    systemctl list-units

    echo "❌ Status for all failed units"
    # the default 10 lines might be a bit too short for troubleshooting some
    # units, 100 should be more than enough
    systemctl status --failed --full --lines=100

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

# NOTE: we should do a lot more here
if (( $# > 0 )); then
    config="$1"
    if jq -e .blueprint.customizations.openscap "${config}"; then
        get_oscap_score "${config}"
    fi

    if jq -e '.blueprint.customizations.cacerts.pem_certs[0]' "${config}"; then
        check_ca_cert "${config}"
    fi

    if jq -e '.expect_modularity' "${config}"; then
        check_modularity "${config}"
    fi
fi
