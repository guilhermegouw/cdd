package chat

import (
	"fmt"
	"image/color"
	"sync"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	glamourstyles "github.com/charmbracelet/glamour/styles"
	"github.com/muesli/termenv"

	"github.com/guilhermegouw/cdd/internal/tui/styles"
)

// MarkdownRenderer handles markdown rendering with caching.
type MarkdownRenderer struct {
	renderer    *glamour.TermRenderer
	cachedWidth int
	mu          sync.RWMutex
}

// NewMarkdownRenderer creates a new markdown renderer.
func NewMarkdownRenderer() *MarkdownRenderer {
	return &MarkdownRenderer{}
}

// Render renders markdown content to styled terminal output.
// It caches the renderer and recreates it only when width changes.
func (m *MarkdownRenderer) Render(content string, width int) (string, error) {
	if content == "" {
		return "", nil
	}

	renderer, err := m.getRenderer(width)
	if err != nil {
		return content, err // Fallback to plain text
	}

	rendered, err := renderer.Render(content)
	if err != nil {
		return content, err // Fallback to plain text
	}

	return rendered, nil
}

func (m *MarkdownRenderer) getRenderer(width int) (*glamour.TermRenderer, error) {
	m.mu.RLock()
	if m.renderer != nil && m.cachedWidth == width {
		defer m.mu.RUnlock()
		return m.renderer, nil
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if m.renderer != nil && m.cachedWidth == width {
		return m.renderer, nil
	}

	style := m.buildStyle()
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStyles(style),
		glamour.WithWordWrap(width),
		glamour.WithEmoji(),
		glamour.WithColorProfile(termenv.TrueColor),
	)
	if err != nil {
		return nil, err
	}

	m.renderer = renderer
	m.cachedWidth = width
	return renderer, nil
}

// buildStyle creates a Glamour style config that matches the app theme.
func (m *MarkdownRenderer) buildStyle() ansi.StyleConfig {
	t := styles.CurrentTheme()

	// Start with the dark style as a base
	style := glamourstyles.DarkStyleConfig

	// Get theme colors as hex strings
	primaryHex := colorToHex(t.Primary)     // #5eb5f7 - Ocean blue
	secondaryHex := colorToHex(t.Secondary) // #7ec8e8 - Light sky blue
	accentHex := colorToHex(t.Accent)       // #8fd4f4 - Bright water
	mutedHex := colorToHex(t.FgMuted)       // #7a8b99 - Muted blue-gray
	subtleHex := colorToHex(t.FgSubtle)     // #4d5b66 - Subtle blue-gray
	baseHex := colorToHex(t.FgBase)         // #c5d1de - Soft white-blue

	// Headers - use accent and primary colors, remove # prefixes
	style.H1.Color = stringPtr(accentHex)
	style.H1.Bold = boolPtr(true)
	style.H1.Prefix = ""
	style.H1.Suffix = ""
	style.H2.Color = stringPtr(primaryHex)
	style.H2.Bold = boolPtr(true)
	style.H2.Prefix = ""
	style.H3.Color = stringPtr(secondaryHex)
	style.H3.Bold = boolPtr(true)
	style.H3.Prefix = ""
	style.H4.Color = stringPtr(secondaryHex)
	style.H4.Prefix = ""
	style.H5.Color = stringPtr(mutedHex)
	style.H5.Prefix = ""
	style.H6.Color = stringPtr(mutedHex)
	style.H6.Prefix = ""

	// Inline code
	style.Code.Color = stringPtr(secondaryHex)

	// Code blocks - customize syntax highlighting
	style.CodeBlock.Chroma.Text.Color = stringPtr(baseHex)
	style.CodeBlock.Chroma.Keyword.Color = stringPtr(primaryHex)
	style.CodeBlock.Chroma.Comment.Color = stringPtr(mutedHex)
	style.CodeBlock.Chroma.CommentPreproc.Color = stringPtr(mutedHex)
	style.CodeBlock.Chroma.Name.Color = stringPtr(baseHex)
	style.CodeBlock.Chroma.NameFunction.Color = stringPtr(accentHex)
	style.CodeBlock.Chroma.NameClass.Color = stringPtr(accentHex)
	style.CodeBlock.Chroma.Operator.Color = stringPtr(primaryHex)

	// Note: Some Chroma fields like KeywordConstant, KeywordDeclaration, String, Number
	// are not available in this version of glamour. They may be added in future versions.

	// Links
	style.Link.Color = stringPtr(primaryHex)
	style.Link.Underline = boolPtr(true)
	style.LinkText.Color = stringPtr(primaryHex)

	// Lists
	style.Item.BlockPrefix = "  "
	style.Enumeration.BlockPrefix = "  "

	// Block quotes
	style.BlockQuote.Color = stringPtr(mutedHex)
	style.BlockQuote.Italic = boolPtr(true)

	// Emphasis
	style.Emph.Italic = boolPtr(true)
	style.Strong.Bold = boolPtr(true)

	// Horizontal rule
	style.HorizontalRule.Color = stringPtr(subtleHex)

	// Tables
	style.Table.Color = stringPtr(baseHex)

	return style
}

// Helper functions for style configuration.
func stringPtr(s string) *string { return &s }
func boolPtr(b bool) *bool       { return &b }

// colorToHex converts a color.Color to hex string.
func colorToHex(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", r>>8, g>>8, b>>8)
}
