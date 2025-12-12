package mockos

import (
	"context"
	"testing"
)

func TestExistsContextWithMock(t *testing.T) {
	ctx := WithExistsFunc(context.Background(), func(name string) bool {
		return name == "/some/existing/path"
	})

	exists := ExistsContext(ctx, nil, "/some/existing/path")
	if !exists {
		t.Errorf("ExistsContext() = false, want true for mocked existing file")
	}

	exists = ExistsContext(ctx, nil, "/some/nonexistent/path")
	if exists {
		t.Errorf("ExistsContext() = true, want false for mocked nonexistent file")
	}
}
