package check

import (
	"context"
	"time"

	"github.com/osbuild/images/cmd/check-host-config/mockos"
	"github.com/osbuild/images/internal/buildconfig"
)

type Metadata struct {
	Name                   string        // Full name of the check
	ShortName              string        // Short name of the check used for logging and verbosity
	Timeout                time.Duration // Maximum time the check is allowed to run
	RequiresBlueprint      bool          // Ensure Blueprint is not nil, skip the check otherwise
	RequiresCustomizations bool          // Ensure Customizations is not nil, skip the check otherwise
	TempDisabled           string        // Set to non-empty string with URL to issue tracker to disable the check temporarily
}

// Logger is an alias to the simplified logger interface.
type Logger = mockos.Logger

type Check interface {
	Metadata() Metadata
	Run(ctx context.Context, log Logger, config *buildconfig.BuildConfig) error
}
