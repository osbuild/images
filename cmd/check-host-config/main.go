package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/osbuild/images/cmd/check-host-config/check"
	"github.com/osbuild/images/internal/buildconfig"
)

// Return codes:
//   0: All checks passed
//   1: Unused
//   2: Wait timeout
// 3-9: Reserved for future use
// 10+: Sequential return codes for each check (see checks.go)
//
// When multiple checks fail, the return code is the failed check with the highest
// number to ensure consistent behavior.

type Result struct {
	Check      check.Check
	Error      error
	ReturnCode int
	Logs       *strings.Builder
}

// waitForSystem waits until the system is reported by systemd as "running" or the timeout is reached.
func waitForSystem(ctx context.Context, timeout time.Duration, logger *log.Logger) error {
	if timeout <= 0 {
		return nil
	}

	waitCtx, waitCancel := context.WithTimeout(ctx, timeout)
	defer waitCancel()

	if err := runningWait(waitCtx, logger); err != nil {
		fmt.Fprintf(os.Stderr, "Error while waiting for system to be running: %v\n", err)
		if errors.Is(err, context.DeadlineExceeded) {
			if activatingUnits := listBadUnits(ctx); activatingUnits != "" {
				fmt.Fprintf(os.Stderr, "Units still activating: %s\n", activatingUnits)
			}
		}
		return err
	}
	return nil
}

// collectResults collects results from the results channel and prints them in order.
func collectResults(results chan *Result, returnCode *int, quiet bool, wg *sync.WaitGroup) {
	defer wg.Done()

	pass := &strings.Builder{}
	warn := &strings.Builder{}
	fail := &strings.Builder{}
	skip := &strings.Builder{}

	for result := range results {
		meta := result.Check.Metadata()

		switch {
		case check.IsSkip(result.Error):
			fmt.Fprintf(skip, "⏭️  %s (%v)\n", meta.Name, result.Error)
		case check.IsWarning(result.Error):
			fmt.Fprintf(warn, "⚠️  %s (%v)\n", meta.Name, result.Error)
		case check.IsFail(result.Error):
			fmt.Fprintf(fail, "❌ %s (%v)\n", meta.Name, result.Error)
			if *returnCode < result.ReturnCode {
				*returnCode = result.ReturnCode
			}
		case result.Error == nil:
			fmt.Fprintf(pass, "✅ %s\n", meta.Name)
		}

		if !quiet {
			fmt.Print(result.Logs.String())
		}
	}

	fmt.Print("Results:\n")
	fmt.Print(skip.String())
	fmt.Print(pass.String())
	fmt.Print(warn.String())
	fmt.Print(fail.String())
}

// runChecks runs all checks concurrently and sends their results to the results channel. Each check has its own
// logger so their logs do not interleave.
func runChecks(parentCtx context.Context, checks []check.Check, results chan *Result, config *buildconfig.BuildConfig, quiet bool) {
	var wg sync.WaitGroup
	for i, chk := range checks {
		wg.Add(1)
		go func(c check.Check, checkIndex int) {
			defer wg.Done()

			meta := c.Metadata()
			ctx, cancel := context.WithTimeout(parentCtx, meta.Timeout)
			defer cancel()

			result := &Result{
				Check:      c,
				Logs:       &strings.Builder{},
				ReturnCode: 10 + checkIndex,
			}
			defer func() { results <- result }()

			if meta.TempDisabled != "" {
				result.Error = check.Skip("temporarily disabled: " + meta.TempDisabled)
				return
			}

			logger := NewLogger(result.Logs, meta.ShortName, quiet)
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

	wg.Wait()
	close(results)
}

func main() {
	log.SetFlags(0)
	ctx, sigStop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer sigStop()

	waitTimeout := flag.Duration("wait-timeout", 15*time.Minute, "timeout for waiting for system to be running (0 to skip)")
	quiet := flag.Bool("quiet", false, "less logging output")
	flag.Parse()
	configFile := flag.Arg(0)
	if configFile == "" {
		log.Fatalf("Missing build config file, usage: %s <config.json>", os.Args[0])
	}

	logger := NewLogger(os.Stdout, "main", *quiet)
	var returnCode int

	var config *buildconfig.BuildConfig
	if configFile != "" {
		var err error
		config, err = buildconfig.New(configFile, nil)
		if err != nil {
			log.Fatalf("Failed to load build config: %v", err)
		}
	}

	if err := waitForSystem(ctx, *waitTimeout, logger); err != nil {
		if ctx.Err() != nil && errors.Is(ctx.Err(), context.Canceled) {
			logger.Println("Received termination signal, exiting")
		} else {
			logger.Println("Problem during waiting for system to be running:", err)
		}
		os.Exit(2)
	}

	results := make(chan *Result, len(checks))
	var wgCollect sync.WaitGroup
	wgCollect.Add(1)
	go collectResults(results, &returnCode, *quiet, &wgCollect)

	runChecks(ctx, checks, results, config, *quiet)
	wgCollect.Wait()

	logger.Printf("Check completed with config %q and return code %d\n", configFile, returnCode)
	os.Exit(returnCode)
}
