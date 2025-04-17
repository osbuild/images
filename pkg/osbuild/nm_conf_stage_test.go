package osbuild

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNMConfStageOptionsValidation(t *testing.T) {
	testCases := map[string]struct {
		options       NMConfStageOptions
		expectedError string
	}{
		"valid-main": {
			options: NMConfStageOptions{
				Path: "/etc/NetworkManager/conf.d/valid-main.conf",
				Settings: NMConfStageSettings{
					Main: &NMConfSettingsMain{
						NoAutoDefault: []string{"eth0", "eth1"},
						Plugins:       []string{"keyfile"},
					},
				},
			},
		},
		"valid-device": {
			options: NMConfStageOptions{
				Path: "/etc/NetworkManager/conf.d/valid-device.conf",
				Settings: NMConfStageSettings{
					Device: []NMConfSettingsDevice{
						{
							Name: "eth42",
							Config: NMConfDeviceConfig{
								Managed:                true,
								WifiScanRandMacAddress: false,
							},
						},
						{
							Name: "eth99",
							Config: NMConfDeviceConfig{
								Managed:                true,
								WifiScanRandMacAddress: true,
							},
						},
					},
				},
			},
		},
		"valid-gdd": {
			options: NMConfStageOptions{
				Path: "/etc/NetworkManager/conf.d/valid-dd.conf",
				Settings: NMConfStageSettings{
					GlobalDNSDomain: []NMConfSettingsGlobalDNSDomain{
						{
							Name: "whatever",
							Config: NMConfSettingsGlobalDNSDomainConfig{
								Servers: []string{
									"server1",
									"server13",
								},
							},
						},
					},
				},
			},
		},
		"valid-keyfile": {
			options: NMConfStageOptions{
				Path: "/valid/path",
				Settings: NMConfStageSettings{
					Keyfile: &NMConfSettingsKeyfile{
						UnmanagedDevices: []string{"eth0", "eth900"},
					},
				},
			},
		},
		"no-path": {
			options: NMConfStageOptions{
				Settings: NMConfStageSettings{
					Main: &NMConfSettingsMain{
						NoAutoDefault: []string{"eth0"},
						Plugins:       []string{"keyfile"},
					},
				},
			},
			expectedError: "org.osbuild.nm.conf: path is a required property",
		},
		"empty-path": {
			options: NMConfStageOptions{
				Path: "",
				Settings: NMConfStageSettings{
					Main: &NMConfSettingsMain{
						NoAutoDefault: []string{"eth0"},
						Plugins:       []string{"keyfile"},
					},
				},
			},
			expectedError: "org.osbuild.nm.conf: path is a required property",
		},
		"no-settings": {
			options: NMConfStageOptions{
				Path: "/valid/path",
			},
			expectedError: "org.osbuild.nm.conf: at least one setting must be set",
		},
		"main-empty-nad": {
			options: NMConfStageOptions{
				Path: "/valid/path",
				Settings: NMConfStageSettings{
					Main: &NMConfSettingsMain{
						NoAutoDefault: []string{},
					},
				},
			},
			expectedError: "org.osbuild.nm.conf: main.no-auto-default requires at least one element when defined",
		},
		"main-empty-plugins": {
			options: NMConfStageOptions{
				Path: "/valid/path",
				Settings: NMConfStageSettings{
					Main: &NMConfSettingsMain{
						Plugins: []string{},
					},
				},
			},
			expectedError: "org.osbuild.nm.conf: main.plugins requires at least one element when defined",
		},
		"global-dnf-domain-no-name": {
			options: NMConfStageOptions{
				Path: "/valid/path",
				Settings: NMConfStageSettings{
					GlobalDNSDomain: []NMConfSettingsGlobalDNSDomain{
						{
							Name: "",
							Config: NMConfSettingsGlobalDNSDomainConfig{
								Servers: []string{"8.8.8.8"},
							},
						},
					},
				},
			},
			expectedError: "org.osbuild.nm.conf: global-dns-domain name is a required property",
		},
		"keyfile-no-devices": {
			options: NMConfStageOptions{
				Path: "/valid/path",
				Settings: NMConfStageSettings{
					Keyfile: &NMConfSettingsKeyfile{
						UnmanagedDevices: []string{},
					},
				},
			},
			expectedError: "org.osbuild.nm.conf: keyfile.unmanaged-devices requires at least one element when defined",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			err := tc.options.validate()
			assert := assert.New(t)
			if tc.expectedError == "" {
				assert.NoError(err)
			} else {
				assert.EqualError(err, tc.expectedError)
			}
		})
	}
}
