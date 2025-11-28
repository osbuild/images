package check

import (
	"context"
	"time"

	"github.com/osbuild/images/cmd/check-host-config/cos"
	"github.com/osbuild/images/internal/buildconfig"
)

type FilesCheck struct{}

func (h FilesCheck) Metadata() Metadata {
	return Metadata{
		Name:                   "Files Check",
		ShortName:              "files",
		Timeout:                5 * time.Second,
		RequiresBlueprint:      true,
		RequiresCustomizations: true,
	}
}

func (h FilesCheck) Run(ctx context.Context, log Logger, config *buildconfig.BuildConfig) error {
	// Note that this test only checks for the existance of the filesystem
	// customizatons target path not the content. For the simple case when
	// "data" is provided we could check but for the "uri" case we do not
	// know the content as the file usually comes from the host.  The
	// existing testing framework makes the content check difficult, so we
	// settle for this for now. There is an alternative approach in
	// https://github.com/osbuild/images/pull/1157/commits/7784f3dc6b435fa03951263e48ea7cfca84c2ebd
	// that may eventually be considered that is more direct and runs
	// runs locally but different from the existing paradigm so it
	// needs further discussion.
	expected := config.Blueprint.Customizations.Files

	for _, file := range expected {
		log.Printf("Checking existence of file: %s\n", file.Path)
		if !cos.ExistsContext(ctx, log, file.Path) {
			return Fail("file does not exist:", file.Path)
		}
	}

	return Pass()
}
