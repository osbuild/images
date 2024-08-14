package distro

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/internal/types"
	"github.com/osbuild/images/pkg/osbuild"
)

func TestImageConfigInheritFrom(t *testing.T) {
	tests := []struct {
		name           string
		distroConfig   *ImageConfig
		imageConfig    *ImageConfig
		expectedConfig *ImageConfig
	}{
		{
			name: "inheritance with overridden values",
			distroConfig: &ImageConfig{
				Timezone: types.Some("America/New_York"),
				TimeSynchronization: &osbuild.ChronyStageOptions{
					Servers: []osbuild.ChronyConfigServer{{Hostname: "127.0.0.1"}},
				},
				Locale: common.ToPtr("en_US.UTF-8"),
				Keyboard: &osbuild.KeymapStageOptions{
					Keymap: "us",
				},
				EnabledServices:  []string{"sshd"},
				DisabledServices: []string{"named"},
				DefaultTarget:    common.ToPtr("multi-user.target"),
			},
			imageConfig: &ImageConfig{
				Timezone: types.Some("UTC"),
				TimeSynchronization: &osbuild.ChronyStageOptions{
					Servers: []osbuild.ChronyConfigServer{
						{
							Hostname: "169.254.169.123",
							Prefer:   common.ToPtr(true),
							Iburst:   common.ToPtr(true),
							Minpoll:  common.ToPtr(4),
							Maxpoll:  common.ToPtr(4),
						},
					},
					LeapsecTz: common.ToPtr(""),
				},
			},
			expectedConfig: &ImageConfig{
				Timezone: types.Some("UTC"),
				TimeSynchronization: &osbuild.ChronyStageOptions{
					Servers: []osbuild.ChronyConfigServer{
						{
							Hostname: "169.254.169.123",
							Prefer:   common.ToPtr(true),
							Iburst:   common.ToPtr(true),
							Minpoll:  common.ToPtr(4),
							Maxpoll:  common.ToPtr(4),
						},
					},
					LeapsecTz: common.ToPtr(""),
				},
				Locale: common.ToPtr("en_US.UTF-8"),
				Keyboard: &osbuild.KeymapStageOptions{
					Keymap: "us",
				},
				EnabledServices:  []string{"sshd"},
				DisabledServices: []string{"named"},
				DefaultTarget:    common.ToPtr("multi-user.target"),
			},
		},
		{
			name: "empty image type configuration",
			distroConfig: &ImageConfig{
				Timezone: types.Some("America/New_York"),
				TimeSynchronization: &osbuild.ChronyStageOptions{
					Servers: []osbuild.ChronyConfigServer{{Hostname: "127.0.0.1"}},
				},
				Locale: common.ToPtr("en_US.UTF-8"),
				Keyboard: &osbuild.KeymapStageOptions{
					Keymap: "us",
				},
				EnabledServices:  []string{"sshd"},
				DisabledServices: []string{"named"},
				DefaultTarget:    common.ToPtr("multi-user.target"),
			},
			imageConfig: &ImageConfig{},
			expectedConfig: &ImageConfig{
				Timezone: types.Some("America/New_York"),
				TimeSynchronization: &osbuild.ChronyStageOptions{
					Servers: []osbuild.ChronyConfigServer{{Hostname: "127.0.0.1"}},
				},
				Locale: common.ToPtr("en_US.UTF-8"),
				Keyboard: &osbuild.KeymapStageOptions{
					Keymap: "us",
				},
				EnabledServices:  []string{"sshd"},
				DisabledServices: []string{"named"},
				DefaultTarget:    common.ToPtr("multi-user.target"),
			},
		},
		{
			name:         "empty distro configuration",
			distroConfig: &ImageConfig{},
			imageConfig: &ImageConfig{
				Timezone: types.Some("America/New_York"),
				TimeSynchronization: &osbuild.ChronyStageOptions{
					Servers: []osbuild.ChronyConfigServer{{Hostname: "127.0.0.1"}},
				},
				Locale: common.ToPtr("en_US.UTF-8"),
				Keyboard: &osbuild.KeymapStageOptions{
					Keymap: "us",
				},
				EnabledServices:  []string{"sshd"},
				DisabledServices: []string{"named"},
				DefaultTarget:    common.ToPtr("multi-user.target"),
			},
			expectedConfig: &ImageConfig{
				Timezone: types.Some("America/New_York"),
				TimeSynchronization: &osbuild.ChronyStageOptions{
					Servers: []osbuild.ChronyConfigServer{{Hostname: "127.0.0.1"}},
				},
				Locale: common.ToPtr("en_US.UTF-8"),
				Keyboard: &osbuild.KeymapStageOptions{
					Keymap: "us",
				},
				EnabledServices:  []string{"sshd"},
				DisabledServices: []string{"named"},
				DefaultTarget:    common.ToPtr("multi-user.target"),
			},
		},
		{
			name:         "empty distro configuration",
			distroConfig: nil,
			imageConfig: &ImageConfig{
				Timezone: types.Some("America/New_York"),
				TimeSynchronization: &osbuild.ChronyStageOptions{
					Servers: []osbuild.ChronyConfigServer{{Hostname: "127.0.0.1"}},
				},
				Locale: common.ToPtr("en_US.UTF-8"),
				Keyboard: &osbuild.KeymapStageOptions{
					Keymap: "us",
				},
				EnabledServices:  []string{"sshd"},
				DisabledServices: []string{"named"},
				DefaultTarget:    common.ToPtr("multi-user.target"),
			},
			expectedConfig: &ImageConfig{
				Timezone: types.Some("America/New_York"),
				TimeSynchronization: &osbuild.ChronyStageOptions{
					Servers: []osbuild.ChronyConfigServer{{Hostname: "127.0.0.1"}},
				},
				Locale: common.ToPtr("en_US.UTF-8"),
				Keyboard: &osbuild.KeymapStageOptions{
					Keymap: "us",
				},
				EnabledServices:  []string{"sshd"},
				DisabledServices: []string{"named"},
				DefaultTarget:    common.ToPtr("multi-user.target"),
			},
		},
	}
	for idx, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.expectedConfig, tt.imageConfig.InheritFrom(tt.distroConfig), "test case %q failed (idx %d)", tt.name, idx)
		})
	}
}

func TestImageConfigDNFSetReleaseVerNotSet(t *testing.T) {
	var expected []*osbuild.DNFConfigStageOptions
	cnf := &ImageConfig{}
	assert.Equal(t, expected, cnf.DNFConfigOptions("9-stream"))

	cnf.DNFSetReleaseVerVar = common.ToPtr(false)
	assert.Equal(t, expected, cnf.DNFConfigOptions("9-stream"))
}

func TestImageConfigDNFConfigOptionsPreExisting(t *testing.T) {
	cnf := &ImageConfig{
		DNFConfig: []*osbuild.DNFConfigStageOptions{
			{
				Config: &osbuild.DNFConfig{
					Main: &osbuild.DNFConfigMain{
						IPResolve: "4",
					},
				},
			},
		},
	}
	assert.Equal(t, cnf.DNFConfig, cnf.DNFConfigOptions("9-stream"))

	cnf.DNFSetReleaseVerVar = common.ToPtr(true)
	assert.PanicsWithError(t, "internal error: currently DNFConfig and DNFSetReleaseVerVar cannot be used together, please reporting this as a feature request", func() {
		cnf.DNFConfigOptions("9-stream")
	})
}
