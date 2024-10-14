package blueprint

import (
	"encoding/json"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osbuild/images/pkg/datasizes"
)

func TestBlueprintParse(t *testing.T) {
	blueprintToml := `
name = "test"
description = "Test description"
version = "0.0.0"

[[packages]]
name = "httpd"
version = "2.4.*"

[[customizations.filesystem]]
mountpoint = "/var"
minsize = 2147483648

[[customizations.filesystem]]
mountpoint = "/opt"
minsize = "20 GiB"
`
	blueprintJSON := `{
		"name": "test",
                "description": "Test description",
                "version": "0.0.0",
                "packages": [
                  {
                    "name": "httpd",
                    "version": "2.4.*"
                  }
                ],
		"customizations": {
		  "filesystem": [
                    {
			"mountpoint": "/var",
			"minsize": 2147483648
		    },
                    {
			"mountpoint": "/opt",
			"minsize": "20 GiB"
                    }
                  ]
		}
	  }`

	var bp, bp2 Blueprint
	err := toml.Unmarshal([]byte(blueprintToml), &bp)
	require.NoError(t, err)
	err = json.Unmarshal([]byte(blueprintJSON), &bp2)
	require.NoError(t, err)
	require.Equal(t, bp, bp2)

	assert.Equal(t, bp.Name, "test")
	assert.Equal(t, bp.Description, "Test description")
	assert.Equal(t, bp.Version, "0.0.0")
	assert.Equal(t, bp.Packages, []Package{{Name: "httpd", Version: "2.4.*"}})
	assert.Equal(t, "/var", bp.Customizations.Filesystem[0].Mountpoint)
	assert.Equal(t, uint64(2147483648), bp.Customizations.Filesystem[0].MinSize)
	assert.Equal(t, "/opt", bp.Customizations.Filesystem[1].Mountpoint)
	assert.Equal(t, uint64(20*datasizes.GiB), bp.Customizations.Filesystem[1].MinSize)
}

func TestGetPackages(t *testing.T) {

	bp := Blueprint{
		Name:        "packages-test",
		Description: "Testing GetPackages function",
		Version:     "0.0.1",
		Packages: []Package{
			{Name: "tmux", Version: "1.2"}},
		Modules: []Package{
			{Name: "openssh-server", Version: "*"}},
		Groups: []Group{
			{Name: "anaconda-tools"}},
	}
	Received_packages := bp.GetPackages()
	assert.ElementsMatch(t, []string{"tmux-1.2", "openssh-server", "@anaconda-tools", "kernel"}, Received_packages)
}

func TestKernelNameCustomization(t *testing.T) {
	kernels := []string{"kernel", "kernel-debug", "kernel-rt"}

	for _, k := range kernels {
		// kernel in customizations
		bp := Blueprint{
			Name:        "kernel-test",
			Description: "Testing GetPackages function with custom Kernel",
			Version:     "0.0.1",
			Packages: []Package{
				{Name: "tmux", Version: "1.2"}},
			Modules: []Package{
				{Name: "openssh-server", Version: "*"}},
			Groups: []Group{
				{Name: "anaconda-tools"}},
			Customizations: &Customizations{
				Kernel: &KernelCustomization{
					Name: k,
				},
			},
		}
		Received_packages := bp.GetPackages()
		assert.ElementsMatch(t, []string{"tmux-1.2", "openssh-server", "@anaconda-tools", k}, Received_packages)
	}

	for _, k := range kernels {
		// kernel in packages
		bp := Blueprint{
			Name:        "kernel-test",
			Description: "Testing GetPackages function with custom Kernel",
			Version:     "0.0.1",
			Packages: []Package{
				{Name: "tmux", Version: "1.2"},
				{Name: k},
			},
			Modules: []Package{
				{Name: "openssh-server", Version: "*"}},
			Groups: []Group{
				{Name: "anaconda-tools"}},
		}
		Received_packages := bp.GetPackages()

		// adds default kernel as well
		assert.ElementsMatch(t, []string{"tmux-1.2", k, "openssh-server", "@anaconda-tools", "kernel"}, Received_packages)
	}

	for _, bk := range kernels {
		for _, ck := range kernels {
			// all combos of both kernels
			bp := Blueprint{
				Name:        "kernel-test",
				Description: "Testing GetPackages function with custom Kernel",
				Version:     "0.0.1",
				Packages: []Package{
					{Name: "tmux", Version: "1.2"},
					{Name: bk},
				},
				Modules: []Package{
					{Name: "openssh-server", Version: "*"}},
				Groups: []Group{
					{Name: "anaconda-tools"}},
				Customizations: &Customizations{
					Kernel: &KernelCustomization{
						Name: ck,
					},
				},
			}
			Received_packages := bp.GetPackages()
			// both kernels are included, even if they're the same
			assert.ElementsMatch(t, []string{"tmux-1.2", bk, "openssh-server", "@anaconda-tools", ck}, Received_packages)
		}
	}
}
