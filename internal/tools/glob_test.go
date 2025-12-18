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
func TestGlobTool(t *testing.T) {
	// Create a temporary directory structure for tests
	tmpDir, err := os.MkdirTemp("", "glob_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir) //nolint:errcheck // Cleanup in tests

	// Create test directory structure
	// tmpDir/
	//   file1.go
	//   file2.go
	//   file3.txt
	//   subdir/
	//     nested.go
	//     nested.txt

	files := map[string]string{
		"file1.go":          "package main",
		"file2.go":          "package main",
		"file3.txt":         "text file",
		"subdir/nested.go":  "package sub",
		"subdir/nested.txt": "nested text",
	}

	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", path, err)
		}
	}

	tool := NewGlobTool(tmpDir)
	ctx := context.Background()

	t.Run("find all go files", func(t *testing.T) {
		resp, err := invokeGlobTool(ctx, tool, GlobParams{Pattern: "**/*.go"})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if resp.IsError {
			t.Fatalf("Unexpected error response: %s", getTextContent(resp))
		}

		content := getTextContent(resp)
		// Should find 3 .go files
		if !strings.Contains(content, "file1.go") {
			t.Errorf("Expected to find file1.go, got: %s", content)
		}
		if !strings.Contains(content, "file2.go") {
			t.Errorf("Expected to find file2.go, got: %s", content)
		}
		if !strings.Contains(content, "nested.go") {
			t.Errorf("Expected to find nested.go, got: %s", content)
		}
	})

	t.Run("find txt files in root", func(t *testing.T) {
		resp, err := invokeGlobTool(ctx, tool, GlobParams{Pattern: "*.txt"})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		content := getTextContent(resp)
		if !strings.Contains(content, "file3.txt") {
			t.Errorf("Expected to find file3.txt, got: %s", content)
		}
	})

	t.Run("no matches", func(t *testing.T) {
		resp, err := invokeGlobTool(ctx, tool, GlobParams{Pattern: "**/*.xyz"})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		content := getTextContent(resp)
		if !strings.Contains(content, "No files found") {
			t.Errorf("Expected 'No files found', got: %s", content)
		}
	})

	t.Run("empty pattern", func(t *testing.T) {
		resp, err := invokeGlobTool(ctx, tool, GlobParams{Pattern: ""})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !resp.IsError {
			t.Error("Expected error response for empty pattern")
		}
	})

	t.Run("search in subdirectory", func(t *testing.T) {
		resp, err := invokeGlobTool(ctx, tool, GlobParams{
			Pattern: "*.go",
			Path:    "subdir",
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		content := getTextContent(resp)
		if !strings.Contains(content, "nested.go") {
			t.Errorf("Expected to find nested.go in subdir, got: %s", content)
		}
		// Should NOT find root level .go files
		if strings.Contains(content, "file1.go") {
			t.Errorf("Should not find file1.go when searching in subdir, got: %s", content)
		}
	})

	t.Run("nonexistent directory", func(t *testing.T) {
		resp, err := invokeGlobTool(ctx, tool, GlobParams{
			Pattern: "*.go",
			Path:    "nonexistent",
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !resp.IsError {
			t.Error("Expected error response for nonexistent directory")
		}
	})
}

func invokeGlobTool(ctx context.Context, tool fantasy.AgentTool, params GlobParams) (fantasy.ToolResponse, error) {
	inputJSON, err := json.Marshal(params)
	if err != nil {
		return fantasy.ToolResponse{}, err
	}

	call := fantasy.ToolCall{
		ID:    "test-call",
		Name:  GlobToolName,
		Input: string(inputJSON),
	}
	return tool.Run(ctx, call)
}
