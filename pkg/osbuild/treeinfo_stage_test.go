package osbuild

import (
	"encoding/json"
	"testing"

	"github.com/osbuild/images/pkg/arch"
	"github.com/stretchr/testify/assert"
)

func TestNewTreeinfoStage(t *testing.T) {
	type testCase struct {
		name         string
		version      string
		architecture arch.Arch
		variant      string
		kernel       string
		initrd       string
		stage2       string
		expectedJSON string
	}

	testCases := []testCase{
		{
			name:         "Fedora",
			version:      "42",
			architecture: arch.ARCH_X86_64,
			variant:      "Workstation",
			kernel:       "images/pxeboot/vmlinuz",
			initrd:       "images/pxeboot/initrd.img",
			stage2:       "LiveOS/squashfs.img",
			expectedJSON: `{
				"type": "org.osbuild.treeinfo",
				"options": {
					"path": ".treeinfo",
					"treeinfo": {
						"release": {
							"name": "Fedora",
							"version": "42"
						},
						"tree": {
							"arch": "x86_64",
							"platforms": ["x86_64"],
							"variants": ["Workstation"]
						},
						"checksums": [
							"images/pxeboot/vmlinuz",
							"images/pxeboot/initrd.img",
							"LiveOS/squashfs.img"
						],
						"stage2": {
							"mainimage": "LiveOS/squashfs.img"
						},
						"images-x86_64": {
							"kernel": "images/pxeboot/vmlinuz",
							"initrd": "images/pxeboot/initrd.img"
						},
						"variant-Workstation": {
							"id": "Workstation",
							"name": "Workstation",
							"type": "variant",
							"uid": "Workstation"
						},
						"general": {
							"arch": "x86_64",
							"family": "Fedora",
							"name": "Fedora 42",
							"platforms": ["x86_64"],
							"variant": "Workstation",
							"version": "42"
						}
					}
				}
			}`,
		},
		{
			name:         "CentOS Stream",
			version:      "9",
			architecture: arch.ARCH_AARCH64,
			variant:      "Server",
			kernel:       "images/pxeboot/vmlinuz",
			initrd:       "images/pxeboot/initrd.img",
			stage2:       "images/install.img",
			expectedJSON: `{
				"type": "org.osbuild.treeinfo",
				"options": {
					"path": ".treeinfo",
					"treeinfo": {
						"release": {
							"name": "CentOS Stream",
							"version": "9"
						},
						"tree": {
							"arch": "aarch64",
							"platforms": ["aarch64"],
							"variants": ["Server"]
						},
						"checksums": [
							"images/pxeboot/vmlinuz",
							"images/pxeboot/initrd.img",
							"images/install.img"
						],
						"stage2": {
							"mainimage": "images/install.img"
						},
						"images-aarch64": {
							"kernel": "images/pxeboot/vmlinuz",
							"initrd": "images/pxeboot/initrd.img"
						},
						"variant-Server": {
							"id": "Server",
							"name": "Server",
							"type": "variant",
							"uid": "Server"
						},
						"general": {
							"arch": "aarch64",
							"family": "CentOS Stream",
							"name": "CentOS Stream 9",
							"platforms": ["aarch64"],
							"variant": "Server",
							"version": "9"
						}
					}
				}
			}`,
		},
		{
			name:         "Red Hat Enterprise Linux",
			version:      "10.0",
			architecture: arch.ARCH_X86_64,
			variant:      "IoT",
			kernel:       "images/pxeboot/vmlinuz",
			initrd:       "images/pxeboot/initrd.img",
			stage2:       "images/install.img",
			expectedJSON: `{
				"type": "org.osbuild.treeinfo",
				"options": {
					"path": ".treeinfo",
					"treeinfo": {
						"release": {
							"name": "Red Hat Enterprise Linux",
							"version": "10.0"
						},
						"tree": {
							"arch": "x86_64",
							"platforms": ["x86_64"],
							"variants": ["IoT"]
						},
						"checksums": [
							"images/pxeboot/vmlinuz",
							"images/pxeboot/initrd.img",
							"images/install.img"
						],
						"stage2": {
							"mainimage": "images/install.img"
						},
						"images-x86_64": {
							"kernel": "images/pxeboot/vmlinuz",
							"initrd": "images/pxeboot/initrd.img"
						},
						"variant-IoT": {
							"id": "IoT",
							"name": "IoT",
							"type": "variant",
							"uid": "IoT"
						},
						"general": {
							"arch": "x86_64",
							"family": "Red Hat Enterprise Linux",
							"name": "Red Hat Enterprise Linux 10.0",
							"platforms": ["x86_64"],
							"variant": "IoT",
							"version": "10.0"
						}
					}
				}
			}`,
		},
		{
			name:         "AlmaLinux",
			version:      "10.0",
			architecture: arch.ARCH_X86_64,
			variant:      "", // No variant
			kernel:       "images/pxeboot/vmlinuz",
			initrd:       "images/pxeboot/initrd.img",
			stage2:       "images/install.img",
			expectedJSON: `{
				"type": "org.osbuild.treeinfo",
				"options": {
					"path": ".treeinfo",
					"treeinfo": {
						"release": {
							"name": "AlmaLinux",
							"version": "10.0"
						},
						"tree": {
							"arch": "x86_64",
							"platforms": ["x86_64"]
						},
						"checksums": [
							"images/pxeboot/vmlinuz",
							"images/pxeboot/initrd.img",
							"images/install.img"
						],
						"stage2": {
							"mainimage": "images/install.img"
						},
						"images-x86_64": {
							"kernel": "images/pxeboot/vmlinuz",
							"initrd": "images/pxeboot/initrd.img"
						},
						"general": {
							"arch": "x86_64",
							"family": "AlmaLinux",
							"name": "AlmaLinux 10.0",
							"platforms": ["x86_64"],
							"version": "10.0"
						}
					}
				}
			}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			stage := NewTreeinfoStage(tc.name, tc.version, tc.architecture, tc.variant, tc.kernel, tc.initrd, tc.stage2)

			actualBytes, err := json.Marshal(stage)
			assert.NoError(t, err)

			var actualMap, expectedMap map[string]any
			err = json.Unmarshal(actualBytes, &actualMap)
			assert.NoError(t, err)
			err = json.Unmarshal([]byte(tc.expectedJSON), &expectedMap)
			assert.NoError(t, err)

			assert.Equal(t, expectedMap, actualMap)
		})
	}
}
