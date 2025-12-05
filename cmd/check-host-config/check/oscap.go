package check

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/osbuild/images/cmd/check-host-config/mockos"
	"github.com/osbuild/images/internal/buildconfig"
)

type OpenSCAPCheck struct{}

func (o OpenSCAPCheck) Metadata() Metadata {
	return Metadata{
		Name:                   "OpenSCAP Check",
		ShortName:              "oscap",
		Timeout:                5 * time.Minute,
		RequiresBlueprint:      true,
		RequiresCustomizations: true,
	}
}

// GetDatastreamFilename returns the full OpenSCAP datastream path based on OSRelease.
// Returns the full path (e.g., "/usr/share/xml/scap/ssg/content/ssg-rhel9-ds.xml") or an error if the OS/version combination is not supported.
func GetDatastreamFilename(release *OSRelease) (string, error) {
	// Map of OS ID and version to datastream filenames
	datastreamMap := map[string]string{
		"rhel:8":    "ssg-rhel8-ds.xml",
		"rhel:9":    "ssg-rhel9-ds.xml",
		"rhel:10":   "ssg-rhel10-ds.xml",
		"centos:8":  "ssg-centos8-ds.xml",
		"centos:9":  "ssg-cs9-ds.xml",
		"centos:10": "ssg-cs10-ds.xml",
		"fedora":    "ssg-fedora-ds.xml",
	}

	// Extract major version from VersionID (e.g., "9.0" -> "9")
	majorVersion := release.VersionID
	if idx := strings.Index(majorVersion, "."); idx != -1 {
		majorVersion = majorVersion[:idx]
	}

	// Build lookup key
	var key string
	switch release.ID {
	case "rhel", "centos":
		if majorVersion == "" {
			return "", fmt.Errorf("unsupported OS version: %s %s", release.ID, release.VersionID)
		}
		key = fmt.Sprintf("%s:%s", release.ID, majorVersion)
	case "fedora":
		key = "fedora"
	default:
		return "", fmt.Errorf("unsupported OS ID: %s", release.ID)
	}

	filename, ok := datastreamMap[key]
	if !ok {
		return "", fmt.Errorf("no datastream found for %s version %s", release.ID, majorVersion)
	}

	return "/usr/share/xml/scap/ssg/content/" + filename, nil
}

func (o OpenSCAPCheck) Run(ctx context.Context, log Logger, config *buildconfig.BuildConfig) error {
	oscap := config.Blueprint.Customizations.OpenSCAP
	if oscap == nil {
		return Skip("no OpenSCAP customization")
	}

	baselineScore := 0.8
	profile := oscap.ProfileID
	datastream := oscap.DataStream

	if profile == "" {
		return Skip("incomplete OpenSCAP configuration")
	}

	// Handle null/empty datastream by finding default datastream
	// See pkg/customizations/oscap/oscap.go:datastream fallbacks
	if datastream == "" || datastream == "null" {
		osRelease, err := ParseOSRelease(ctx, log, "/etc/os-release")
		if err != nil {
			return Fail("failed to read OS ID from /etc/os-release:", err.Error())
		}

		datastream, err = GetDatastreamFilename(osRelease)
		if err != nil {
			return Fail("failed to determine datastream filename:", err.Error())
		}

		log.Printf("Using default datastream: %s\n", datastream)
	}

	profileName := profile + "_osbuild_tailoring"

	// Run oscap evaluation
	// NOTE: sudo works here without password because we test this only on ami
	// initialised with cloud-init, which sets sudo NOPASSWD for the user
	// NOTE: oscap returns exit code 2 for any failed rules, so we ignore the error
	out, _, err := mockos.ExecContext(ctx, log, "sudo", "oscap", "xccdf", "eval",
		"--results", "results.xml",
		"--profile", profileName,
		"--tailoring-file", "/oscap_data/tailoring.xml",
		datastream)

	// oscap may return non-zero exit code even on success (exit code 2 for failed rules)
	// so we check if results.xml was created instead
	if !mockos.ExistsContext(ctx, log, "results.xml") {
		errMsg := ""
		if err != nil {
			errMsg = err.Error()
		}
		return Fail("oscap evaluation failed:", string(out), "error:", errMsg)
	}

	_, _, err = mockos.ExecContext(ctx, log, "sudo", "chown", fmt.Sprintf("%d", os.Getuid()), "results.xml")
	if err != nil {
		log.Printf("Warning: failed to chown results.xml: %v\n", err)
	}

	out, _, err = mockos.ExecContext(ctx, log, "xmlstarlet", "sel", "-N", "x=http://checklists.nist.gov/xccdf/1.2", "-t", "-v", "//x:score", "results.xml")
	if err != nil {
		return Fail("failed to extract score from results.xml:", err.Error())
	}

	scoreStr := strings.TrimSpace(string(out))
	hardenedScore, err := strconv.ParseFloat(scoreStr, 64)
	if err != nil {
		return Fail("failed to parse oscap score:", scoreStr, "error:", err.Error())
	}
	hardenedScore = hardenedScore / 100.0 // Convert percentage to decimal

	log.Printf("Hardened score: %.2f%%\n", hardenedScore*100)

	out, _, err = mockos.ExecContext(ctx, log, "xmlstarlet", "sel", "-N", "x=http://checklists.nist.gov/xccdf/1.2", "-t", "-v", "//x:rule-result[@severity='high']", "results.xml")
	if err != nil {
		return Fail("failed to extract rule results from results.xml:", err.Error())
	}

	highSeverityOutput := string(out)
	severityCount := strings.Count(highSeverityOutput, "fail")
	log.Printf("Severity count: %d\n", severityCount)

	var failedRules []string
	if severityCount > 0 {
		lines := strings.Split(highSeverityOutput, "\n")
		for _, line := range lines {
			if strings.Contains(line, "fail") {
				failedRules = append(failedRules, line)
			}
		}
	}

	log.Printf("Baseline score: %.2f%%\n", baselineScore*100)
	log.Printf("Hardened score: %.2f%%\n", hardenedScore*100)

	if hardenedScore < baselineScore {
		return Fail("hardened image score (", fmt.Sprintf("%.2f", hardenedScore*100),
			"%) did not improve baseline score (", fmt.Sprintf("%.2f", baselineScore*100), "%)")
	}

	if severityCount > 0 {
		log.Println("Failed high severity rules:")
		for _, rule := range failedRules {
			log.Printf("  %s\n", rule)
		}
		return Fail("one or more oscap rules with high severity failed")
	}

	return Pass()
}
