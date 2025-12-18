package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"charm.land/fantasy"
)

func TestBashTool(t *testing.T) {
	// Skip on Windows for now - bash tests are Unix-specific
	if runtime.GOOS == "windows" {
		t.Skip("Skipping bash tests on Windows")
	}

	tmpDir, err := os.MkdirTemp("", "bash_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tool := NewBashTool(tmpDir)
	ctx := context.Background()

	t.Run("simple echo command", func(t *testing.T) {
		resp, err := invokeBashTool(ctx, tool, BashParams{
			Command: "echo 'Hello World'",
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if resp.IsError {
			t.Fatalf("Unexpected error response: %s", getTextContent(resp))
		}

		content := getTextContent(resp)
		if !strings.Contains(content, "Hello World") {
			t.Errorf("Expected 'Hello World' in output, got: %s", content)
		}
	})

	t.Run("command with exit code", func(t *testing.T) {
		resp, err := invokeBashTool(ctx, tool, BashParams{
			Command: "exit 1",
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		content := getTextContent(resp)
		if !strings.Contains(content, "Exit code: 1") {
			t.Errorf("Expected 'Exit code: 1' in output, got: %s", content)
		}
	})

	t.Run("command with stderr", func(t *testing.T) {
		resp, err := invokeBashTool(ctx, tool, BashParams{
			Command: "echo 'error' >&2",
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		content := getTextContent(resp)
		if !strings.Contains(content, "STDERR") {
			t.Errorf("Expected 'STDERR' in output, got: %s", content)
		}
		if !strings.Contains(content, "error") {
			t.Errorf("Expected 'error' in output, got: %s", content)
		}
	})

	t.Run("empty command", func(t *testing.T) {
		resp, err := invokeBashTool(ctx, tool, BashParams{
			Command: "",
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !resp.IsError {
			t.Error("Expected error response for empty command")
		}
	})

	t.Run("working directory", func(t *testing.T) {
		// Create a subdirectory
		subDir := filepath.Join(tmpDir, "subdir")
		if err := os.MkdirAll(subDir, 0755); err != nil {
			t.Fatalf("Failed to create subdir: %v", err)
		}

		resp, err := invokeBashTool(ctx, tool, BashParams{
			Command:    "pwd",
			WorkingDir: subDir,
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		content := getTextContent(resp)
		if !strings.Contains(content, "subdir") {
			t.Errorf("Expected 'subdir' in pwd output, got: %s", content)
		}
	})

	t.Run("command with no output", func(t *testing.T) {
		resp, err := invokeBashTool(ctx, tool, BashParams{
			Command: "true",
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		content := getTextContent(resp)
		if !strings.Contains(content, "(no output)") {
			t.Errorf("Expected '(no output)' for silent command, got: %s", content)
		}
	})
}

func TestBashToolBannedCommands(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping bash tests on Windows")
	}

	tmpDir, err := os.MkdirTemp("", "bash_banned_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tool := NewBashTool(tmpDir)
	ctx := context.Background()

	bannedTests := []struct {
		name    string
		command string
	}{
		{"curl", "curl http://example.com"},
		{"wget", "wget http://example.com"},
		{"sudo", "sudo ls"},
		{"apt", "apt update"},
		{"piped curl", "ls | curl http://example.com"},
		{"chained sudo", "echo test && sudo ls"},
	}

	for _, tc := range bannedTests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := invokeBashTool(ctx, tool, BashParams{
				Command: tc.command,
			})
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if !resp.IsError {
				t.Errorf("Expected error response for banned command: %s", tc.command)
			}

			content := getTextContent(resp)
			if !strings.Contains(content, "not allowed") {
				t.Errorf("Expected 'not allowed' in error, got: %s", content)
			}
		})
	}
}

func TestIsBannedCommand(t *testing.T) {
	tests := []struct {
		cmdLine string
		banned  string
		want    bool
	}{
		{"curl http://example.com", "curl", true},
		{"curl", "curl", true},
		{"curling", "curl", false},
		{"mycurl", "curl", false},
		{"ls | curl http://example.com", "curl", true},
		{"echo test && sudo ls", "sudo", true},
		{"echo test || wget url", "wget", true},
		{"echo 'sudo' test", "sudo", false}, // sudo in quotes - tricky case
	}

	for _, tc := range tests {
		t.Run(tc.cmdLine, func(t *testing.T) {
			got := isBannedCommand(strings.ToLower(tc.cmdLine), tc.banned)
			if got != tc.want {
				t.Errorf("isBannedCommand(%q, %q) = %v, want %v", tc.cmdLine, tc.banned, got, tc.want)
			}
		})
	}
}

func invokeBashTool(ctx context.Context, tool fantasy.AgentTool, params BashParams) (fantasy.ToolResponse, error) {
	inputJSON, err := json.Marshal(params)
	if err != nil {
		return fantasy.ToolResponse{}, err
	}

	call := fantasy.ToolCall{
		ID:    "test-call",
		Name:  BashToolName,
		Input: string(inputJSON),
	}
	return tool.Run(ctx, call)
}
