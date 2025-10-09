package firstboot_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/osbuild/blueprint/pkg/blueprint"
	"github.com/osbuild/images/pkg/customizations/firstboot"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFirstbootOptionsFromBP(t *testing.T) {
	tests := []struct {
		name string
		json string
		want firstboot.FirstbootOptions
		err  string
	}{
		{
			name: "custom",
			json: `{"scripts": [{"type":"custom","name":"custom","contents":"echo hello"}]}`,
			want: firstboot.FirstbootOptions{
				Scripts: []firstboot.FirstbootOption{
					firstboot.CustomFirstbootOptions{
						FirstbootCommonOptions: firstboot.FirstbootCommonOptions{
							Name:          "custom",
							IgnoreFailure: false,
						},
						Contents: "echo hello",
					},
				},
			},
		},
		{
			name: "satellite",
			json: `{"scripts": [{"type":"satellite","name":"satellite","command":"echo hello"}]}`,
			want: firstboot.FirstbootOptions{
				Scripts: []firstboot.FirstbootOption{
					firstboot.SatelliteFirstbootOptions{
						FirstbootCommonOptions: firstboot.FirstbootCommonOptions{
							Name:          "satellite",
							IgnoreFailure: false,
						},
						CACerts: nil,
						Command: "echo hello",
					},
				},
			},
		},
		{
			name: "aap",
			json: `{"scripts": [{"type":"aap","name":"aap","host_config_key":"key","job_template_url":"https://aap.example.com/api/v2/job_templates/9/callback/"}]}`,
			want: firstboot.FirstbootOptions{
				Scripts: []firstboot.FirstbootOption{
					firstboot.AAPFirstbootOptions{
						FirstbootCommonOptions: firstboot.FirstbootCommonOptions{
							Name:          "aap",
							IgnoreFailure: false,
						},
						CACerts:        nil,
						JobTemplateURL: "https://aap.example.com/api/v2/job_templates/9/callback/",
						HostConfigKey:  "key",
					},
				},
			},
		},
		{
			name: "sat-sat",
			json: `{"scripts": [{"type":"satellite","name":"sat","command":"echo hello"},{"type":"satellite","name":"sat","command":"echo hello"}]}`,
			err:  "firstboot customization already set: satellite",
		},
		{
			name: "aap-aap",
			json: `{"scripts": [{"type":"aap","name":"aap","host_config_key":"key","job_template_url":"https://aap.example.com/api/v2/job_templates/9/callback/"},{"type":"aap","name":"aap","host_config_key":"key","job_template_url":"https://aap.example.com/api/v2/job_templates/9/callback/"}]}`,
			err:  "firstboot customization already set: aap",
		},
		{
			name: "sat-c1-c2-aap",
			json: `{"scripts": [
				{"type":"satellite","name":"sat","command":"echo hello"},
				{"type":"custom","name":"c1","contents":"echo hello"},
				{"type":"custom","name":"c2","contents":"echo hello"},
				{"type":"aap","name":"aap","host_config_key":"key","job_template_url":"https://aap.example.com/api/v2/job_templates/9/callback/"}
			]}`,
			want: firstboot.FirstbootOptions{
				Scripts: []firstboot.FirstbootOption{
					firstboot.SatelliteFirstbootOptions{
						FirstbootCommonOptions: firstboot.FirstbootCommonOptions{
							Name:          "sat",
							IgnoreFailure: false,
						},
						CACerts: nil,
						Command: "echo hello",
					},
					firstboot.CustomFirstbootOptions{
						FirstbootCommonOptions: firstboot.FirstbootCommonOptions{
							Name:          "c1",
							IgnoreFailure: false,
						},
						Contents: "echo hello",
					},
					firstboot.CustomFirstbootOptions{
						FirstbootCommonOptions: firstboot.FirstbootCommonOptions{
							Name:          "c2",
							IgnoreFailure: false,
						},
						Contents: "echo hello",
					},
					firstboot.AAPFirstbootOptions{
						FirstbootCommonOptions: firstboot.FirstbootCommonOptions{
							Name:          "aap",
							IgnoreFailure: false,
						},
						CACerts:        nil,
						JobTemplateURL: "https://aap.example.com/api/v2/job_templates/9/callback/",
						HostConfigKey:  "key",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var input blueprint.FirstbootCustomization
			err := json.Unmarshal([]byte(tt.json), &input)
			assert.NoError(t, err)

			got, err := firstboot.FirstbootOptionsFromBP(input)
			if tt.err != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)

			var names []string
			got.Each(func(c firstboot.CustomFirstbootOptions) error {
				names = append(names, c.Name)
				return nil
			}, func(s firstboot.SatelliteFirstbootOptions) error {
				names = append(names, s.Name)
				return nil
			}, func(a firstboot.AAPFirstbootOptions) error {
				names = append(names, a.Name)
				return nil
			})
			assert.Equal(t, tt.name, strings.Join(names, "-"))

			assert.Equal(t, tt.want, *got)
		})
	}
}
