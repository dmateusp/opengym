package flagsecret

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSecret_Set(t *testing.T) {
	t.Run("sets plain string value", func(t *testing.T) {
		var s Secret
		err := s.Set("my-secret-value")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := s.Value(); got != "my-secret-value" {
			t.Errorf("expected value to be %q, got %q", "my-secret-value", got)
		}
	})

	t.Run("reads value from file", func(t *testing.T) {
		// Create a temporary file
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "secret.txt")
		content := "file-secret-value"
		if err := os.WriteFile(tmpFile, []byte(content), 0600); err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}

		var s Secret
		err := s.Set("file://" + tmpFile)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := s.Value(); got != content {
			t.Errorf("expected value to be %q, got %q", content, got)
		}
	})

	t.Run("trims whitespace from file content", func(t *testing.T) {
		// Create a temporary file with whitespace
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "secret.txt")
		content := "  file-secret-with-spaces  \n\t"
		if err := os.WriteFile(tmpFile, []byte(content), 0600); err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}

		var s Secret
		err := s.Set("file://" + tmpFile)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := "file-secret-with-spaces"
		if got := s.Value(); got != expected {
			t.Errorf("expected value to be %q, got %q", expected, got)
		}
	})

	t.Run("returns error when file does not exist", func(t *testing.T) {
		var s Secret
		err := s.Set("file:///nonexistent/path/to/file.txt")
		if err == nil {
			t.Fatal("expected error when file does not exist, got nil")
		}
	})

	t.Run("String method returns redacted", func(t *testing.T) {
		var s Secret
		_ = s.Set("my-secret")
		if got := s.String(); got != "[REDACTED]" {
			t.Errorf("expected String() to return %q, got %q", "[REDACTED]", got)
		}
	})
}
