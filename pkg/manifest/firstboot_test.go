package manifest

import (
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
		Satellite: &firstboot.SatelliteFirstbootOptions{
			Command: "curl https://sat.example.com/register",
			CACerts: []string{"cert1"},
		},
		AAP: &firstboot.AAPFirstbootOptions{
			JobTemplateURL: "https://aap.example.com/api/v2/job_templates/9/callback/",
			HostConfigKey:  "host-config-key",
			CACerts:        []string{"cert2"},
		},
		Custom: []firstboot.CustomFirstbootOptions{
			{
				Contents: "#!/usr/bin/bash\necho 'Unnamed'",
			},
			{
				Contents:      "#!/usr/bin/bash\necho 'Do not ignore errors'",
				Name:          "no-ignore-errors",
				IgnoreFailure: false,
			},
			{
				Contents:      "#!/usr/bin/bash\necho 'Ignore errors'",
				Name:          "ignore-errors",
				IgnoreFailure: true,
			},
		},
	}

	want := `
### /usr/local/sbin/osbuild-first-satellite ###
curl https://sat.example.com/register

### /usr/local/sbin/osbuild-first-aap ###
#!/usr/bin/bash
curl -s -i --data "host_config_key=host-config-key" https://aap.example.com/api/v2/job_templates/9/callback/

### /usr/local/sbin/osbuild-first-custom-1 ###
#!/usr/bin/bash
echo 'Unnamed'

### /usr/local/sbin/osbuild-first-no-ignore-errors ###
#!/usr/bin/bash
echo 'Do not ignore errors'

### /usr/local/sbin/osbuild-first-ignore-errors ###
#!/usr/bin/bash
echo 'Ignore errors'

### /etc/systemd/system/osbuild-first-boot.service ###
[Unit]
ConditionPathExists=!/var/local/.osbuild-custom-first-boot-done
Wants=network-online.target
After=network-online.target
After=osbuild-first-boot.service

[Service]
Type=oneshot

ExecStart=-/usr/local/sbin/osbuild-first-satellite
ExecStart=-/usr/local/sbin/osbuild-first-aap
ExecStart=/usr/local/sbin/osbuild-first-custom-1
ExecStart=/usr/local/sbin/osbuild-first-no-ignore-errors
ExecStart=-/usr/local/sbin/osbuild-first-ignore-errors
ExecStartPost=/usr/bin/touch /var/local/.osbuild-custom-first-boot-done
RemainAfterExit=yes

[Install]
WantedBy=basic.target
`
	certs, files, err := firstbootFileNodes(fbo)
	assert.NoError(t, err)

	assert.Equal(t, []string{"cert1", "cert2"}, certs)

	got := concatFiles(files)
	assert.Equal(t, want, got)
}
