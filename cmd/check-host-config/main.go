package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/osbuild/images/cmd/check-host-config/check"
	"github.com/osbuild/images/internal/buildconfig"
)

// waitForSystem waits until the system is reported by systemd as "running" or the timeout is reached.
func waitForSystem(timeout time.Duration) error {
	if timeout <= 0 {
		return nil
	}

	if err := runningWait(timeout, 15*time.Second); err != nil {
		fmt.Fprintf(os.Stderr, "Error while waiting for system to be running: %v\n", err)
		if strings.Contains(err.Error(), "timeout") {
			if activatingUnits := listBadUnits(); activatingUnits != "" {
				fmt.Fprintf(os.Stderr, "Units still activating: %s\n", activatingUnits)
			}
		}
		return err
	}
	return nil
}

// runChecks runs all checks sequentially and processes their results.
func runChecks(checks []check.RegisteredCheck, config *buildconfig.BuildConfig, quiet bool) bool {
	defer log.SetPrefix("")
	if quiet {
		log.SetOutput(io.Discard)
		defer log.SetOutput(os.Stdout)
	}

	var results check.SortedResults
	for _, chk := range checks {
		var err error
		meta := chk.Meta
		log.SetPrefix(meta.ShortName + ": ")

		switch {
		case meta.TempDisabled != "":
			err = check.Skip("temporarily disabled: " + meta.TempDisabled)
		case meta.RequiresBlueprint && (config == nil || config.Blueprint == nil):
			err = check.Skip("no blueprint")
		case meta.RequiresCustomizations && (config == nil || config.Blueprint == nil || config.Blueprint.Customizations == nil):
			err = check.Skip("no customizations")
		default:
			err = chk.Func(meta, config)
		}

		results = append(results, check.Result{Meta: meta, Error: err})

		if err != nil {
			log.Println(err)
		}
	}

	log.SetOutput(os.Stdout)
	sort.Sort(results)
	var seenError bool
	for _, res := range results {
		err := res.Error
		icon := check.IconFor(err)

		switch {
		case err == nil:
			fmt.Printf("%s %s: passed\n", icon, res.Meta.Name)
		default:
			if !check.IsSkip(err) && !check.IsWarning(err) {
				seenError = true
			}
			fmt.Printf("%s %s: %s\n", icon, res.Meta.Name, err)
		}
	}

	return !seenError
}

// Return codes:
//
//	0: All checks passed
//	1: One or more checks failed
//	2: Wait timeout
func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(0)

	waitTimeout := flag.Duration("wait-timeout", 15*time.Minute, "timeout for waiting for system to be running (0 to skip)")
	quiet := flag.Bool("quiet", false, "less logging output")
	flag.Parse()
	configFile := flag.Arg(0)
	if configFile == "" {
		log.Fatalf("Missing build config file, usage: %s <config.json>", os.Args[0])
	}

	var config *buildconfig.BuildConfig
	if configFile != "" {
		var err error
		config, err = buildconfig.New(configFile, nil)
		if err != nil {
			log.Fatalf("Failed to load build config: %v", err)
		}
	}

	if err := waitForSystem(*waitTimeout); err != nil {
		log.Println("Problem during waiting for system to be running, exit code 2:", err)
		os.Exit(2)
	}

	if !runChecks(checks, config, *quiet) {
		log.Printf("Check completed with config %q and return code 1\n", configFile)
		os.Exit(1)
	}
}
