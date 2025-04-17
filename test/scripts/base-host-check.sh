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
    echo "📗 Checking enabled modules"

    # Verify modules that are enabled on a system, if any. Modules can either be enabled separately
    # or they can be installed through packages directly. We test both cases here.
    #
    # Caveat is that when a module is enabled yet _no_ packages are installed from it this breaks.
    # Let's not do that in the test?

    modules_expected_0=$(jq -rc '.blueprint.enabled_modules[]? | .name + ":" + .stream' "${config}")
    modules_expected_1=$(jq -rc '.blueprint.packages[]? | select(.name | startswith("@") and contains(":")) | .name' "${config}" | cut -c2-)

    modules_expected="${modules_expected_0}\n${modules_expected_1}"
    modules_enabled=$(dnf module list --enabled 2>&1 | tail -n+4 | head -n -2 | tr -s ' ' | cut -d' ' -f1,2 | tr ' ' ':')

    # Go over the expected modules and check if each of them is installed
    echo "$modules_expected" | while read -r module_expected; do
        echo "📗 Module expected: ${module_expected}"
        if [[ $module_expected != *$modules_enabled* ]]; then
            echo "❌ Module was not enabled: ${module_expected}"
            exit 1
        else
            echo "Module was enabled"
        fi
    done
}

check_hostname() {
    echo "📗 Checking hostname"

    # Verify that the hostname is set by running `hostname`
    expected_hostname=$(jq -r '.blueprint.customizations.hostname' "${config}")
    actual_hostname=$(hostname)

    if [[ $actual_hostname != "${expected_hostname}" ]]; then 
        echo "❌ Hostname was not set: hostname=${actual_hostname} expected=${expected_hostname}"
        exit 1
    else
        echo "Hostname was set"
    fi
}

check_services_enabled() {
    echo "📗 Checking enabled services"

    services_expected=$(jq -rc '.blueprint.customizations.services.enabled[]' "${config}")

    echo "$services_expected" | while read -r service_expected; do
        state=$(systemctl is-enabled "${service_expected}")
        if [[ "${state}" == "enabled" ]]; then
            echo "Service was enabled service=${service_expected} state=${state}"
        else
            echo "❌ Service was not enabled service=${service_expected} state=${state}"
            exit 1
        fi
    done
}

check_services_disabled() {
    echo "📗 Checking disabled services"

    services_expected=$(jq -rc '.blueprint.customizations.services.disabled[]' "${config}")

    echo "$services_expected" | while read -r service_expected; do
        state=$(systemctl is-enabled "${service_expected}" || true)
        if [[ "${state}" == "disabled" ]]; then
            echo "Service was disabled service=${service_expected} state=${state}"
        else
            echo "❌ Service was not disabled service=${service_expected} state=${state}"
            exit 1
        fi
    done
}

check_firewall_services_enabled() {
    echo "📗 Checking enabled firewall services"

    services_expected=$(jq -rc '.blueprint.customizations.firewall.services.enabled[]' "${config}")

    echo "$services_expected" | while read -r service_expected; do
        # NOTE: sudo works here without password because we test this only on ami
        # initialised with cloud-init, which sets sudo NOPASSWD for the user
        state=$(sudo firewall-cmd --query-service="${service_expected}")
        if [[ "${state}" == "yes" ]]; then
            echo "Firewall service was enabled service=${service_expected} state=${state}"
        else
            echo "❌ Firewall service was not enabled service=${service_expected} state=${state}"
            exit 1
        fi
    done
}

check_firewall_services_disabled() {
    echo "📗 Checking disabled firewall services"

    services_expected=$(jq -rc '.blueprint.customizations.firewall.services.disabled[]' "${config}")

    echo "$services_expected" | while read -r service_expected; do
        # NOTE: sudo works here without password because we test this only on ami
        # initialised with cloud-init, which sets sudo NOPASSWD for the user
        state=$(sudo firewall-cmd --query-service="${service_expected}")
        if [[ "${state}" == "no" ]]; then
            echo "Firewall service was disabled service=${service_expected} state=${state}"
        else
            echo "❌ Firewall service was not disabled service=${service_expected} state=${state}"
            exit 1
        fi
    done
}

check_users() {
    echo "📗 Checking users"

    users_expected=$(jq -rc '.blueprint.customizations.user[]' "$config")

    echo "$users_expected" | while read -r user_expected; do
        username=$(echo "$user_expected" | jq -rc .name)
        if ! id "$username"; then
            echo "❌ User did not exist: username=${username}"
            exit 1
        else
            echo "User ${username} exists"
        fi
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

    if jq -e '.blueprint.enabled_modules' "${config}" || jq -e '.blueprint.packages[] | select(.name | startswith("@") and contains(":")) | .name' "${config}"; then
        check_modularity "${config}"
    fi

    if jq -e '.blueprint.customizations.user' "${config}"; then
        check_users "${config}"
    fi

    if jq -e '.blueprint.customizations.services.enabled' "${config}"; then
        check_services_enabled "${config}"
    fi

    if jq -e '.blueprint.customizations.services.disabled' "${config}"; then
        check_services_disabled "${config}"
    fi

    if jq -e '.blueprint.customizations.firewall.services.enabled' "${config}"; then
        check_firewall_services_enabled "${config}"
    fi

    if jq -e '.blueprint.customizations.firewall.services.disabled' "${config}"; then
        check_firewall_services_disabled "${config}"
    fi

    if jq -e '.blueprint.customizations.hostname' "${config}"; then
        check_hostname "${config}"
    fi
fi
