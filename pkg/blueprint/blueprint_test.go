package blueprint

import (
	"encoding/json"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	assert.Equal(t, uint64(20*1000*1000*1000), bp.Customizations.Filesystem[1].MinSize)

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
	assert.Equal(t, uint64(20*1024*1024*1024), bp.Customizations.Filesystem[0].MinSize)
}

func TestGetPackagesInclude(t *testing.T) {

	bp := Blueprint{
		Name:        "packages-test",
		Description: "Testing GetPackagesInclude function",
		Version:     "0.0.1",
		PackagesInclude: []Package{
			{Name: "tmux", Version: "1.2"}},
		ModulesInclude: []Package{
			{Name: "openssh-server", Version: "*"}},
		GroupsInclude: []Group{
			{Name: "anaconda-tools"}},
	}
	Received_packages := bp.GetPackagesInclude()
	assert.ElementsMatch(t, []string{"tmux-1.2", "openssh-server", "@anaconda-tools", "kernel"}, Received_packages)
}

func TestKernelNameCustomization(t *testing.T) {
	kernels := []string{"kernel", "kernel-debug", "kernel-rt"}

	for _, k := range kernels {
		// kernel in customizations
		bp := Blueprint{
			Name:        "kernel-test",
			Description: "Testing GetPackagesInclude function with custom Kernel",
			Version:     "0.0.1",
			PackagesInclude: []Package{
				{Name: "tmux", Version: "1.2"}},
			ModulesInclude: []Package{
				{Name: "openssh-server", Version: "*"}},
			GroupsInclude: []Group{
				{Name: "anaconda-tools"}},
			Customizations: &Customizations{
				Kernel: &KernelCustomization{
					Name: k,
				},
			},
		}
		Received_packages := bp.GetPackagesInclude()
		assert.ElementsMatch(t, []string{"tmux-1.2", "openssh-server", "@anaconda-tools", k}, Received_packages)
	}

	for _, k := range kernels {
		// kernel in packages
		bp := Blueprint{
			Name:        "kernel-test",
			Description: "Testing GetPackagesInclude function with custom Kernel",
			Version:     "0.0.1",
			PackagesInclude: []Package{
				{Name: "tmux", Version: "1.2"},
				{Name: k},
			},
			ModulesInclude: []Package{
				{Name: "openssh-server", Version: "*"}},
			GroupsInclude: []Group{
				{Name: "anaconda-tools"}},
		}
		Received_packages := bp.GetPackagesInclude()

		// adds default kernel as well
		assert.ElementsMatch(t, []string{"tmux-1.2", k, "openssh-server", "@anaconda-tools", "kernel"}, Received_packages)
	}

	for _, bk := range kernels {
		for _, ck := range kernels {
			// all combos of both kernels
			bp := Blueprint{
				Name:        "kernel-test",
				Description: "Testing GetPackagesInclude function with custom Kernel",
				Version:     "0.0.1",
				PackagesInclude: []Package{
					{Name: "tmux", Version: "1.2"},
					{Name: bk},
				},
				ModulesInclude: []Package{
					{Name: "openssh-server", Version: "*"}},
				GroupsInclude: []Group{
					{Name: "anaconda-tools"}},
				Customizations: &Customizations{
					Kernel: &KernelCustomization{
						Name: ck,
					},
				},
			}
			Received_packages := bp.GetPackagesInclude()
			// both kernels are included, even if they're the same
			assert.ElementsMatch(t, []string{"tmux-1.2", bk, "openssh-server", "@anaconda-tools", ck}, Received_packages)
		}
	}
}
