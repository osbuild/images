package check_test

import (
	"errors"
	"testing"

	"github.com/osbuild/blueprint/pkg/blueprint"
	check "github.com/osbuild/images/cmd/check-host-config/check"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// dnf module list --enabled output fixtures for different RHEL versions
func dnfModuleListOutputRHEL7() string {
	return "Last metadata expiration check: 1:23:45 ago on Mon 01 Jan 2024 12:00:00 PM UTC.\n" +
		"Dependencies resolved.\n" +
		"Module Stream Profiles\n" +
		"nodejs           10        [e]       common [d], development, minimal\n" +
		"python36         3.6       [e]       build, common [d], devel\n" +
		"Hint: [d]efault, [e]nabled, [x]disabled, [i]nstalled, [a]ctive\n"
}

func dnfModuleListOutputRHEL8() string {
	return "Last metadata expiration check: 0:00:00 ago on Mon 01 Jan 2024 12:00:00 PM UTC.\n" +
		"Dependencies resolved.\n" +
		"Module Stream Profiles\n" +
		"nodejs           12        [e]       common [d], development, minimal, s2i\n" +
		"python38         3.8       [e]       build, common [d], devel, minimal\n" +
		"postgresql       12        [e]       client, server [d]\n" +
		"Hint: [d]efault, [e]nabled, [x]disabled, [i]nstalled, [a]ctive\n"
}

func dnfModuleListOutputRHEL9() string {
	return "Last metadata expiration check: 0:00:00 ago\n" +
		"Dependencies resolved.\n" +
		"Module Stream Profiles\n" +
		"nodejs           18        [d]       common [d], development, minimal, s2i\n" +
		"python39         3.9       [d]       build, common [d], devel, minimal\n" +
		"Hint: [d]efault, [e]nabled, [x]disabled, [i]nstalled, [a]ctive\n"
}

func dnfModuleListOutputRHEL10() string {
	return "Last metadata expiration check: 0:00:00 ago\n" +
		"Dependencies resolved.\n" +
		"Module Stream Profiles\n" +
		"nodejs           20        [e]       common [d], development, minimal, s2i\n" +
		"python312        3.12      [e]       build, common [d], devel, minimal\n" +
		"postgresql       16        [e]       client, server [d], devel\n" +
		"Use \"dnf module info <module:stream>\" to get more information.\n" +
		"Hint: [d]efault, [e]nabled, [x]disabled, [i]nstalled, [a]ctive\n"
}

func dnfModuleListOutputCentOS9() string {
	return `CentOS Stream 9 - AppStream
Name      Stream    Profiles                                Summary             
nodejs    18 [e]    common [d], development, minimal, s2i   Javascript runtime  

Hint: [d]efault, [e]nabled, [x]disabled, [i]nstalled
`
}

func dnfModuleListOutputMultiple() string {
	return "Last metadata expiration check: 0:00:00 ago\n" +
		"Dependencies resolved.\n" +
		"Module Stream Profiles\n" +
		"nodejs           18        [e]       common [d], development, minimal, s2i\n" +
		"python39         3.9       [e]       build, common [d], devel, minimal\n" +
		"postgresql       13        [e]       client, server [d]\n" +
		"ruby              3.1       [e]       common [d], devel\n" +
		"Hint: [d]efault, [e]nabled, [x]disabled, [i]nstalled, [a]ctive\n"
}

func TestModularityCheck(t *testing.T) {
	tests := []struct {
		name     string
		config   []blueprint.EnabledModule
		mockExec map[string]ExecResult
		wantErr  error
	}{
		{
			name:    "skip when no modules",
			config:  []blueprint.EnabledModule{},
			wantErr: check.ErrCheckSkipped,
		},
		{
			name: "pass with single module (RHEL 9 style)",
			config: []blueprint.EnabledModule{
				{Name: "nodejs", Stream: "18"},
			},
			mockExec: map[string]ExecResult{
				"dnf -y -q module list --enabled": {Stdout: []byte(dnfModuleListOutputRHEL9())},
			},
		},
		{
			name: "pass RHEL 7 format",
			config: []blueprint.EnabledModule{
				{Name: "nodejs", Stream: "10"},
				{Name: "python36", Stream: "3.6"},
			},
			mockExec: map[string]ExecResult{
				"dnf -y -q module list --enabled": {Stdout: []byte(dnfModuleListOutputRHEL7())},
			},
		},
		{
			name: "pass RHEL 8 format",
			config: []blueprint.EnabledModule{
				{Name: "nodejs", Stream: "12"},
				{Name: "python38", Stream: "3.8"},
				{Name: "postgresql", Stream: "12"},
			},
			mockExec: map[string]ExecResult{
				"dnf -y -q module list --enabled": {Stdout: []byte(dnfModuleListOutputRHEL8())},
			},
		},
		{
			name: "pass RHEL 9 format",
			config: []blueprint.EnabledModule{
				{Name: "nodejs", Stream: "18"},
				{Name: "python39", Stream: "3.9"},
			},
			mockExec: map[string]ExecResult{
				"dnf -y -q module list --enabled": {Stdout: []byte(dnfModuleListOutputRHEL9())},
			},
		},
		{
			name: "pass RHEL 10 format",
			config: []blueprint.EnabledModule{
				{Name: "nodejs", Stream: "20"},
				{Name: "python312", Stream: "3.12"},
				{Name: "postgresql", Stream: "16"},
			},
			mockExec: map[string]ExecResult{
				"dnf -y -q module list --enabled": {Stdout: []byte(dnfModuleListOutputRHEL10())},
			},
		},
		{
			name: "pass CentOS 9 format",
			config: []blueprint.EnabledModule{
				{Name: "nodejs", Stream: "18"},
			},
			mockExec: map[string]ExecResult{
				"dnf -y -q module list --enabled": {Stdout: []byte(dnfModuleListOutputCentOS9())},
			},
		},
		{
			name: "pass multiple modules",
			config: []blueprint.EnabledModule{
				{Name: "nodejs", Stream: "18"},
				{Name: "python39", Stream: "3.9"},
				{Name: "postgresql", Stream: "13"},
				{Name: "ruby", Stream: "3.1"},
			},
			mockExec: map[string]ExecResult{
				"dnf -y -q module list --enabled": {Stdout: []byte(dnfModuleListOutputMultiple())},
			},
		},
		{
			name: "fail when dnf errors",
			config: []blueprint.EnabledModule{
				{Name: "nodejs", Stream: "18"},
			},
			mockExec: map[string]ExecResult{
				"dnf -y -q module list --enabled": {Code: 1, Err: errors.New("dnf failed")},
			},
			wantErr: check.ErrCheckFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			installMockExec(t, tt.mockExec)

			chk, found := check.FindCheckByName("modularity")
			require.True(t, found, "modularity check not found")
			config := buildConfigWithBlueprint(func(bp *blueprint.Blueprint) {
				bp.EnabledModules = tt.config
				if len(tt.config) == 0 {
					bp.Packages = []blueprint.Package{}
				}
			})

			err := chk.Func(chk.Meta, config)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.True(t, errors.Is(err, tt.wantErr))
			} else {
				require.NoError(t, err)
			}
		})
	}
}
