package chat

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/guilhermegouw/cdd/internal/tui/styles"
)

// Spinner animation frames (braille pattern).
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// spinnerInterval is the time between spinner frame updates.
const spinnerInterval = 100 * time.Millisecond

// ToolStatus represents the status of a tool operation.
type ToolStatus int

// Tool status constants.
const (
	ToolStatusPending ToolStatus = iota
	ToolStatusRunning
	ToolStatusDone
	ToolStatusError
)

// ToolActivity represents a single tool operation.
type ToolActivity struct {
	Name    string
	Summary string
	Status  ToolStatus
}

// SpinnerTickMsg is sent to advance the spinner animation.
type SpinnerTickMsg struct{}

// ActivityPanel shows real-time activity during AI interactions.
type ActivityPanel struct { //nolint:govet // fieldalignment: preserving logical field order
	spinner  int            // Current spinner frame index
	thinking bool           // Whether we're in thinking state
	tools    []ToolActivity // Active/recent tool calls
	width    int
	maxTools int // Max visible tools (default: 5)
}

// NewActivityPanel creates a new activity panel.
func NewActivityPanel() *ActivityPanel {
	return &ActivityPanel{
		maxTools: 5,
	}
}

// SetThinking sets the thinking state and starts/stops the spinner.
func (a *ActivityPanel) SetThinking(thinking bool) tea.Cmd {
	a.thinking = thinking
	if thinking {
		return a.tickSpinner()
	}
	return nil
}

// AddTool adds a tool operation to the activity list.
func (a *ActivityPanel) AddTool(name, input string) {
	summary := toolSummary(name, input)
	tool := ToolActivity{
		Name:    name,
		Summary: summary,
		Status:  ToolStatusRunning,
	}

	a.tools = append(a.tools, tool)

	// Trim to maxTools
	if len(a.tools) > a.maxTools {
		a.tools = a.tools[len(a.tools)-a.maxTools:]
	}
}

// MarkToolDone marks a tool as completed.
func (a *ActivityPanel) MarkToolDone(name string) {
	// Mark the most recent tool with this name as done
	for i := len(a.tools) - 1; i >= 0; i-- {
		if a.tools[i].Name == name && a.tools[i].Status == ToolStatusRunning {
			a.tools[i].Status = ToolStatusDone
			break
		}
	}
}

// MarkToolError marks a tool as failed.
func (a *ActivityPanel) MarkToolError(name string) {
	// Mark the most recent tool with this name as error
	for i := len(a.tools) - 1; i >= 0; i-- {
		if a.tools[i].Name == name && a.tools[i].Status == ToolStatusRunning {
			a.tools[i].Status = ToolStatusError
			break
		}
	}
}

// Clear resets the activity panel.
func (a *ActivityPanel) Clear() {
	a.thinking = false
	a.tools = nil
	a.spinner = 0
}

// SetWidth sets the panel width.
func (a *ActivityPanel) SetWidth(width int) {
	a.width = width
}

// Height returns the current height of the panel (0 when hidden).
func (a *ActivityPanel) Height() int {
	if !a.thinking && len(a.tools) == 0 {
		return 0
	}

	height := 0
	if a.thinking {
		height++ // Thinking line
	}
	height += len(a.tools) // Tool lines
	return height
}

// IsActive returns true if the panel has content to show.
func (a *ActivityPanel) IsActive() bool {
	return a.thinking || len(a.tools) > 0
}

// Update handles messages for the activity panel.
func (a *ActivityPanel) Update(msg tea.Msg) (*ActivityPanel, tea.Cmd) {
	if _, ok := msg.(SpinnerTickMsg); ok && a.thinking {
		a.spinner = (a.spinner + 1) % len(spinnerFrames)
		cmd := a.tickSpinner()
		return a, cmd
	}
	return a, nil
}

// tickSpinner returns a command that sends a SpinnerTickMsg after the interval.
func (a *ActivityPanel) tickSpinner() tea.Cmd {
	return tea.Tick(spinnerInterval, func(time.Time) tea.Msg {
		return SpinnerTickMsg{}
	})
}

// View renders the activity panel.
func (a *ActivityPanel) View() string {
	if !a.IsActive() {
		return ""
	}

	t := styles.CurrentTheme()

	// Pre-allocate lines slice
	lineCount := len(a.tools)
	if a.thinking {
		lineCount++
	}
	lines := make([]string, 0, lineCount)

	// Thinking line with spinner
	if a.thinking {
		spinnerChar := spinnerFrames[a.spinner]
		thinkingStyle := t.S().Info
		thinkingLine := thinkingStyle.Render(spinnerChar + " Thinking...")
		lines = append(lines, thinkingLine)
	}

	// Tool lines
	for i, tool := range a.tools {
		var prefix string
		if i == len(a.tools)-1 {
			prefix = "   └─ "
		} else {
			prefix = "   ├─ "
		}

		statusStyle := a.statusStyle(t, tool.Status)

		// Format: prefix + tool_name: summary
		toolLine := t.S().Muted.Render(prefix) +
			statusStyle.Render(tool.Name) +
			t.S().Muted.Render(": ") +
			t.S().Text.Render(a.truncateSummary(tool.Summary))

		lines = append(lines, toolLine)
	}

	content := strings.Join(lines, "\n")

	// Apply padding
	return lipgloss.NewStyle().
		Padding(0, 1).
		Width(a.width).
		Render(content)
}

// statusStyle returns the appropriate style for a tool status.
func (a *ActivityPanel) statusStyle(t *styles.Theme, status ToolStatus) lipgloss.Style {
	//nolint:exhaustive // ToolStatusPending uses default case
	switch status {
	case ToolStatusRunning:
		return t.S().Warning
	case ToolStatusDone:
		return t.S().Success
	case ToolStatusError:
		return t.S().Error
	default:
		return t.S().Muted
	}
}

// truncateSummary truncates a summary to fit the available width.
func (a *ActivityPanel) truncateSummary(summary string) string {
	// Reserve space for prefix, tool name, and some padding
	maxLen := a.width - 30
	if maxLen < 10 {
		maxLen = 10
	}

	if len(summary) <= maxLen {
		return summary
	}

	return summary[:maxLen-3] + "..."
}

// toolSummary extracts a human-readable summary from tool input JSON.
func toolSummary(name, input string) string {
	// Parse the input JSON
	var params map[string]any
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		// If parsing fails, return a truncated version of the input
		return truncate(input, 50)
	}

	switch name {
	case "read", "write", "edit":
		return summarizeFileTool(params)
	case "grep":
		return summarizeGrepTool(params)
	case "glob":
		return summarizeGlobTool(params)
	case "bash":
		return summarizeBashTool(params)
	default:
		return summarizeFallback(params)
	}
}

func summarizeFileTool(params map[string]any) string {
	if path, ok := params["file_path"].(string); ok {
		return extractFilename(path)
	}
	return summarizeFallback(params)
}

func summarizeGrepTool(params map[string]any) string {
	var parts []string
	if pattern, ok := params["pattern"].(string); ok {
		parts = append(parts, fmt.Sprintf("%q", truncate(pattern, 20)))
	}
	if include, ok := params["include"].(string); ok {
		parts = append(parts, "in "+include)
	} else if path, ok := params["path"].(string); ok && path != "" && path != "." {
		parts = append(parts, "in "+extractFilename(path))
	}
	if len(parts) > 0 {
		return strings.Join(parts, " ")
	}
	return summarizeFallback(params)
}

func summarizeGlobTool(params map[string]any) string {
	var parts []string
	if pattern, ok := params["pattern"].(string); ok {
		parts = append(parts, pattern)
	}
	if path, ok := params["path"].(string); ok && path != "" && path != "." {
		parts = append(parts, "in "+extractFilename(path))
	}
	if len(parts) > 0 {
		return strings.Join(parts, " ")
	}
	return summarizeFallback(params)
}

func summarizeBashTool(params map[string]any) string {
	if cmd, ok := params["command"].(string); ok {
		return truncate(cmd, 50)
	}
	return summarizeFallback(params)
}

func summarizeFallback(params map[string]any) string {
	// Return first parameter value found
	for _, v := range params {
		if s, ok := v.(string); ok && s != "" {
			return truncate(s, 50)
		}
	}
	return ""
}

// extractFilename extracts just the filename from a path.
func extractFilename(path string) string {
	// Find the last separator
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[i+1:]
		}
	}
	return path
}

// truncate truncates a string to maxLen, adding ellipsis if needed.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
