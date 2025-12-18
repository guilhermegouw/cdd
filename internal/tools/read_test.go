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

func TestReadTool(t *testing.T) {
	// Create a temporary directory for tests
	tmpDir, err := os.MkdirTemp("", "read_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tool := NewReadTool(tmpDir)
	ctx := context.Background()

	t.Run("read entire file", func(t *testing.T) {
		resp, err := invokeReadTool(ctx, tool, ReadParams{FilePath: testFile})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if resp.IsError {
			t.Fatalf("Unexpected error response: %s", getTextContent(resp))
		}

		content := getTextContent(resp)
		if !strings.Contains(content, "Line 1") {
			t.Errorf("Expected content to contain 'Line 1', got: %s", content)
		}
		if !strings.Contains(content, "Line 5") {
			t.Errorf("Expected content to contain 'Line 5', got: %s", content)
		}
	})

	t.Run("read with offset", func(t *testing.T) {
		resp, err := invokeReadTool(ctx, tool, ReadParams{
			FilePath: testFile,
			Offset:   2,
			Limit:    2,
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		content := getTextContent(resp)
		if !strings.Contains(content, "Line 3") {
			t.Errorf("Expected content to contain 'Line 3' (after offset), got: %s", content)
		}
	})

	t.Run("file not found", func(t *testing.T) {
		resp, err := invokeReadTool(ctx, tool, ReadParams{
			FilePath: filepath.Join(tmpDir, "nonexistent.txt"),
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !resp.IsError {
			t.Error("Expected error response for nonexistent file")
		}

		content := getTextContent(resp)
		if !strings.Contains(content, "not found") {
			t.Errorf("Expected 'not found' in error, got: %s", content)
		}
	})

	t.Run("empty file_path", func(t *testing.T) {
		resp, err := invokeReadTool(ctx, tool, ReadParams{FilePath: ""})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !resp.IsError {
			t.Error("Expected error response for empty file_path")
		}
	})

	t.Run("read directory", func(t *testing.T) {
		resp, err := invokeReadTool(ctx, tool, ReadParams{FilePath: tmpDir})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !resp.IsError {
			t.Error("Expected error response when reading a directory")
		}

		content := getTextContent(resp)
		if !strings.Contains(content, "directory") {
			t.Errorf("Expected 'directory' in error, got: %s", content)
		}
	})

	t.Run("line numbers are added", func(t *testing.T) {
		resp, err := invokeReadTool(ctx, tool, ReadParams{FilePath: testFile})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		content := getTextContent(resp)
		// Line numbers should be present
		if !strings.Contains(content, "1\t") {
			t.Errorf("Expected line numbers in output, got: %s", content)
		}
	})
}

func TestReadToolLongLines(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "read_test_long")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create file with a very long line
	longLine := strings.Repeat("x", MaxLineLength+100)
	testFile := filepath.Join(tmpDir, "long.txt")
	if err := os.WriteFile(testFile, []byte(longLine), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tool := NewReadTool(tmpDir)
	ctx := context.Background()

	resp, err := invokeReadTool(ctx, tool, ReadParams{FilePath: testFile})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	content := getTextContent(resp)
	// Line should be truncated with ...
	if !strings.Contains(content, "...") {
		t.Errorf("Expected truncated line to contain '...', got length: %d", len(content))
	}
}

// Helper functions

func invokeReadTool(ctx context.Context, tool fantasy.AgentTool, params ReadParams) (fantasy.ToolResponse, error) {
	inputJSON, err := json.Marshal(params)
	if err != nil {
		return fantasy.ToolResponse{}, err
	}

	call := fantasy.ToolCall{
		ID:    "test-call",
		Name:  ReadToolName,
		Input: string(inputJSON),
	}

	return tool.Run(ctx, call)
}

func getTextContent(resp fantasy.ToolResponse) string {
	return resp.Content
}
