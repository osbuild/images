package users

import (
	"github.com/osbuild/images/internal/types"
	"github.com/osbuild/images/pkg/blueprint"
)

type User struct {
	Name               string
	Description        types.Option[string]
	Password           types.Option[string]
	Key                types.Option[string]
	Home               types.Option[string]
	Shell              types.Option[string]
	Groups             []string
	UID                types.Option[int]
	GID                types.Option[int]
	ExpireDate         types.Option[int]
	ForcePasswordReset types.Option[bool]
}

type Group struct {
	Name string
	GID  *int
}

func UsersFromBP(userCustomizations []blueprint.UserCustomization) []User {
	users := make([]User, len(userCustomizations))
	for idx := range userCustomizations {
		// currently, they have the same structure, so we convert directly
		// this will fail to compile as soon as one of the two changes
		users[idx] = User(userCustomizations[idx])
	}
	return users
}

func GroupsFromBP(groupCustomizations []blueprint.GroupCustomization) []Group {
	groups := make([]Group, len(groupCustomizations))
	for idx := range groupCustomizations {
		// currently, they have the same structure, so we convert directly
		// this will fail to compile as soon as one of the two changes
		groups[idx] = Group(groupCustomizations[idx])
	}
	return groups
}
