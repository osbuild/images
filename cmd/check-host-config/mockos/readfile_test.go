package mockos

import (
	"context"
	"errors"
	"testing"
)

func TestReadFileContextWithMock(t *testing.T) {
	expectedContent := []byte("mocked content")
	ctx := WithReadFileFunc(context.Background(), func(filename string) ([]byte, error) {
		return expectedContent, nil
	})

	data, err := ReadFileContext(ctx, nil, "/some/path")
	if err != nil {
		t.Fatalf("ReadFileContext failed: %v", err)
	}

	if string(data) != string(expectedContent) {
		t.Errorf("ReadFileContext() = %q, want %q", string(data), string(expectedContent))
	}
}

func TestReadFileContextWithMockError(t *testing.T) {
	expectedError := errors.New("mocked file error")
	ctx := WithReadFileFunc(context.Background(), func(filename string) ([]byte, error) {
		return nil, expectedError
	})

	_, err := ReadFileContext(ctx, nil, "/some/path")
	if err == nil {
		t.Error("ReadFileContext() expected error, got nil")
	}
	if err != expectedError {
		t.Errorf("ReadFileContext() error = %v, want %v", err, expectedError)
	}
}

func TestReadFileContextWithMockEmptyFile(t *testing.T) {
	ctx := WithReadFileFunc(context.Background(), func(filename string) ([]byte, error) {
		return []byte{}, nil
	})

	data, err := ReadFileContext(ctx, nil, "/some/path")
	if err != nil {
		t.Fatalf("ReadFileContext failed: %v", err)
	}

	if len(data) > 0 {
		t.Errorf("ReadFileContext() = %q, want empty", string(data))
	}
}
