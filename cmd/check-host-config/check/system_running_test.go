package check_test

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	check "github.com/osbuild/images/cmd/check-host-config/check"
	"github.com/osbuild/images/internal/buildconfig"
	"github.com/stretchr/testify/assert"
)

func TestBootTimeCheck(t *testing.T) {
	// Save original values
	// XXX: Use Go 1.24 test time mocking when we switch to it
	origStartTime := check.ProgramStartTime
	origThreshold := check.WarningThreshold
	defer func() {
		check.ProgramStartTime = origStartTime
		check.WarningThreshold = origThreshold
	}()

	chk := check.BootTimeCheck{}
	logger := log.New(os.Stdout, "", 0)
	config := &buildconfig.BuildConfig{}

	// Case 1: Fast enough
	check.ProgramStartTime = time.Now()
	check.WarningThreshold = 1 * time.Hour
	err := chk.Run(context.Background(), logger, config)
	assert.NoError(t, err)

	// Case 2: Too slow (Warning)
	// We simulate that the program started 1 minute ago, and threshold is 1 second.
	check.ProgramStartTime = time.Now().Add(-1 * time.Minute)
	check.WarningThreshold = 1 * time.Second
	err = chk.Run(context.Background(), logger, config)
	assert.Error(t, err)
	assert.True(t, check.IsWarning(err))
}
