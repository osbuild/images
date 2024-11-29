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
    serial=$(jq -r '.blueprint.customizations.cacerts.pem_certs[0]' "${config}" | openssl x509 -noout -serial | cut -d= -f 2-)

    echo "📗 Checking CA cert anchor file"
    if ! [ -e "/etc/pki/ca-trust/source/anchors/${serial}.pem" ]; then
        echo "Anchor CA file does not exist"
        exit 1
    fi

    echo "📗 Checking extracted CA cert file"
    if ! [ -e "/etc/pki/ca-trust/source/extracted/pem/directory-hash/Test_CA_for_osbuild.pem.pem" ]; then
        echo "Extracted CA file does not exist"
        exit 1
    fi
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
fi
