package osbuild_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/customizations/users"
	"github.com/osbuild/images/pkg/osbuild"
)

func TestKickstartStageJsonHappy(t *testing.T) {
	opts := &osbuild.KickstartStageOptions{
		Path: "/osbuild.ks",
		Bootloader: &osbuild.BootloaderOptions{
			Append: "karg1 karg2=0",
		},
	}
	stage := osbuild.NewKickstartStage(opts)
	require.NotNil(t, stage)
	stageJson, err := json.MarshalIndent(stage, "", "  ")
	require.Nil(t, err)
	assert.Equal(t, string(stageJson), `{
  "type": "org.osbuild.kickstart",
  "options": {
    "path": "/osbuild.ks",
    "bootloader": {
      "append": "karg1 karg2=0"
    }
  }
}`)
}

func TestKickstartStageUsers(t *testing.T) {
	type testCase struct {
		users    []users.User
		expected *osbuild.KickstartStageOptions
		expErr   string
	}

	testCases := map[string]testCase{
		"empty": {
			users:    nil,
			expected: &osbuild.KickstartStageOptions{},
			expErr:   "",
		},
		"1-user": {
			users: []users.User{
				{
					Name:               "user",
					Description:        common.ToPtr("I am user"),
					Password:           common.ToPtr("$6$fakesalt$fakehashedpassword"),
					Key:                common.ToPtr("ssh-ed25519 AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"),
					Home:               common.ToPtr("/var/home/user"),
					Shell:              common.ToPtr("/usr/bin/fish"),
					Groups:             []string{"grp1", "wheel"},
					UID:                common.ToPtr(1010),
					GID:                common.ToPtr(1020),
					ExpireDate:         common.ToPtr(1756486205),
					ForcePasswordReset: common.ToPtr(false),
				},
			},
			expected: &osbuild.KickstartStageOptions{
				Users: map[string]osbuild.UsersStageOptionsUser{
					"user": {
						UID:                common.ToPtr(1010),
						GID:                common.ToPtr(1020),
						Groups:             []string{"grp1", "wheel"},
						Description:        common.ToPtr("I am user"),
						Home:               common.ToPtr("/var/home/user"),
						Shell:              common.ToPtr("/usr/bin/fish"),
						Password:           common.ToPtr("$6$fakesalt$fakehashedpassword"),
						Key:                common.ToPtr("ssh-ed25519 AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"),
						ExpireDate:         common.ToPtr(1756486205),
						ForcePasswordReset: common.ToPtr(false),
					},
				},
			},
			expErr: "",
		},
		"2-user+root": {
			users: []users.User{
				{
					Name:               "user",
					Description:        common.ToPtr("I am user"),
					Password:           common.ToPtr("$6$fakesalt$fakehashedpassword"),
					Key:                common.ToPtr("ssh-ed25519 AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"),
					Home:               common.ToPtr("/var/home/user"),
					Shell:              common.ToPtr("/usr/bin/fish"),
					Groups:             []string{"grp1", "wheel"},
					UID:                common.ToPtr(1010),
					GID:                common.ToPtr(1020),
					ExpireDate:         common.ToPtr(1756486205),
					ForcePasswordReset: common.ToPtr(false),
				},
				{
					Name:               "root",
					Description:        common.ToPtr("super!"),
					Password:           common.ToPtr("$6$fakesaltroot$fakehashedpasswordroot"),
					Key:                common.ToPtr("ssh-ed25519 BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB"),
					Home:               common.ToPtr("/rooot"),
					Shell:              common.ToPtr("/usr/bin/zsh"),
					Groups:             []string{"wheel?"},
					UID:                common.ToPtr(10),
					GID:                common.ToPtr(20),
					ExpireDate:         common.ToPtr(1756486205),
					ForcePasswordReset: common.ToPtr(false),
				},
			},
			expected: &osbuild.KickstartStageOptions{
				Users: map[string]osbuild.UsersStageOptionsUser{
					"user": {
						Description:        common.ToPtr("I am user"),
						Password:           common.ToPtr("$6$fakesalt$fakehashedpassword"),
						Key:                common.ToPtr("ssh-ed25519 AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"),
						Home:               common.ToPtr("/var/home/user"),
						Shell:              common.ToPtr("/usr/bin/fish"),
						Groups:             []string{"grp1", "wheel"},
						UID:                common.ToPtr(1010),
						GID:                common.ToPtr(1020),
						ExpireDate:         common.ToPtr(1756486205),
						ForcePasswordReset: common.ToPtr(false),
					},
					"root": {
						Description:        common.ToPtr("super!"),
						Password:           common.ToPtr("$6$fakesaltroot$fakehashedpasswordroot"),
						Key:                common.ToPtr("ssh-ed25519 BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB"),
						Home:               common.ToPtr("/rooot"),
						Shell:              common.ToPtr("/usr/bin/zsh"),
						Groups:             []string{"wheel?"},
						UID:                common.ToPtr(10),
						GID:                common.ToPtr(20),
						ExpireDate:         common.ToPtr(1756486205),
						ForcePasswordReset: common.ToPtr(false),
					},
				},
			},
			expErr: "",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			ksOpts, err := osbuild.NewKickstartStageOptions("", tc.users, nil)
			if tc.expErr != "" {
				assert.EqualError(err, tc.expErr)
				return
			}

			assert.NoError(err)
			assert.Equal(tc.expected, ksOpts)
		})
	}

}
