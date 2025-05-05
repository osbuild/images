package osbuild

import (
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUdevRulesStage(t *testing.T) {
	stage := NewUdevRulesStage(
		&UdevRulesStageOptions{
			Filename: "/etc/udev/udev.rules",
			Rules: UdevRules{
				NewUdevRuleComment([]string{"This is a comment"}),
				NewUdevRule(
					[]UdevKV{
						{K: "ACTION", O: "==", V: "add"},
						{K: "ENV", A: "OSBUILD", O: "=", V: "1"},
					},
				),
			},
		},
	)

	want := &Stage{
		Type: "org.osbuild.udev.rules",
		Options: &UdevRulesStageOptions{
			Filename: "/etc/udev/udev.rules",
			Rules: UdevRules{
				UdevRuleComment{
					Comment: []string{"This is a comment"},
				},
				UdevOps{
					UdevOpSimple{
						Key:   "ACTION",
						Op:    "==",
						Value: "add",
					},
					UdevOpArg{
						Key: UdevRuleKeyArg{
							Name: "ENV",
							Arg:  "OSBUILD",
						},
						Op:    "=",
						Value: "1",
					},
				},
			},
		},
	}

	assert.Equal(t, stage, want)
}

func TestNewUdevRulesStageValidate(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		name     string
		filename string
		rules    []UdevRule
	}{
		{
			name: "no filename",
			rules: UdevRules{
				UdevRuleComment{
					Comment: []string{
						"This is a comment",
					},
				},
			},
		},
		{
			name:     "wrong filename",
			filename: "/etc/udev/udev.conf",
			rules: UdevRules{
				UdevRuleComment{
					Comment: []string{
						"This is a comment",
					},
				},
			},
		},
		{
			name:     "missing rules",
			filename: "/etc/udev/rules.d/osbuild.rules",
		},
		{
			name:     "empty rules",
			filename: "/etc/udev/rules.d/osbuild.rules",
			rules:    UdevRules{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Panics(func() {
				NewUdevRulesStage(&UdevRulesStageOptions{
					Filename: tt.filename,
					Rules:    tt.rules,
				})
			})
		})
	}
}

func TestUdevRuleValidation(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		name string
		rule UdevKV
	}{
		{
			name: "no key",
			rule: UdevKV{
				O: "==",
				V: "add",
			},
		},
		{
			name: "no op",
			rule: UdevKV{
				K: "ACTION",
				V: "add",
			},
		},
		{
			name: "no value",
			rule: UdevKV{
				K: "ACTION",
				O: "==",
			},
		},
		{
			name: "unknown key",
			rule: UdevKV{
				K: "ACHILLEAS",
				O: "==",
				V: "RE GOMBARE",
			},
		},
		{
			name: "missing arg",
			rule: UdevKV{
				K: "ENV",
				O: "==",
				V: "RE GOMBARE",
			},
		},
		{
			name: "unknown op",
			rule: UdevKV{
				K: "ENV",
				O: "?",
				V: "RE GOMBARE",
			},
		},
		{
			name: "false assign",
			rule: UdevKV{
				K: "ACTION",
				O: "=",
				V: "RE GOMBARE",
			},
		},
		{
			name: "false match",
			rule: UdevKV{
				K: "OPTIONS",
				O: "==",
				V: "RE GOMBARE",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Panics(func() {
				NewUdevRule([]UdevKV{tt.rule})
			})
		})
	}
}

func TestUdevRuleUnmarshal(t *testing.T) {
	inputYAML := []byte(`
filename: "/etc/udev/rules.d/68-azure-sriov-nm-unmanaged.rules"
rules:
  - comment:
      - "comment1"
      - "comment2"
  - rule:
    - K: "SUBSYSTEM"
      O: "=="
      V: "net"
    - K: "DRIVERS"
      O: "=="
      V: "hv_pci"
    - K: "ACTION"
      O: "=="
      V: "add"
    - K: "ENV"
      A: "NM_UNMANAGED"
      O: "="
      V: "1"
`)
	var options UdevRulesStageOptions
	err := yaml.Unmarshal(inputYAML, &options)
	require.NoError(t, err)
	expected := UdevRulesStageOptions{
		Filename: "/etc/udev/rules.d/68-azure-sriov-nm-unmanaged.rules",
		Rules: UdevRules{
			UdevRuleComment{
				Comment: []string{"comment1", "comment2"},
			},
			UdevOps{
				UdevOpSimple{
					Key:   "SUBSYSTEM",
					Op:    "==",
					Value: "net",
				},
				UdevOpSimple{
					Key:   "DRIVERS",
					Op:    "==",
					Value: "hv_pci",
				},
				UdevOpSimple{
					Key:   "ACTION",
					Op:    "==",
					Value: "add",
				},
				UdevOpArg{
					Key: UdevRuleKeyArg{
						Name: "ENV",
						Arg:  "NM_UNMANAGED",
					},
					Op:    "=",
					Value: "1",
				},
			},
		},
	}
	assert.Equal(t, expected, options)
}
