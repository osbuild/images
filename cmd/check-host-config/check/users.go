package check

import (
	"context"
	"strings"
	"time"

	"github.com/osbuild/images/cmd/check-host-config/cos"
	"github.com/osbuild/images/internal/buildconfig"
)

type UsersCheck struct{}

func (u UsersCheck) Metadata() Metadata {
	return Metadata{
		Name:                   "Users Check",
		ShortName:              "users",
		Timeout:                10 * time.Second,
		RequiresBlueprint:      true,
		RequiresCustomizations: true,
	}
}

func (u UsersCheck) Run(ctx context.Context, log Logger, config *buildconfig.BuildConfig) error {
	users := config.Blueprint.Customizations.User
	if len(users) == 0 {
		return Skip("no users to check")
	}

	for _, user := range users {
		log.Printf("Checking user: %s\n", user.Name)
		out, err := cos.ExecContext(ctx, log, "id", user.Name)
		if err != nil {
			return Fail("user does not exist:", user.Name)
		}
		log.Printf("User %s exists: %s\n", user.Name, strings.TrimSpace(string(out)))
	}

	return Pass()
}
