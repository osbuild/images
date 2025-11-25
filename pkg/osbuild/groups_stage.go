package osbuild

import (
	"github.com/osbuild/images/pkg/customizations/users"
)

type GroupsStageOptions struct {
	Groups map[string]GroupsStageOptionsGroup `json:"groups"`
}

func (GroupsStageOptions) isStageOptions() {}

var _ = PathChanger(GroupsStageOptions{})

func (g GroupsStageOptions) PathsChanged() []string {
	// this is very precise but because we don't own the groupadd code if it
	// would start writing some extra/aux files (in a few years) we would
	// not catch it so maybe just return /etc here?
	return []string{"/etc/group", "/etc/gshadow", "/etc/group-", "/etc/gshadow-"}
}

type GroupsStageOptionsGroup struct {
	GID *int `json:"gid,omitempty"`
}

func NewGroupsStage(options *GroupsStageOptions) *Stage {
	return &Stage{
		Type:    "org.osbuild.groups",
		Options: options,
	}
}

func NewGroupsStageOptions(groups []users.Group) *GroupsStageOptions {
	if len(groups) == 0 {
		return nil
	}

	options := GroupsStageOptions{
		Groups: map[string]GroupsStageOptionsGroup{},
	}

	for _, group := range groups {
		options.Groups[group.Name] = GroupsStageOptionsGroup{
			GID: group.GID,
		}
	}

	return &options
}

func GenGroupsStage(groups []users.Group) *Stage {
	return NewGroupsStage(NewGroupsStageOptions(groups))
}
