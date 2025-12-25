package chat

import (
	"strings"
	"testing"
)

func TestActivityPanel_NewActivityPanel(t *testing.T) {
	p := NewActivityPanel()

	if p == nil {
		t.Fatal("expected non-nil panel")
	}
	if p.maxTools != 5 {
		t.Errorf("expected maxTools=5, got %d", p.maxTools)
	}
	if p.thinking {
		t.Error("expected thinking=false initially")
	}
	if len(p.tools) != 0 {
		t.Error("expected empty tools initially")
	}
}

func TestActivityPanel_Height(t *testing.T) {
	p := NewActivityPanel()
	p.SetWidth(80)

	// Empty panel has zero height
	if h := p.Height(); h != 0 {
		t.Errorf("expected height=0 when empty, got %d", h)
	}

	// Thinking adds height
	p.SetThinking(true)
	if h := p.Height(); h != 1 {
		t.Errorf("expected height=1 with thinking only, got %d", h)
	}

	// Tools add height
	p.AddTool("read", `{"file_path": "/path/to/file.go"}`)
	if h := p.Height(); h != 2 {
		t.Errorf("expected height=2 with thinking + 1 tool, got %d", h)
	}

	// Multiple tools
	p.AddTool("grep", `{"pattern": "func.*", "include": "*.go"}`)
	if h := p.Height(); h != 3 {
		t.Errorf("expected height=3 with thinking + 2 tools, got %d", h)
	}

	// Clear resets height
	p.Clear()
	if h := p.Height(); h != 0 {
		t.Errorf("expected height=0 after clear, got %d", h)
	}
}

func TestActivityPanel_IsActive(t *testing.T) {
	p := NewActivityPanel()

	if p.IsActive() {
		t.Error("expected IsActive=false when empty")
	}

	p.SetThinking(true)
	if !p.IsActive() {
		t.Error("expected IsActive=true when thinking")
	}

	p.SetThinking(false)
	if p.IsActive() {
		t.Error("expected IsActive=false when not thinking and no tools")
	}

	p.AddTool("read", `{"file_path": "/test.go"}`)
	if !p.IsActive() {
		t.Error("expected IsActive=true when has tools")
	}
}

func TestActivityPanel_SpinnerAnimation(t *testing.T) {
	p := NewActivityPanel()
	p.SetWidth(80)
	p.SetThinking(true)

	// Get first frame
	frame1 := p.View()

	// Advance spinner
	p.Update(SpinnerTickMsg{})
	frame2 := p.View()

	// Frames should be different (spinner advanced)
	if frame1 == frame2 {
		t.Error("expected spinner to animate between frames")
	}

	// Spinner should cycle through frames
	initialSpinner := p.spinner
	for i := 0; i < len(spinnerFrames); i++ {
		p.Update(SpinnerTickMsg{})
	}
	if p.spinner != initialSpinner {
		t.Error("expected spinner to cycle back to initial frame")
	}
}

func TestActivityPanel_ToolManagement(t *testing.T) {
	p := NewActivityPanel()
	p.SetWidth(80)

	// Add tool
	p.AddTool("read", `{"file_path": "/path/to/file.go"}`)
	if len(p.tools) != 1 {
		t.Errorf("expected 1 tool, got %d", len(p.tools))
	}
	if p.tools[0].Name != "read" {
		t.Errorf("expected tool name 'read', got %q", p.tools[0].Name)
	}
	if p.tools[0].Status != ToolStatusRunning {
		t.Error("expected tool status Running")
	}

	// Mark done
	p.MarkToolDone("read")
	if p.tools[0].Status != ToolStatusDone {
		t.Error("expected tool status Done after MarkToolDone")
	}

	// Add another and mark error
	p.AddTool("bash", `{"command": "go test"}`)
	p.MarkToolError("bash")
	if p.tools[1].Status != ToolStatusError {
		t.Error("expected tool status Error after MarkToolError")
	}
}

func TestActivityPanel_MaxTools(t *testing.T) {
	p := NewActivityPanel()
	p.maxTools = 3

	// Add more tools than maxTools
	for i := 0; i < 5; i++ {
		p.AddTool("read", `{"file_path": "/file.go"}`)
	}

	if len(p.tools) != 3 {
		t.Errorf("expected tools to be capped at 3, got %d", len(p.tools))
	}
}

func TestActivityPanel_View(t *testing.T) {
	p := NewActivityPanel()
	p.SetWidth(80)

	// Empty view
	if v := p.View(); v != "" {
		t.Errorf("expected empty view when inactive, got %q", v)
	}

	// Thinking view
	p.SetThinking(true)
	view := p.View()
	if !strings.Contains(view, "Thinking...") {
		t.Error("expected 'Thinking...' in view")
	}
	if !containsSpinnerFrame(view) {
		t.Error("expected spinner character in view")
	}

	// With tools
	p.AddTool("read", `{"file_path": "/path/to/file.go"}`)
	view = p.View()
	if !strings.Contains(view, "read") {
		t.Error("expected tool name 'read' in view")
	}
	if !strings.Contains(view, "file.go") {
		t.Error("expected filename 'file.go' in view")
	}
}

func TestActivityPanel_Clear(t *testing.T) {
	p := NewActivityPanel()

	p.SetThinking(true)
	p.AddTool("read", `{"file_path": "/test.go"}`)
	p.spinner = 5

	p.Clear()

	if p.thinking {
		t.Error("expected thinking=false after Clear")
	}
	if len(p.tools) != 0 {
		t.Error("expected empty tools after Clear")
	}
	if p.spinner != 0 {
		t.Error("expected spinner=0 after Clear")
	}
}

func TestToolSummary(t *testing.T) {
	tests := []struct { //nolint:govet // fieldalignment: test struct clarity over optimization
		testName string
		toolName string
		input    string
		expected string
	}{
		{
			testName: "read file",
			toolName: "read",
			input:    `{"file_path": "/home/user/code/main.go"}`,
			expected: "main.go",
		},
		{
			testName: "write file",
			toolName: "write",
			input:    `{"file_path": "/home/user/new_file.go", "content": "package main"}`,
			expected: "new_file.go",
		},
		{
			testName: "edit file",
			toolName: "edit",
			input:    `{"file_path": "/path/to/edit.go", "old_string": "foo", "new_string": "bar"}`,
			expected: "edit.go",
		},
		{
			testName: "grep with include",
			toolName: "grep",
			input:    `{"pattern": "func.*", "include": "*.go"}`,
			expected: `"func.*" in *.go`,
		},
		{
			testName: "grep with path",
			toolName: "grep",
			input:    `{"pattern": "test", "path": "/some/dir"}`,
			expected: `"test" in dir`,
		},
		{
			testName: "glob pattern only",
			toolName: "glob",
			input:    `{"pattern": "**/*.go"}`,
			expected: "**/*.go",
		},
		{
			testName: "glob with path",
			toolName: "glob",
			input:    `{"pattern": "*.ts", "path": "/src"}`,
			expected: "*.ts in src",
		},
		{
			testName: "bash command",
			toolName: "bash",
			input:    `{"command": "go test ./..."}`,
			expected: "go test ./...",
		},
		{
			testName: "bash long command",
			toolName: "bash",
			input:    `{"command": "this is a very long command that should be truncated because it exceeds the maximum length"}`,
			expected: "this is a very long command that should be trun...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			result := toolSummary(tt.toolName, tt.input)
			if result != tt.expected {
				t.Errorf("toolSummary(%q, ...) = %q, want %q", tt.toolName, result, tt.expected)
			}
		})
	}
}

func TestToolSummary_InvalidJSON(t *testing.T) {
	result := toolSummary("read", "not json")
	if result != "not json" {
		t.Errorf("expected raw input for invalid JSON, got %q", result)
	}

	// Long invalid input should be truncated
	longInput := strings.Repeat("x", 100)
	result = toolSummary("read", longInput)
	if len(result) > 53 { // 50 + "..."
		t.Errorf("expected truncated input, got length %d", len(result))
	}
}

func TestExtractFilename(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/home/user/file.go", "file.go"},
		{"/a/b/c/d.txt", "d.txt"},
		{"file.go", "file.go"},
		{"/file.go", "file.go"},
		{"C:\\Users\\test\\file.go", "file.go"},
	}

	for _, tt := range tests {
		result := extractFilename(tt.path)
		if result != tt.expected {
			t.Errorf("extractFilename(%q) = %q, want %q", tt.path, result, tt.expected)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct { //nolint:govet // fieldalignment: test struct clarity over optimization
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is longer", 10, "this is..."},
		{"abc", 3, "abc"},
		{"abcd", 3, "abc"},
	}

	for _, tt := range tests {
		result := truncate(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
		}
	}
}

func containsSpinnerFrame(s string) bool {
	for _, frame := range spinnerFrames {
		if strings.Contains(s, frame) {
			return true
		}
	}
	return false
}
