package mockos

import (
	"context"
	"errors"
	"testing"
)

func TestGrepContextWithMock(t *testing.T) {
	ctx := WithGrepFunc(context.Background(), func(pattern, filename string) (bool, error) {
		return pattern == "mocked-pattern", nil
	})

	found, err := GrepContext(ctx, nil, "mocked-pattern", "/some/file")
	if err != nil {
		t.Fatalf("GrepContext failed: %v", err)
	}

	if !found {
		t.Errorf("GrepContext() = false, want true for mocked pattern")
	}

	found, err = GrepContext(ctx, nil, "other-pattern", "/some/file")
	if err != nil {
		t.Fatalf("GrepContext failed: %v", err)
	}

	if found {
		t.Errorf("GrepContext() = true, want false for non-matching pattern")
	}
}

func TestGrepContextWithMockError(t *testing.T) {
	expectedError := errors.New("mocked file error")
	ctx := WithGrepFunc(context.Background(), func(pattern, filename string) (bool, error) {
		return false, expectedError
	})

	_, err := GrepContext(ctx, nil, "pattern", "/some/file")
	if err == nil {
		t.Error("GrepContext() expected error, got nil")
	}
	if err != expectedError {
		t.Errorf("GrepContext() error = %v, want %v", err, expectedError)
	}
}

func TestGrepContextWithMockPatternNotFound(t *testing.T) {
	ctx := WithGrepFunc(context.Background(), func(pattern, filename string) (bool, error) {
		return false, nil
	})

	found, err := GrepContext(ctx, nil, "nonexistent-pattern", "/some/file")
	if err != nil {
		t.Fatalf("GrepContext failed: %v", err)
	}

	if found {
		t.Errorf("GrepContext() = true, want false for nonexistent pattern")
	}
}
