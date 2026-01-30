package check_test

import (
	"errors"
	"testing"

	"github.com/osbuild/blueprint/pkg/blueprint"
	check "github.com/osbuild/images/cmd/check-host-config/check"
	"github.com/osbuild/images/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKernelCheck(t *testing.T) {
	tests := []struct {
		name          string
		kernel        *blueprint.KernelCustomization
		rpmError      error
		rpmExitCode   int
		readFileData  []byte
		readFileError error
		wantError     bool
		wantSkip      bool
		wantFail      bool
	}{
		{
			name:      "skip when kernel is nil",
			kernel:    nil,
			wantError: true,
			wantSkip:  true,
		},
		{
			name: "pass with empty append and no name",
			kernel: &blueprint.KernelCustomization{
				Append: "",
			},
			wantError: false,
		},
		{
			name: "pass with matching append and no name",
			kernel: &blueprint.KernelCustomization{
				Append: "debug",
			},
			readFileData: []byte("BOOT_IMAGE=/vmlinuz-6.1.0 root=UUID=1234-5678 ro quiet debug"),
			wantError:    false,
		},
		{
			name: "pass with matching kernel name",
			kernel: &blueprint.KernelCustomization{
				Name: "kernel",
			},
			rpmExitCode: 0,
			wantError:   false,
		},
		{
			name: "fail when rpm query fails",
			kernel: &blueprint.KernelCustomization{
				Name: "kernel",
			},
			rpmError:    errors.New("rpm command failed"),
			rpmExitCode: 1,
			wantError:   true,
			wantFail:    true,
		},
		{
			name: "fail when append does not match",
			kernel: &blueprint.KernelCustomization{
				Append: "debug",
			},
			readFileData: []byte("BOOT_IMAGE=/vmlinuz-6.1.0 root=UUID=1234-5678 ro quiet"),
			wantError:    true,
			wantFail:     true,
		},
		{
			name: "pass with matching kernel-debug name",
			kernel: &blueprint.KernelCustomization{
				Name: "kernel-debug",
			},
			rpmExitCode: 0,
			wantError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock rpm command
			if tt.kernel != nil && tt.kernel.Name != "" {
				test.MockGlobal(t, &check.Exec, func(name string, arg ...string) ([]byte, []byte, int, error) {
					if name == "rpm" && len(arg) >= 3 && arg[0] == "-q" && arg[1] == "--provides" && arg[2] == tt.kernel.Name {
						if tt.rpmError != nil {
							return nil, nil, tt.rpmExitCode, tt.rpmError
						}
						return nil, nil, tt.rpmExitCode, nil
					}
					return nil, nil, 1, errors.New("unexpected command")
				})
			}

			// Mock ReadFile for cmdline checks
			if tt.kernel != nil && tt.kernel.Append != "" {
				test.MockGlobal(t, &check.ReadFile, func(filename string) ([]byte, error) {
					if filename == "/proc/cmdline" {
						if tt.readFileError != nil {
							return nil, tt.readFileError
						}
						return tt.readFileData, nil
					}
					return nil, errors.New("file not found")
				})
			}

			chk, found := check.FindCheckByName("kernel")
			require.True(t, found, "Kernel Check not found")
			config := buildConfig(&blueprint.Customizations{
				Kernel: tt.kernel,
			})

			err := chk.Func(chk.Meta, config)
			if tt.wantError {
				require.Error(t, err)
				if tt.wantSkip {
					assert.True(t, check.IsSkip(err))
				}
				if tt.wantFail {
					assert.True(t, check.IsFail(err))
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}
