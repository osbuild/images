package osbuild

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/osbuild/images/pkg/customizations/firstboot"
	"github.com/osbuild/images/pkg/customizations/fsnode"
	"github.com/stretchr/testify/assert"
)

func concatFirstbootFiles(files []*fsnode.File) string {
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

func TestGenFirstbootFromOptions(t *testing.T) {
	fbo := &firstboot.FirstbootOptions{
		Scripts: []firstboot.Script{
			{
				Filename:      "osbuild-first-satellite",
				Contents:      "#!/usr/bin/bash\ncurl https://sat.example.com/register",
				IgnoreFailure: true,
				Certs:         []string{"cert1", "cert2"},
			},
			{
				Filename:      "osbuild-first-aap",
				Contents:      "#!/usr/bin/bash\ncurl -i --data 'host_config_key=host-config-key' 'https://aap.example.com/api/v2/job_templates/9/callback/'\n",
				IgnoreFailure: true,
				Certs:         []string{"cert3", "cert4"},
			},
			{
				Filename: "osbuild-first-custom-1",
				Contents: "echo 'unnamed'",
			},
			{
				Filename: "osbuild-first-custom-2",
				Contents: "echo 'unnamed'",
			},
			{
				Filename: "osbuild-first-my-script",
				Contents: "echo 'my-script'",
			},
			{
				Filename:      "osbuild-first-ignore-errors",
				Contents:      "echo 'ignore errors'",
				IgnoreFailure: true,
			},
			{
				Filename: "osbuild-first-custom-3",
				Contents: "echo 'unnamed'",
			},
		},
	}

	want := `
### /var/local/.osbuild-custom-first-boot ###


### /usr/local/bin/osbuild-first-satellite ###
#!/usr/bin/bash
curl https://sat.example.com/register

### /usr/local/bin/osbuild-first-aap ###
#!/usr/bin/bash
curl -i --data 'host_config_key=host-config-key' 'https://aap.example.com/api/v2/job_templates/9/callback/'


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
        "/var/local/.osbuild-custom-first-boot"
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
        "/usr/bin/rm /var/local/.osbuild-custom-first-boot"
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

	certs, files, unit, err := GenFirstbootFromOptions(fbo)
	assert.NoError(t, err)

	assert.Equal(t, []string{"cert1", "cert2", "cert3", "cert4"}, certs)

	got := concatFirstbootFiles(files)
	assert.Equal(t, want, got)

	buf, err := json.MarshalIndent(unit, "", "  ")
	assert.NoError(t, err)
	assert.JSONEq(t, wantUnit, string(buf))
}
