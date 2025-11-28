package main

import (
	"context"
	"testing"

	"github.com/osbuild/images/cmd/check-host-config/cos"
)

func TestRunningWait(t *testing.T) {
	responses := make(chan []byte, 2)
	responses <- []byte("starting\n")
	responses <- []byte("running\n")

	ctx := cos.WithExecFunc(context.Background(), func(name string, arg ...string) ([]byte, error) {
		return <-responses, nil // reading 3rd time will block and fail test
	})

	// XXX: use synctest.Run to speed it up after Go 1.24+ upgrade
	if err := runningWait(ctx, nil); err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
}

func TestGetActivatingUnits(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected string
	}{
		{
			name:     "single unit",
			output:   "foo.service loaded activating auto vendor preset: enabled\n",
			expected: "foo.service",
		},
		{
			name: "multiple units",
			output: `foo.service loaded activating auto vendor preset: enabled
bar.service loaded activating auto vendor preset: enabled
baz.service loaded activating auto vendor preset: enabled
`,
			expected: "foo.service bar.service baz.service",
		},
		{
			name:     "empty output",
			output:   "",
			expected: "",
		},
		{
			name:     "whitespace only",
			output:   "   \n  \n",
			expected: "",
		},
		{
			name:     "unit with spaces in name",
			output:   "foo-bar.service loaded activating auto vendor preset: enabled\n",
			expected: "foo-bar.service",
		},
		{
			name: "mixed with empty lines",
			output: `foo.service loaded activating auto vendor preset: enabled

bar.service loaded activating auto vendor preset: enabled
`,
			expected: "foo.service bar.service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := cos.WithExecFunc(context.Background(), func(name string, arg ...string) ([]byte, error) {
				if name == "systemctl" {
					return []byte(tt.output), nil
				}
				return nil, nil
			})

			result := getActivatingUnits(ctx)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
