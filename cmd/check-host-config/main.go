package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/osbuild/images/cmd/check-host-config/check"
	"github.com/osbuild/images/internal/buildconfig"
)

// Return codes:
//   0: All checks passed
//   1: Unused
//   2: Wait timeout
// 3-9: Reserved for future use
// 10+: Sequential return codes for each check
//
// When multiple checks fail, the return code is the first failure encountered
// which will be random. Do not rely on the order of the checks, or the return code.

var checks = []check.Check{
	check.HostnameCheck{},                 // return code 10
	check.FilesCheck{},                    // return code 11
	check.UsersCheck{},                    // return code 12
	check.ServicesEnabledCheck{},          // return code 13
	check.ServicesDisabledCheck{},         // return code 14
	check.ServicesMaskedCheck{},           // return code 15
	check.FirewallServicesEnabledCheck{},  // return code 16
	check.FirewallServicesDisabledCheck{}, // return code 17
	check.FirewallPortsCheck{},            // return code 18
	check.CACertsCheck{},                  // return code 19
	check.ModularityCheck{},               // return code 20
	check.OpenSCAPCheck{},                 // return code 21
}

var MaxShortCheckName int

func init() {
	for _, check := range checks {
		nameLen := len(check.Metadata().ShortName)
		if nameLen > MaxShortCheckName {
			MaxShortCheckName = nameLen
		}
	}
}

type Result struct {
	Check      check.Check
	Error      error
	ReturnCode int
	Logs       *strings.Builder
}

func main() {
	configFile := flag.String("config", "", "build config file")
	waitTimeout := flag.Duration("wait-timeout", 15*time.Minute, "timeout for waiting for system to be running (0 to skip)")
	quiet := flag.Bool("quiet", false, "less logging output")
	flag.Parse()

	logger := NewLogger(os.Stdout, "main", *quiet)
	var returnCode int

	var config *buildconfig.BuildConfig
	if *configFile != "" {
		var err error
		logger.Printf("Loading build config from %s\n", *configFile)
		config, err = buildconfig.New(*configFile, nil)
		if err != nil {
			log.Fatalf("Failed to load build config: %v", err)
		}
	}

	if *waitTimeout > 0 {
		waitCtx, waitCancel := context.WithTimeout(context.Background(), *waitTimeout)
		defer waitCancel()

		if err := runningWait(waitCtx, logger); err != nil {
			fmt.Fprintf(os.Stderr, "Error while waiting for system to be running: %v\n", err)
			// If it's a timeout, check which units are still activating
			if errors.Is(err, context.DeadlineExceeded) {
				activatingUnits := getActivatingUnits(context.Background())
				if activatingUnits != "" {
					fmt.Fprintf(os.Stderr, "Units still activating: %s\n", activatingUnits)
				}
			}
			os.Exit(2)
		}
	}

	results := make(chan *Result, len(checks))
	wgSrv := sync.WaitGroup{}
	wgSrv.Add(1)
	go func() {
		defer wgSrv.Done()
		passed := &strings.Builder{}
		warn := &strings.Builder{}
		fail := &strings.Builder{}

		for result := range results {
			meta := result.Check.Metadata()

			switch {
			case check.IsWarning(result.Error):
				fmt.Fprintf(warn, "⚠️  %s (%v)\n", meta.Name, result.Error)
			case check.IsFail(result.Error):
				fmt.Fprintf(fail, "❌ %s (%v)\n", meta.Name, result.Error)
				// When multiple checks fail, we return the first failure code encountered.
				if returnCode == 0 {
					returnCode = result.ReturnCode
				}
			case result.Error == nil:
				fmt.Fprintf(passed, "✅ %s\n", meta.Name)
			}

			if !*quiet {
				fmt.Print(result.Logs.String())
			}
		}

		fmt.Print("Results:\n")
		fmt.Print(passed.String())
		fmt.Print(warn.String())
		fmt.Print(fail.String())
	}()

	wgCheck := sync.WaitGroup{}
	for i, chk := range checks {
		wgCheck.Add(1)

		go func(c check.Check, checkIndex int) {
			defer wgCheck.Done()
			meta := c.Metadata()

			ctx, cancel := context.WithTimeout(context.Background(), meta.Timeout)
			defer cancel()

			result := &Result{
				Check:      c,
				Logs:       &strings.Builder{},
				ReturnCode: 10 + checkIndex, // Return codes start at 10
			}
			defer func() {
				results <- result
			}()

			logger := NewLogger(result.Logs, meta.ShortName, *quiet)
			if meta.RequiresBlueprint && (config == nil || config.Blueprint == nil) {
				result.Error = check.Skip("no blueprint")
				return
			}
			if meta.RequiresCustomizations && (config == nil || config.Blueprint == nil || config.Blueprint.Customizations == nil) {
				result.Error = check.Skip("no customizations")
				return
			}

			if err := c.Run(ctx, logger, config); err != nil {
				result.Error = err
			}
		}(chk, i)
	}

	wgCheck.Wait()
	close(results)
	wgSrv.Wait()

	os.Exit(returnCode)
}
