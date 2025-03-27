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

# Check if the containers specified in the blueprint are embedded in the image
# by checking if the container source is present in the podman images list
check_container_embedding() {
    local config_file="$1"
    if [[ -z "${config_file}" ]]; then
        echo "❌ check_container_embedding(): no config file provided"
        exit 1
    fi

    echo "📗 Checking embedded containers"

    local error=0
    for container in $(jq -rc '.blueprint.containers[]?' "${config_file}") ; do
        local bp_container_source
        bp_container_source=$(echo "${container}" | jq -r '.source')
        if [[ "${bp_container_source}" == "null" ]]; then
            echo "❌ Container source not found: ${container}"
            error=1
            continue
        fi

        local podman_containers
        podman_containers=$(sudo podman images --format json | jq -rc "[.[] | select(any(.Names[]; startswith(\"${bp_container_source}\")))]")
        local podman_containers_count
        podman_containers_count=$(echo "${podman_containers}" | jq -r 'length')
        if [[ "${podman_containers_count}" -ne 1 ]]; then
            echo "❌ Unexpected number of containers found: ${podman_containers_count}, expected 1"
            echo "📄 Podman containers:"
            echo "${podman_containers}"
            error=1
            continue
        fi
    done

    if (( error > 0 )); then
        echo "❌ Container embedding check failed"
        exit 1
    fi
}

# Check that the rootless and rootfull podman would use the same network backend.
# This is especially important in cases when we embed containers in the image,
# because some versions of podman would default to using 'cni' network backend
# if there are existing container images in the image. We embed the container
# as root, so the rootfull podman would use 'cni' network backend, while the
# rootless podman would use 'netavark' network backend in such case.
# We do not want this inconsistency, so we check for it.
check_podman_network_backend_consistency() {
    echo "📗 Checking podman network backend consistency for rootfull and rootless podman"

    local rootfull_network_backend
    rootfull_network_backend=$(sudo podman info --format json | jq -r '.host.networkBackend // "undefined"')
    echo "ℹ️ Rootfull podman network backend: ${rootfull_network_backend}"
    local rootless_network_backend
    rootless_network_backend=$(podman info --format json | jq -r '.host.networkBackend // "undefined"')
    echo "ℹ️ Rootless podman network backend: ${rootless_network_backend}"
    if [[ "${rootfull_network_backend}" != "${rootless_network_backend}" ]]; then
        echo "❌ Podman network backends are inconsistent"
        echo "Rootfull podman network backend: ${rootfull_network_backend}"
        echo "Rootless podman network backend: ${rootless_network_backend}"
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

    if ! type -p jq &>/dev/null; then
        echo "❌ ERROR: jq not found, which is required for the tests"
        exit 1
    fi

    if jq -e .blueprint.customizations.openscap "${config}"; then
        get_oscap_score "${config}"
    fi

    if jq -e '.blueprint.customizations.cacerts.pem_certs[0]' "${config}"; then
        check_ca_cert "${config}"
    fi

    if jq -e '.blueprint.enabled_modules' "${config}" || jq -e '.blueprint.packages[] | select(.name | startswith("@") and contains(":")) | .name' "${config}"; then
        check_modularity "${config}"
    fi

    if jq -e '.blueprint.containers' "${config}"; then
        check_container_embedding "${config}"
        check_podman_network_backend_consistency
    fi
fi
