package blueprint

import (
	"encoding/json"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osbuild/images/internal/common"
)

func TestBlueprintParse(t *testing.T) {
	blueprint := `
name = "test"
description = "Test"
version = "0.0.0"

[[packages]]
name = "httpd"
version = "2.4.*"

[[customizations.filesystem]]
mountpoint = "/var"
size = 2147483648

[[customizations.filesystem]]
mountpoint = "/opt"
size = "20 GB"
`

	var bp Blueprint
	err := toml.Unmarshal([]byte(blueprint), &bp)
	require.Nil(t, err)
	assert.Equal(t, bp.Name, "test")
	assert.Equal(t, "/var", bp.Customizations.Filesystem[0].Mountpoint)
	assert.Equal(t, uint64(2147483648), bp.Customizations.Filesystem[0].MinSize)
	assert.Equal(t, "/opt", bp.Customizations.Filesystem[1].Mountpoint)
	assert.Equal(t, uint64(20*common.GB), bp.Customizations.Filesystem[1].MinSize)

	blueprint = `{
		"name": "test",
		"customizations": {
		  "filesystem": [{
			"mountpoint": "/opt",
			"minsize": "20 GiB"
		  }]
		}
	  }`
	err = json.Unmarshal([]byte(blueprint), &bp)
	require.Nil(t, err)
	assert.Equal(t, bp.Name, "test")
	assert.Equal(t, "/opt", bp.Customizations.Filesystem[0].Mountpoint)
	assert.Equal(t, uint64(20*common.GiB), bp.Customizations.Filesystem[0].MinSize)
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
