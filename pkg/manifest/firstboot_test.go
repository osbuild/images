package manifest

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/osbuild/images/pkg/customizations/firstboot"
	"github.com/osbuild/images/pkg/customizations/fsnode"
	"github.com/stretchr/testify/assert"
)

func concatFiles(files []*fsnode.File) string {
	var result strings.Builder
	result.WriteString("\n")

	for i, f := range files {
		result.WriteString("### " + f.Path() + " ###\n")
		result.Write(f.Data())
		if i < len(files)-1 {
			result.WriteString("\n\n")
		}
	}

	return result.String()
}

func TestFirstbootFileNodes(t *testing.T) {
	fbo := &firstboot.FirstbootOptions{
		Scripts: []firstboot.FirstbootOption{
			firstboot.SatelliteFirstbootOptions{
				FirstbootCommonOptions: firstboot.FirstbootCommonOptions{
					Name:          "satellite",
					IgnoreFailure: true,
				},
				CACerts: []string{"cert1", "cert2"},
				Command: "#!/usr/bin/bash\ncurl https://sat.example.com/register",
			},
			firstboot.AAPFirstbootOptions{
				FirstbootCommonOptions: firstboot.FirstbootCommonOptions{
					Name:          "aap",
					IgnoreFailure: true,
				},
				CACerts:        []string{"cert3", "cert4"},
				HostConfigKey:  "host-config-key",
				JobTemplateURL: "https://aap.example.com/api/v2/job_templates/9/callback/",
			},
			firstboot.CustomFirstbootOptions{
				Contents: "echo 'unnamed'",
			},
			firstboot.CustomFirstbootOptions{
				Contents: "echo 'unnamed'",
			},
			firstboot.CustomFirstbootOptions{
				FirstbootCommonOptions: firstboot.FirstbootCommonOptions{
					Name: "my-script",
				},
				Contents: "echo 'my-script'",
			},
			firstboot.CustomFirstbootOptions{
				FirstbootCommonOptions: firstboot.FirstbootCommonOptions{
					Name:          "ignore-errors",
					IgnoreFailure: true,
				},
				Contents: "echo 'ignore errors'",
			},
			firstboot.CustomFirstbootOptions{
				Contents: "echo 'unnamed'",
			},
		},
	}

	want := `
### /usr/local/bin/osbuild-first-satellite ###
#!/usr/bin/bash
curl https://sat.example.com/register

### /usr/local/bin/osbuild-first-aap ###
#!/usr/bin/bash
curl -s -i --data 'host_config_key=host-config-key' 'https://aap.example.com/api/v2/job_templates/9/callback/'

### /usr/local/bin/osbuild-first-custom-1 ###
echo 'unnamed'

### /usr/local/bin/osbuild-first-custom-2 ###
echo 'unnamed'

### /usr/local/bin/osbuild-first-my-script ###
echo 'my-script'

### /usr/local/bin/osbuild-first-ignore-errors ###
echo 'ignore errors'

### /usr/local/bin/osbuild-first-custom-3 ###
echo 'unnamed'`

	wantUnit := `
{
  "filename": "osbuild-custom-first-boot.service",
  "unit-type": "system",
  "unit-path": "usr",
  "config": {
    "Unit": {
      "ConditionPathExists": [
        "!/var/local/.osbuild-custom-first-boot-done"
      ],
      "Wants": [
        "network-online.target"
      ],
      "After": [
        "network-online.target",
        "osbuild-first-boot.service"
      ]
    },
    "Service": {
      "Type": "oneshot",
      "RemainAfterExit": true,
      "ExecStartPre": [
        "/usr/bin/touch /var/local/.osbuild-custom-first-boot-done"
      ],
      "ExecStart": [
        "-/usr/local/bin/osbuild-first-satellite",
        "-/usr/local/bin/osbuild-first-aap",
        "/usr/local/bin/osbuild-first-custom-1",
        "/usr/local/bin/osbuild-first-custom-2",
        "/usr/local/bin/osbuild-first-my-script",
        "-/usr/local/bin/osbuild-first-ignore-errors",
        "/usr/local/bin/osbuild-first-custom-3"
      ]
    },
    "Install": {
      "WantedBy": [
        "basic.target"
      ]
    }
  }
}`

	certs, files, unit, err := parse(fbo)
	assert.NoError(t, err)

	assert.Equal(t, []string{"cert1", "cert2", "cert3", "cert4"}, certs)

	got := concatFiles(files)
	assert.Equal(t, want, got)

	buf, err := json.MarshalIndent(unit, "", "  ")
	assert.NoError(t, err)
	assert.JSONEq(t, wantUnit, string(buf))
}
