package osbuild

import "github.com/osbuild/images/pkg/customizations/users"

type KickstartStageOptions struct {
	// Where to place the kickstart file
	Path string `json:"path"`

	OSTreeCommit *OSTreeCommitOptions `json:"ostree,omitempty"`

	LiveIMG *LiveIMGOptions `json:"liveimg,omitempty"`

	Users map[string]UsersStageOptionsUser `json:"users,omitempty"`

	Groups map[string]GroupsStageOptionsGroup `json:"groups,omitempty"`
}

type LiveIMGOptions struct {
	URL string `json:"url"`
}

type OSTreeCommitOptions struct {
	OSName string `json:"osname"`
	URL    string `json:"url"`
	Ref    string `json:"ref"`
	GPG    bool   `json:"gpg"`
}

func (KickstartStageOptions) isStageOptions() {}

// Creates an Anaconda kickstart file
func NewKickstartStage(options *KickstartStageOptions) *Stage {
	return &Stage{
		Type:    "org.osbuild.kickstart",
		Options: options,
	}
}

func NewKickstartStageOptions(
	path string,
	imageURL string,
	userCustomizations []users.User,
	groupCustomizations []users.Group,
	ostreeURL string,
	ostreeRef string,
	osName string) (*KickstartStageOptions, error) {

	var users map[string]UsersStageOptionsUser
	if usersOptions, err := NewUsersStageOptions(userCustomizations, false); err != nil {
		return nil, err
	} else if usersOptions != nil {
		users = usersOptions.Users
	}

	var groups map[string]GroupsStageOptionsGroup
	if groupsOptions := NewGroupsStageOptions(groupCustomizations); groupsOptions != nil {
		groups = groupsOptions.Groups
	}

	var ostreeCommitOptions *OSTreeCommitOptions
	if ostreeURL != "" {
		ostreeCommitOptions = &OSTreeCommitOptions{
			OSName: osName,
			URL:    ostreeURL,
			Ref:    ostreeRef,
			GPG:    false,
		}
	}

	var liveImg *LiveIMGOptions
	if imageURL != "" {
		liveImg = &LiveIMGOptions{
			URL: imageURL,
		}
	}
	return &KickstartStageOptions{
		Path:         path,
		OSTreeCommit: ostreeCommitOptions,
		LiveIMG:      liveImg,
		Users:        users,
		Groups:       groups,
	}, nil
}
