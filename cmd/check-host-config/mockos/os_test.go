package mockos

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestReadFileContext(t *testing.T) {
	// Create a temporary file for testing
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := []byte("test content\nline 2")
	err := os.WriteFile(testFile, content, 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ctx := context.Background()
	data, err := ReadFileContext(ctx, nil, testFile)
	if err != nil {
		t.Fatalf("ReadFileContext failed: %v", err)
	}

	if string(data) != string(content) {
		t.Errorf("ReadFileContext() = %q, want %q", string(data), string(content))
	}
}

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

func TestReadFileContextNotFound(t *testing.T) {
	ctx := context.Background()
	_, err := ReadFileContext(ctx, nil, "/nonexistent/file")
	if err == nil {
		t.Error("ReadFileContext() expected error for nonexistent file, got nil")
	}
}
