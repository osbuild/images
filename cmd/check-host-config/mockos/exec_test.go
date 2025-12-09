package mockos

import (
	"context"
	"errors"
	"testing"
)

func TestExecContextWithMock(t *testing.T) {
	expectedStdout := []byte("mocked stdout")
	expectedStderr := []byte("mocked stderr")
	ctx := WithExecFunc(context.Background(), func(name string, arg ...string) ([]byte, []byte, error) {
		return expectedStdout, expectedStderr, nil
	})

	stdout, stderr, err := ExecContext(ctx, nil, "some-command", "arg1", "arg2")
	if err != nil {
		t.Fatalf("ExecContext failed: %v", err)
	}

	if string(stdout) != string(expectedStdout) {
		t.Errorf("ExecContext() stdout = %q, want %q", string(stdout), string(expectedStdout))
	}

	if string(stderr) != string(expectedStderr) {
		t.Errorf("ExecContext() stderr = %q, want %q", string(stderr), string(expectedStderr))
	}
}

func TestExecContextWithMockError(t *testing.T) {
	expectedError := errors.New("mocked error")
	ctx := WithExecFunc(context.Background(), func(name string, arg ...string) ([]byte, []byte, error) {
		return []byte("stdout"), []byte("stderr"), expectedError
	})

	stdout, stderr, err := ExecContext(ctx, nil, "some-command")
	if err == nil {
		t.Error("ExecContext() expected error, got nil")
	}
	if err != expectedError {
		t.Errorf("ExecContext() error = %v, want %v", err, expectedError)
	}

	if string(stdout) != "stdout" {
		t.Errorf("ExecContext() stdout = %q, want %q", string(stdout), "stdout")
	}

	if string(stderr) != "stderr" {
		t.Errorf("ExecContext() stderr = %q, want %q", string(stderr), "stderr")
	}
}

func TestExecContextWithMockEmptyOutput(t *testing.T) {
	ctx := WithExecFunc(context.Background(), func(name string, arg ...string) ([]byte, []byte, error) {
		return []byte{}, []byte{}, nil
	})

	stdout, stderr, err := ExecContext(ctx, nil, "some-command")
	if err != nil {
		t.Fatalf("ExecContext failed: %v", err)
	}

	if len(stdout) > 0 {
		t.Errorf("ExecContext() stdout = %q, want empty", string(stdout))
	}

	if len(stderr) > 0 {
		t.Errorf("ExecContext() stderr = %q, want empty", string(stderr))
	}
}
