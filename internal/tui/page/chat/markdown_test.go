package chat

import (
	"regexp"
	"strings"
	"testing"
)

// stripANSI removes ANSI escape codes from a string for testing purposes.
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

func TestMarkdownRenderer_Render(t *testing.T) {
	r := NewMarkdownRenderer()

	tests := []struct {
		name     string
		content  string
		width    int
		wantErr  bool
		contains []string // Substrings that should be in output
	}{
		{
			name:    "empty content",
			content: "",
			width:   80,
			wantErr: false,
		},
		{
			name:     "plain text",
			content:  "Hello world",
			width:    80,
			wantErr:  false,
			contains: []string{"Hello world"},
		},
		{
			name:     "header",
			content:  "# Title",
			width:    80,
			wantErr:  false,
			contains: []string{"Title"},
		},
		{
			name:     "code block",
			content:  "```go\nfmt.Println(\"hello\")\n```",
			width:    80,
			wantErr:  false,
			contains: []string{"fmt", "Println"},
		},
		{
			name:     "list",
			content:  "- Item 1\n- Item 2",
			width:    80,
			wantErr:  false,
			contains: []string{"Item 1", "Item 2"},
		},
		{
			name:     "bold text",
			content:  "This is **bold** text",
			width:    80,
			wantErr:  false,
			contains: []string{"bold"},
		},
		{
			name:     "italic text",
			content:  "This is *italic* text",
			width:    80,
			wantErr:  false,
			contains: []string{"italic"},
		},
		{
			name:     "inline code",
			content:  "Use `code` here",
			width:    80,
			wantErr:  false,
			contains: []string{"code"},
		},
		{
			name:     "link",
			content:  "Check [this link](https://example.com)",
			width:    80,
			wantErr:  false,
			contains: []string{"this link"},
		},
		{
			name:     "blockquote",
			content:  "> This is a quote",
			width:    80,
			wantErr:  false,
			contains: []string{"This is a quote"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.Render(tt.content, tt.width)
			if (err != nil) != tt.wantErr {
				t.Errorf("Render() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// Strip ANSI codes for content comparison since glamour adds styling
			stripped := stripANSI(got)
			for _, substr := range tt.contains {
				if !strings.Contains(stripped, substr) {
					t.Errorf("Render() output should contain %q, got (stripped) %q", substr, stripped)
				}
			}
		})
	}
}

func TestMarkdownRenderer_CachesRenderer(t *testing.T) {
	r := NewMarkdownRenderer()

	// First render
	_, err := r.Render("# Test", 80)
	if err != nil {
		t.Fatal(err)
	}

	// Check cache is populated
	r.mu.RLock()
	firstRenderer := r.renderer
	cachedWidth := r.cachedWidth
	r.mu.RUnlock()

	if firstRenderer == nil {
		t.Error("Expected renderer to be cached after first render")
	}
	if cachedWidth != 80 {
		t.Errorf("Expected cached width to be 80, got %d", cachedWidth)
	}

	// Second render with same width
	_, err = r.Render("# Test 2", 80)
	if err != nil {
		t.Fatal(err)
	}

	r.mu.RLock()
	secondRenderer := r.renderer
	r.mu.RUnlock()

	if firstRenderer != secondRenderer {
		t.Error("Expected renderer to be reused for same width")
	}
}

func TestMarkdownRenderer_InvalidatesOnWidthChange(t *testing.T) {
	r := NewMarkdownRenderer()

	// First render at width 80
	_, err := r.Render("# Test", 80)
	if err != nil {
		t.Fatal(err)
	}

	r.mu.RLock()
	firstRenderer := r.renderer
	r.mu.RUnlock()

	// Second render at width 100
	_, err = r.Render("# Test", 100)
	if err != nil {
		t.Fatal(err)
	}

	r.mu.RLock()
	secondRenderer := r.renderer
	newWidth := r.cachedWidth
	r.mu.RUnlock()

	if firstRenderer == secondRenderer {
		t.Error("Expected renderer to be recreated for different width")
	}
	if newWidth != 100 {
		t.Errorf("Expected cached width to be 100, got %d", newWidth)
	}
}

func TestMarkdownRenderer_EmptyContent(t *testing.T) {
	r := NewMarkdownRenderer()

	got, err := r.Render("", 80)
	if err != nil {
		t.Errorf("Render() unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("Render() expected empty string, got %q", got)
	}
}

func TestMarkdownRenderer_NarrowWidth(t *testing.T) {
	r := NewMarkdownRenderer()

	// Test with very narrow width - should still work
	got, err := r.Render("This is a long line that should wrap", 20)
	if err != nil {
		t.Errorf("Render() unexpected error: %v", err)
	}
	if got == "" {
		t.Error("Render() expected non-empty output")
	}
}
