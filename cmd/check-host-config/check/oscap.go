package check

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/osbuild/images/cmd/check-host-config/cos"
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

func (o OpenSCAPCheck) Run(ctx context.Context, log Logger, config *buildconfig.BuildConfig) error {
	oscap := config.Blueprint.Customizations.OpenSCAP
	if oscap == nil {
		return Skip("no OpenSCAP customization")
	}

	baselineScore := 0.8
	profile := oscap.ProfileID
	datastream := oscap.DataStream

	if profile == "" || datastream == "" {
		return Skip("incomplete OpenSCAP configuration")
	}

	log.Println("Running oscap scanner")
	profileName := profile + "_osbuild_tailoring"

	// Run oscap evaluation
	// NOTE: sudo works here without password because we test this only on ami
	// initialised with cloud-init, which sets sudo NOPASSWD for the user
	// NOTE: oscap returns exit code 2 for any failed rules, so we ignore the error
	out, err := cos.ExecContext(ctx, log, "sudo", "oscap", "xccdf", "eval",
		"--results", "results.xml",
		"--profile", profileName,
		"--tailoring-file", "/oscap_data/tailoring.xml",
		datastream)

	// oscap may return non-zero exit code even on success (exit code 2 for failed rules)
	// so we check if results.xml was created instead
	if !cos.ExistsContext(ctx, log, "results.xml") {
		errMsg := ""
		if err != nil {
			errMsg = err.Error()
		}
		return Fail("oscap evaluation failed:", string(out), "error:", errMsg)
	}

	log.Println("Saving results")
	// Change ownership of results.xml
	_, err = cos.ExecContext(ctx, log, "sudo", "chown", fmt.Sprintf("%d", os.Getuid()), "results.xml")
	if err != nil {
		log.Printf("Warning: failed to chown results.xml: %v\n", err)
	}

	// Read and parse results.xml using xmlstarlet (matching shell script approach)
	log.Println("Checking oscap score")

	// Extract score using xmlstarlet
	out, err = cos.ExecContext(ctx, log, "xmlstarlet", "sel", "-N", "x=http://checklists.nist.gov/xccdf/1.2", "-t", "-v", "//x:score", "results.xml")
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

	// Check for failed high severity rules
	log.Println("Checking for failed rules")
	out, err = cos.ExecContext(ctx, log, "xmlstarlet", "sel", "-N", "x=http://checklists.nist.gov/xccdf/1.2", "-t", "-v", "//x:rule-result[@severity='high']", "results.xml")
	if err != nil {
		return Fail("failed to extract rule results from results.xml:", err.Error())
	}

	highSeverityOutput := string(out)
	// Count occurrences of "fail" in the output
	severityCount := strings.Count(highSeverityOutput, "fail")
	log.Printf("Severity count: %d\n", severityCount)

	// Extract failed rules for logging
	var failedRules []string
	if severityCount > 0 {
		lines := strings.Split(highSeverityOutput, "\n")
		for _, line := range lines {
			if strings.Contains(line, "fail") {
				failedRules = append(failedRules, line)
			}
		}
	}

	log.Println("Checking for test result")
	log.Printf("Baseline score: %.2f%%\n", baselineScore*100)
	log.Printf("Hardened score: %.2f%%\n", hardenedScore*100)

	// Compare scores
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
