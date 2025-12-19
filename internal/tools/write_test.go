package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"charm.land/fantasy"
)

//nolint:gocyclo // Test functions naturally have high complexity
func TestWriteTool(t *testing.T) {
	// Create a temporary directory for tests
	tmpDir, err := os.MkdirTemp("", "write_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir) //nolint:errcheck // Cleanup in tests

	// Clear file records before each test
	ClearFileRecords()

	tool := NewWriteTool(tmpDir)
	ctx := context.Background()

	t.Run("create new file", func(t *testing.T) {
		ClearFileRecords()
		testFile := filepath.Join(tmpDir, "new_file.txt")
		content := "Hello, World!"

		resp, err := invokeWriteTool(ctx, tool, WriteParams{
			FilePath: testFile,
			Content:  content,
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if resp.IsError {
			t.Fatalf("Unexpected error response: %s", getTextContent(resp))
		}

		// Verify file was created
		data, err := os.ReadFile(testFile) //nolint:gosec // G304: Test file path is controlled
		if err != nil {
			t.Fatalf("Failed to read created file: %v", err)
		}

		if string(data) != content {
			t.Errorf("Expected content %q, got %q", content, string(data))
		}

		respText := getTextContent(resp)
		if !strings.Contains(respText, "created") {
			t.Errorf("Expected 'created' in response, got: %s", respText)
		}
	})

	t.Run("overwrite existing file", func(t *testing.T) {
		ClearFileRecords()
		testFile := filepath.Join(tmpDir, "existing.txt")
		originalContent := "Original content"
		newContent := "New content"

		// Create the file first
		if err := os.WriteFile(testFile, []byte(originalContent), 0o600); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Record that we read it (required before writing)
		RecordFileRead(testFile)

		resp, err := invokeWriteTool(ctx, tool, WriteParams{
			FilePath: testFile,
			Content:  newContent,
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if resp.IsError {
			t.Fatalf("Unexpected error response: %s", getTextContent(resp))
		}

		// Verify content was updated
		data, err := os.ReadFile(testFile) //nolint:gosec // G304: Test file path is controlled
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		if string(data) != newContent {
			t.Errorf("Expected content %q, got %q", newContent, string(data))
		}
	})

	t.Run("empty file_path", func(t *testing.T) {
		resp, err := invokeWriteTool(ctx, tool, WriteParams{
			FilePath: "",
			Content:  "test",
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !resp.IsError {
			t.Error("Expected error response for empty file_path")
		}
	})

	t.Run("empty content", func(t *testing.T) {
		resp, err := invokeWriteTool(ctx, tool, WriteParams{
			FilePath: filepath.Join(tmpDir, "empty.txt"),
			Content:  "",
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !resp.IsError {
			t.Error("Expected error response for empty content")
		}
	})

	t.Run("create file in nested directory", func(t *testing.T) {
		ClearFileRecords()
		testFile := filepath.Join(tmpDir, "a", "b", "c", "nested.txt")
		content := "Nested content"

		resp, err := invokeWriteTool(ctx, tool, WriteParams{
			FilePath: testFile,
			Content:  content,
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if resp.IsError {
			t.Fatalf("Unexpected error response: %s", getTextContent(resp))
		}

		// Verify file was created
		data, err := os.ReadFile(testFile) //nolint:gosec // G304: Test file path is controlled
		if err != nil {
			t.Fatalf("Failed to read nested file: %v", err)
		}

		if string(data) != content {
			t.Errorf("Expected content %q, got %q", content, string(data))
		}
	})

	t.Run("write to directory path fails", func(t *testing.T) {
		ClearFileRecords()
		dirPath := filepath.Join(tmpDir, "testdir")
		if err := os.MkdirAll(dirPath, 0o750); err != nil {
			t.Fatalf("Failed to create test directory: %v", err)
		}

		resp, err := invokeWriteTool(ctx, tool, WriteParams{
			FilePath: dirPath,
			Content:  "test",
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !resp.IsError {
			t.Error("Expected error response when writing to directory")
		}

		respText := getTextContent(resp)
		if !strings.Contains(respText, "directory") {
			t.Errorf("Expected 'directory' in error, got: %s", respText)
		}
	})

	t.Run("no-op write is rejected", func(t *testing.T) {
		ClearFileRecords()
		testFile := filepath.Join(tmpDir, "same_content.txt")
		content := "Same content"

		// Create the file first
		if err := os.WriteFile(testFile, []byte(content), 0o600); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Record that we read it
		RecordFileRead(testFile)

		// Try to write same content
		resp, err := invokeWriteTool(ctx, tool, WriteParams{
			FilePath: testFile,
			Content:  content,
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !resp.IsError {
			t.Error("Expected error response for no-op write")
		}

		respText := getTextContent(resp)
		if !strings.Contains(respText, "already contains") {
			t.Errorf("Expected 'already contains' in error, got: %s", respText)
		}
	})
}

func invokeWriteTool(ctx context.Context, tool fantasy.AgentTool, params WriteParams) (fantasy.ToolResponse, error) {
	inputJSON, err := json.Marshal(params)
	if err != nil {
		return fantasy.ToolResponse{}, err
	}

	call := fantasy.ToolCall{
		ID:    "test-call",
		Name:  WriteToolName,
		Input: string(inputJSON),
	}
	return tool.Run(ctx, call)
}
