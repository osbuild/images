package distro_test

import (
	"reflect"
	"testing"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/distro"
	"github.com/stretchr/testify/assert"
)

func TestValidateSupportedConfig(t *testing.T) {
	type testCase struct {
		supported []string
		config    any
		expErr    string
	}

	type pkg struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}

	type user struct {
		Name     string `json:"name"`
		Password string `json:"password"`
	}

	type systemd struct {
		Enable  []string `json:"enable"`
		Disable []string `json:"disable"`
	}

	type name struct {
		Name string `json:"name"`
	}

	type custWithEmbed struct {
		Type string `json:"type"`
		name
	}

	type customizations struct {
		Users   []user        `json:"users"`
		Systemd *systemd      `json:"systemd"`
		Embed   custWithEmbed `json:"embed"`
	}

	type testConfigType struct {
		Name              string         `json:"name"`
		Enable            *bool          `json:"enable"`
		Packages          []pkg          `json:"packages"`
		InstallerPackages []pkg          `json:"installer_packages"`
		Customizations    customizations `json:"customizations"`
	}

	testCases := map[string]testCase{
		"empty": {
			supported: nil,
			config:    struct{}{},
			expErr:    "",
		},
		"simple": {
			supported: []string{
				"name",
				"packages",
				"customizations",
			},
			config: testConfigType{
				Name: "test_1",
				Packages: []pkg{
					{
						Name: "osbuild-composer",
					},
				},
			},
			expErr: "",
		},
		"nested": {
			supported: []string{
				"name",
				"packages",
				"installer_packages.name",
				"customizations.systemd",
			},
			config: testConfigType{
				Name: "test_2",
				Packages: []pkg{
					{
						Name:    "osbuild",
						Version: "100",
					},
				},
				InstallerPackages: []pkg{
					{
						Name: "btrfs-tools",
					},
				},
				Customizations: customizations{
					Systemd: &systemd{
						Enable:  []string{"sshd.service", "cockpit.socket"},
						Disable: []string{"firewalld.service"},
					},
				},
			},
			expErr: "",
		},
		"nested-with-pointer": {
			supported: []string{
				"name",
				"packages",
				"installer_packages.name",
				"customizations.systemd.enable",
			},
			config: testConfigType{
				Name: "test_2",
				Packages: []pkg{
					{
						Name:    "osbuild",
						Version: "100",
					},
				},
				InstallerPackages: []pkg{
					{
						Name: "btrfs-tools",
					},
				},
				Customizations: customizations{
					Systemd: &systemd{
						Enable: []string{"sshd.service", "cockpit.socket"},
					},
				},
			},
			expErr: "",
		},
		"installer-not-allowed": {
			supported: []string{
				"name",
				"packages",
				"customizations",
			},
			config: testConfigType{
				Name: "test_1",
				Packages: []pkg{
					{
						Name: "osbuild-composer",
					},
				},
				InstallerPackages: []pkg{
					{
						Name: "lvm2",
					},
				},
			},
			expErr: "installer_packages: not supported",
		},
		"enable-not-allowed": {
			supported: []string{
				"name",
				"packages",
				"customizations",
			},
			config: testConfigType{
				Name:   "test_1",
				Enable: common.ToPtr(false),
				Packages: []pkg{
					{
						Name: "osbuild-composer",
					},
				},
			},
			expErr: "enable: not supported",
		},
		"installer.version-not-allowed": {
			supported: []string{
				"name",
				"packages",
				"installer_packages.name",
				"customizations.systemd",
			},
			config: testConfigType{
				Name: "test_2",
				Packages: []pkg{
					{
						Name:    "osbuild",
						Version: "100",
					},
				},
				InstallerPackages: []pkg{
					{
						Name: "btrfs-tools",
					},
					{
						Name:    "lvm2",
						Version: "22",
					},
				},
				Customizations: customizations{
					Systemd: &systemd{
						Enable:  []string{"sshd.service", "cockpit.socket"},
						Disable: []string{"firewalld.service"},
					},
				},
			},
			expErr: "installer_packages[1].version: not supported",
		},
		"customizations.user-not-supported": {
			supported: []string{
				"name",
				"packages",
				"installer_packages.name",
				"customizations.systemd",
			},
			config: testConfigType{
				Name: "test_2",
				Packages: []pkg{
					{
						Name:    "osbuild",
						Version: "100",
					},
				},
				InstallerPackages: []pkg{
					{
						Name: "btrfs-tools",
					},
				},
				Customizations: customizations{
					Systemd: &systemd{
						Enable:  []string{"sshd.service", "cockpit.socket"},
						Disable: []string{"firewalld.service"},
					},
					Users: []user{
						{
							Name: "Bob",
						},
					},
				},
			},
			expErr: "customizations.users: not supported",
		},
		"customizations.embed-supported": {
			supported: []string{
				"name",
				"customizations.embed",
			},
			config: testConfigType{
				Customizations: customizations{
					Embed: custWithEmbed{
						name: name{
							Name: "embedded structure",
						},
					},
				},
			},
		},
		"customizations.embed-child-not-supported": {
			supported: []string{
				"name",
				"customizations.embed.type",
			},
			config: testConfigType{
				Customizations: customizations{
					Embed: custWithEmbed{
						name: name{
							Name: "embedded structure",
						},
					},
				},
			},
			expErr: "customizations.embed.name: not supported",
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			v := reflect.ValueOf(tc.config)
			err := distro.ValidateSupportedConfig(tc.supported, v)
			if tc.expErr == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expErr)
			}
		})
	}
}

func TestValidateRequiredConfig(t *testing.T) {
	type testCase struct {
		required []string
		config   any
		expErr   string
	}

	type pkg struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}

	type user struct {
		Name     string `json:"name"`
		Password string `json:"password"`
	}

	type systemd struct {
		Enable  []string `json:"enable"`
		Disable []string `json:"disable"`
	}

	type name struct {
		Name string `json:"name"`
	}

	type custWithEmbed struct {
		Type string `json:"type"`
		name
	}

	type customizations struct {
		Users   []user        `json:"users"`
		Systemd *systemd      `json:"systemd"`
		Embed   custWithEmbed `json:"embed"`
	}

	type testConfigType struct {
		Name              string         `json:"name"`
		Enable            *bool          `json:"enable"`
		Packages          []pkg          `json:"packages"`
		InstallerPackages []pkg          `json:"installer_packages"`
		Customizations    customizations `json:"customizations"`
	}

	testCases := map[string]testCase{
		"empty": {
			required: nil,
			config:   struct{}{},
			expErr:   "",
		},
		"simple": {
			required: []string{
				"name",
				"packages",
			},
			config: testConfigType{
				Name: "test_1",
				Packages: []pkg{
					{
						Name: "osbuild-composer",
					},
				},
			},
			expErr: "",
		},
		"nested": {
			required: []string{
				"name",
				"packages",
				"installer_packages.name",
				"customizations.systemd",
			},
			config: testConfigType{
				Name: "test_2",
				Packages: []pkg{
					{
						Name:    "osbuild",
						Version: "100",
					},
				},
				InstallerPackages: []pkg{
					{
						Name: "btrfs-tools",
					},
				},
				Customizations: customizations{
					Systemd: &systemd{
						Enable:  []string{"sshd.service", "cockpit.socket"},
						Disable: []string{"firewalld.service"},
					},
				},
			},
			expErr: "",
		},
		"nested-pointer": {
			required: []string{
				"name",
				"packages",
				"installer_packages.name",
				"customizations.systemd.enable",
			},
			config: testConfigType{
				Name: "test_2",
				Packages: []pkg{
					{
						Name:    "osbuild",
						Version: "100",
					},
				},
				InstallerPackages: []pkg{
					{
						Name: "btrfs-tools",
					},
				},
				Customizations: customizations{
					Systemd: &systemd{
						Enable:  []string{"sshd.service", "cockpit.socket"},
						Disable: []string{"firewalld.service"},
					},
				},
			},
			expErr: "",
		},
		"customizations-required": {
			required: []string{
				"name",
				"packages",
				"customizations",
			},
			config: testConfigType{
				Name: "test_1",
				Packages: []pkg{
					{
						Name: "osbuild-composer",
					},
				},
				InstallerPackages: []pkg{
					{
						Name: "lvm2",
					},
				},
			},
			expErr: "customizations: required",
		},
		"name-required": {
			required: []string{
				"name",
				"packages",
			},
			config: testConfigType{
				Packages: []pkg{
					{
						Name: "osbuild-composer",
					},
				},
			},
			expErr: "name: required",
		},
		"user.name-required": {
			required: []string{
				"name",
				"packages",
				"customizations.users.name",
			},
			config: testConfigType{
				Name: "test_2",
				Packages: []pkg{
					{
						Name:    "osbuild",
						Version: "100",
					},
				},
				Customizations: customizations{
					Users: []user{
						{
							Name:     "me",
							Password: "moi",
						},
						{
							Password: "I have no name but I must pass",
						},
					},
				},
			},
			expErr: "customizations.users[1].name: required",
		},
		"customizations.systemd-required": {
			required: []string{
				"name",
				"packages",
				"customizations.systemd",
			},
			config: testConfigType{
				Name: "test_2",
				Packages: []pkg{
					{
						Name:    "osbuild",
						Version: "100",
					},
				},
				Customizations: customizations{
					Users: []user{
						{
							Name: "Bob",
						},
					},
				},
			},
			expErr: "customizations.systemd: required",
		},
		"required-does-not-exist": {
			required: []string{
				"customizations.does-not-exist",
			},
			config: testConfigType{
				Customizations: customizations{
					Users: []user{
						{
							Name: "Bob",
						},
					},
				},
			},
			expErr: "customizations.does-not-exist: customizations does not have a field with JSON tag \"does-not-exist\"",
		},
		"customizations.embed-required-ok": {
			required: []string{
				"customizations.embed",
			},
			config: testConfigType{
				Customizations: customizations{
					Embed: custWithEmbed{
						name: name{
							Name: "embedded structure",
						},
					},
				},
			},
		},
		"customizations.embed-required-error": {
			required: []string{
				"customizations.embed.name",
			},
			config: testConfigType{
				Customizations: customizations{
					Embed: custWithEmbed{
						Type: "t",
					},
				},
			},
			expErr: "customizations.embed.name: required",
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			v := reflect.ValueOf(tc.config)
			err := distro.ValidateRequiredConfig(tc.required, v)
			if tc.expErr == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expErr)
			}
		})
	}
}
