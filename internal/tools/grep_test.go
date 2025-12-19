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
func TestGrepTool(t *testing.T) {
	// Create a temporary directory structure for tests
	tmpDir, err := os.MkdirTemp("", "grep_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir) //nolint:errcheck // Cleanup in tests

	// Create test files with content to search
	files := map[string]string{
		"main.go": `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`,
		"util.go": `package main

func helper() {
	// This is a helper function
}

func anotherHelper() {
	// Another helper
}
`,
		"readme.txt": `This is a README file.
It contains documentation.
Hello from the docs!
`,
		"subdir/nested.go": `package sub

func NestedFunc() {
	// Nested function
}
`,
	}

	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o750); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o600); err != nil {
			t.Fatalf("Failed to create file %s: %v", path, err)
		}
	}

	tool := NewGrepTool(tmpDir)
	ctx := context.Background()

	t.Run("simple pattern search", func(t *testing.T) {
		resp, err := invokeGrepTool(ctx, tool, GrepParams{Pattern: "Hello"})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if resp.IsError {
			t.Fatalf("Unexpected error response: %s", getTextContent(resp))
		}

		content := getTextContent(resp)
		// Should find Hello in main.go and readme.txt
		if !strings.Contains(content, "main.go") {
			t.Errorf("Expected to find match in main.go, got: %s", content)
		}
		if !strings.Contains(content, "readme.txt") {
			t.Errorf("Expected to find match in readme.txt, got: %s", content)
		}
	})

	t.Run("regex pattern search", func(t *testing.T) {
		resp, err := invokeGrepTool(ctx, tool, GrepParams{Pattern: "func.*Helper"})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		content := getTextContent(resp)
		if !strings.Contains(content, "util.go") {
			t.Errorf("Expected to find regex match in util.go, got: %s", content)
		}
	})

	t.Run("literal text search", func(t *testing.T) {
		resp, err := invokeGrepTool(ctx, tool, GrepParams{
			Pattern:     "func.*Helper",
			LiteralText: true,
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		content := getTextContent(resp)
		// With literal text, the regex should NOT match
		if !strings.Contains(content, "No matches found") {
			t.Errorf("Expected no matches for literal regex string, got: %s", content)
		}
	})

	t.Run("search with include filter", func(t *testing.T) {
		resp, err := invokeGrepTool(ctx, tool, GrepParams{
			Pattern: "Hello",
			Include: "*.go",
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		content := getTextContent(resp)
		// Should find Hello only in .go files
		if !strings.Contains(content, "main.go") {
			t.Errorf("Expected to find match in main.go, got: %s", content)
		}
		// Should NOT find it in readme.txt (due to include filter)
		if strings.Contains(content, "readme.txt") {
			t.Errorf("Should not find match in readme.txt with *.go filter, got: %s", content)
		}
	})

	t.Run("no matches", func(t *testing.T) {
		resp, err := invokeGrepTool(ctx, tool, GrepParams{Pattern: "NONEXISTENT_STRING_XYZ"})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		content := getTextContent(resp)
		if !strings.Contains(content, "No matches found") {
			t.Errorf("Expected 'No matches found', got: %s", content)
		}
	})

	t.Run("empty pattern", func(t *testing.T) {
		resp, err := invokeGrepTool(ctx, tool, GrepParams{Pattern: ""})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !resp.IsError {
			t.Error("Expected error response for empty pattern")
		}
	})

	t.Run("invalid regex", func(t *testing.T) {
		resp, err := invokeGrepTool(ctx, tool, GrepParams{Pattern: "[invalid"})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !resp.IsError {
			t.Error("Expected error response for invalid regex")
		}
	})

	t.Run("search in specific path", func(t *testing.T) {
		resp, err := invokeGrepTool(ctx, tool, GrepParams{
			Pattern: "func",
			Path:    "subdir",
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		content := getTextContent(resp)
		if !strings.Contains(content, "nested.go") {
			t.Errorf("Expected to find match in nested.go, got: %s", content)
		}
		// Should NOT find matches in root files
		if strings.Contains(content, "main.go") {
			t.Errorf("Should not find main.go when searching in subdir, got: %s", content)
		}
	})

	t.Run("nonexistent directory", func(t *testing.T) {
		resp, err := invokeGrepTool(ctx, tool, GrepParams{
			Pattern: "test",
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

func invokeGrepTool(ctx context.Context, tool fantasy.AgentTool, params GrepParams) (fantasy.ToolResponse, error) {
	inputJSON, err := json.Marshal(params)
	if err != nil {
		return fantasy.ToolResponse{}, err
	}

	call := fantasy.ToolCall{
		ID:    "test-call",
		Name:  GrepToolName,
		Input: string(inputJSON),
	}
	return tool.Run(ctx, call)
}
