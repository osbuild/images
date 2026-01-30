package firstboot_test

import (
	"encoding/json"
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
				Scripts: []firstboot.Script{
					{
						Filename:      "osbuild-first-custom",
						Contents:      "echo hello",
						IgnoreFailure: false,
					},
				},
			},
		},
		{
			name: "satellite",
			json: `{"scripts": [{"type":"satellite","name":"satellite","command":"echo hello"}]}`,
			want: firstboot.FirstbootOptions{
				Scripts: []firstboot.Script{
					{
						Filename:      "osbuild-first-satellite",
						Contents:      "echo hello",
						IgnoreFailure: false,
						Certs:         nil,
					},
				},
			},
		},
		{
			name: "aap",
			json: `{"scripts": [{"type":"aap","name":"aap","host_config_key":"key","job_template_url":"https://aap.example.com/api/v2/job_templates/9/callback/"}]}`,
			want: firstboot.FirstbootOptions{
				Scripts: []firstboot.Script{
					{
						Filename:      "osbuild-first-aap",
						Contents:      "#!/usr/bin/bash\ncurl -i --data 'host_config_key=key' 'https://aap.example.com/api/v2/job_templates/9/callback/'\n",
						IgnoreFailure: false,
						Certs:         nil,
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
				{"type":"custom","name":"c1","contents":"echo hello c1"},
				{"type":"custom","name":"c2","contents":"echo hello c2"},
				{"type":"aap","name":"aap","host_config_key":"key","job_template_url":"https://aap.example.com/api/v2/job_templates/9/callback/"}
			]}`,
			want: firstboot.FirstbootOptions{
				Scripts: []firstboot.Script{
					{
						Filename:      "osbuild-first-sat",
						Contents:      "echo hello",
						IgnoreFailure: false,
						Certs:         nil,
					},
					{
						Filename:      "osbuild-first-c1",
						Contents:      "echo hello c1",
						IgnoreFailure: false,
					},
					{
						Filename:      "osbuild-first-c2",
						Contents:      "echo hello c2",
						IgnoreFailure: false,
					},
					{
						Filename:      "osbuild-first-aap",
						Contents:      "#!/usr/bin/bash\ncurl -i --data 'host_config_key=key' 'https://aap.example.com/api/v2/job_templates/9/callback/'\n",
						IgnoreFailure: false,
						Certs:         nil,
					},
				},
			},
		},
		{
			name: "path-traversal",
			json: `{"scripts": [
				{"type":"custom","name":"../bad","contents":"echo bad"},
				{"type":"custom","name":"/absolute/bad","contents":"echo bad"},
				{"type":"custom","name":"good","contents":"echo good"}
			]}`,
			want: firstboot.FirstbootOptions{
				Scripts: []firstboot.Script{
					{
						Filename:      "osbuild-first-custom-1",
						Contents:      "echo bad",
						IgnoreFailure: false,
					},
					{
						Filename:      "osbuild-first-custom-2",
						Contents:      "echo bad",
						IgnoreFailure: false,
					},
					{
						Filename:      "osbuild-first-good",
						Contents:      "echo good",
						IgnoreFailure: false,
					},
				},
			},
		},
		{
			name: "duplicate-name",
			json: `{"scripts": [
				{"type":"custom","name":"test","contents":"echo test"},
				{"type":"custom","name":"test","contents":"echo test"}
			]}`,
			want: firstboot.FirstbootOptions{
				Scripts: []firstboot.Script{
					{
						Filename: "osbuild-first-test",
						Contents: "echo test",
					},
					{
						Filename: "osbuild-first-custom-1",
						Contents: "echo test",
					},
				},
			},
		},
		{
			name: "reserved-name",
			json: `{"scripts": [
				{"type":"custom","name":"custom-42","contents":"echo test"},
				{"type":"custom","name":"aap-13","contents":"echo test"}
			]}`,
			want: firstboot.FirstbootOptions{
				Scripts: []firstboot.Script{
					{
						Filename: "osbuild-first-custom-1",
						Contents: "echo test",
					},
					{
						Filename: "osbuild-first-custom-2",
						Contents: "echo test",
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

			assert.Equal(t, tt.want, *got)
		})
	}
}
