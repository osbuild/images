package check

import (
	"context"
	"time"

	"github.com/osbuild/images/internal/buildconfig"
)

type BootTimeCheck struct{}

var (
	ProgramStartTime = time.Now()
	WarningThreshold = 10 * time.Minute
)

func (h BootTimeCheck) Metadata() Metadata {
	return Metadata{
		Name:      "Boot Time Check",
		ShortName: "boot-time",
		Timeout:   1 * time.Second,
	}
}

func (h BootTimeCheck) Run(ctx context.Context, log Logger, config *buildconfig.BuildConfig) error {
	// Before any check is run, we wait for systemd to report "running" state.
	// If this takes too long, we issue a warning because there are some hosts taking too long.
	if time.Since(ProgramStartTime) > WarningThreshold {
		return Warning("waiting for systemd is-system-running exceeded", WarningThreshold.String())
	}

	return Pass()
}
