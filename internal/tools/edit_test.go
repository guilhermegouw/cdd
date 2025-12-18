//nolint:dupl // Test cases are intentionally similar but test different scenarios
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
func TestEditTool(t *testing.T) {
	// Create a temporary directory for tests
	tmpDir, err := os.MkdirTemp("", "edit_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir) //nolint:errcheck // Cleanup in tests

	tool := NewEditTool(tmpDir)
	ctx := context.Background()

	t.Run("replace single occurrence", func(t *testing.T) {
		ClearFileRecords()
		testFile := filepath.Join(tmpDir, "replace_single.txt")
		originalContent := "Hello World"

		if err := os.WriteFile(testFile, []byte(originalContent), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		RecordFileRead(testFile)

		resp, err := invokeEditTool(ctx, tool, EditParams{
			FilePath:  testFile,
			OldString: "World",
			NewString: "Go",
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if resp.IsError {
			t.Fatalf("Unexpected error response: %s", getTextContent(resp))
		}

		data, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		if string(data) != "Hello Go" {
			t.Errorf("Expected 'Hello Go', got %q", string(data))
		}
	})

	t.Run("replace all occurrences", func(t *testing.T) {
		ClearFileRecords()
		testFile := filepath.Join(tmpDir, "replace_all.txt")
		originalContent := "foo bar foo baz foo"

		if err := os.WriteFile(testFile, []byte(originalContent), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		RecordFileRead(testFile)

		resp, err := invokeEditTool(ctx, tool, EditParams{
			FilePath:   testFile,
			OldString:  "foo",
			NewString:  "qux",
			ReplaceAll: true,
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if resp.IsError {
			t.Fatalf("Unexpected error response: %s", getTextContent(resp))
		}

		data, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		if string(data) != "qux bar qux baz qux" {
			t.Errorf("Expected 'qux bar qux baz qux', got %q", string(data))
		}
	})

	t.Run("error on multiple occurrences without replace_all", func(t *testing.T) {
		ClearFileRecords()
		testFile := filepath.Join(tmpDir, "multiple.txt")
		originalContent := "foo bar foo"

		if err := os.WriteFile(testFile, []byte(originalContent), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		RecordFileRead(testFile)

		resp, err := invokeEditTool(ctx, tool, EditParams{
			FilePath:  testFile,
			OldString: "foo",
			NewString: "baz",
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !resp.IsError {
			t.Error("Expected error response for multiple occurrences without replace_all")
		}

		respText := getTextContent(resp)
		if !strings.Contains(respText, "multiple times") {
			t.Errorf("Expected 'multiple times' in error, got: %s", respText)
		}
	})

	t.Run("create new file with empty old_string", func(t *testing.T) {
		ClearFileRecords()
		testFile := filepath.Join(tmpDir, "new_via_edit.txt")
		content := "New file content"

		resp, err := invokeEditTool(ctx, tool, EditParams{
			FilePath:  testFile,
			OldString: "",
			NewString: content,
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if resp.IsError {
			t.Fatalf("Unexpected error response: %s", getTextContent(resp))
		}

		data, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		if string(data) != content {
			t.Errorf("Expected %q, got %q", content, string(data))
		}
	})

	t.Run("delete content with empty new_string", func(t *testing.T) {
		ClearFileRecords()
		testFile := filepath.Join(tmpDir, "delete.txt")
		originalContent := "Hello World"

		if err := os.WriteFile(testFile, []byte(originalContent), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		RecordFileRead(testFile)

		resp, err := invokeEditTool(ctx, tool, EditParams{
			FilePath:  testFile,
			OldString: " World",
			NewString: "",
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if resp.IsError {
			t.Fatalf("Unexpected error response: %s", getTextContent(resp))
		}

		data, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		if string(data) != "Hello" {
			t.Errorf("Expected 'Hello', got %q", string(data))
		}
	})

	t.Run("error on file not read first", func(t *testing.T) {
		ClearFileRecords()
		testFile := filepath.Join(tmpDir, "not_read.txt")

		if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		// Intentionally NOT calling RecordFileRead

		resp, err := invokeEditTool(ctx, tool, EditParams{
			FilePath:  testFile,
			OldString: "content",
			NewString: "new",
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !resp.IsError {
			t.Error("Expected error response when file not read first")
		}

		respText := getTextContent(resp)
		if !strings.Contains(respText, "must read the file") {
			t.Errorf("Expected 'must read the file' in error, got: %s", respText)
		}
	})

	t.Run("error on file not found", func(t *testing.T) {
		ClearFileRecords()
		resp, err := invokeEditTool(ctx, tool, EditParams{
			FilePath:  filepath.Join(tmpDir, "nonexistent.txt"),
			OldString: "foo",
			NewString: "bar",
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !resp.IsError {
			t.Error("Expected error response for nonexistent file")
		}
	})

	t.Run("error on old_string not found", func(t *testing.T) {
		ClearFileRecords()
		testFile := filepath.Join(tmpDir, "no_match.txt")

		if err := os.WriteFile(testFile, []byte("Hello World"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		RecordFileRead(testFile)

		resp, err := invokeEditTool(ctx, tool, EditParams{
			FilePath:  testFile,
			OldString: "NOTFOUND",
			NewString: "bar",
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !resp.IsError {
			t.Error("Expected error response when old_string not found")
		}

		respText := getTextContent(resp)
		if !strings.Contains(respText, "not found") {
			t.Errorf("Expected 'not found' in error, got: %s", respText)
		}
	})

	t.Run("empty file_path", func(t *testing.T) {
		resp, err := invokeEditTool(ctx, tool, EditParams{
			FilePath:  "",
			OldString: "foo",
			NewString: "bar",
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !resp.IsError {
			t.Error("Expected error response for empty file_path")
		}
	})
}

func invokeEditTool(ctx context.Context, tool fantasy.AgentTool, params EditParams) (fantasy.ToolResponse, error) {
	inputJSON, err := json.Marshal(params)
	if err != nil {
		return fantasy.ToolResponse{}, err
	}

	call := fantasy.ToolCall{
		ID:    "test-call",
		Name:  EditToolName,
		Input: string(inputJSON),
	}
	return tool.Run(ctx, call)
}
