package styles

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/rivo/uniseg"
)

// Theme defines the color scheme and styles for the TUI.
type Theme struct {
	BgOverlay   color.Color
	FgMuted     color.Color
	Primary     color.Color
	FgBase      color.Color
	Tertiary    color.Color
	Accent      color.Color
	BgBase      color.Color
	BgSubtle    color.Color
	Info        color.Color
	Warning     color.Color
	Secondary   color.Color
	FgSubtle    color.Color
	Border      color.Color
	BorderFocus color.Color
	Success     color.Color
	Error       color.Color
	styles      *Styles
	Name        string
	IsDark      bool
}

// Styles contains pre-built lipgloss styles.
type Styles struct {
	Base lipgloss.Style

	Title    lipgloss.Style
	Subtitle lipgloss.Style
	Text     lipgloss.Style
	Muted    lipgloss.Style
	Subtle   lipgloss.Style
	Primary  lipgloss.Style

	Success lipgloss.Style
	Error   lipgloss.Style
	Warning lipgloss.Style
	Info    lipgloss.Style

	TextSelection lipgloss.Style

	TextInput textinput.Styles
}

// S returns the pre-built styles, building them if necessary.
func (t *Theme) S() *Styles {
	if t.styles == nil {
		t.styles = t.buildStyles()
	}
	return t.styles
}

func (t *Theme) buildStyles() *Styles {
	base := lipgloss.NewStyle().Foreground(t.FgBase)

	return &Styles{
		Base: base,

		Title: base.
			Foreground(t.Accent).
			Bold(true),

		Subtitle: base.
			Foreground(t.Secondary).
			Bold(true),

		Text:    base,
		Muted:   base.Foreground(t.FgMuted),
		Subtle:  base.Foreground(t.FgSubtle),
		Primary: base.Foreground(t.Primary),

		Success: base.Foreground(t.Success),
		Error:   base.Foreground(t.Error),
		Warning: base.Foreground(t.Warning),
		Info:    base.Foreground(t.Info),

		TextSelection: lipgloss.NewStyle().
			Background(t.Tertiary).
			Foreground(t.Accent),

		TextInput: textinput.Styles{
			Focused: textinput.StyleState{
				Text:        base,
				Placeholder: base.Foreground(t.FgSubtle),
				Prompt:      base.Foreground(t.Primary),
				Suggestion:  base.Foreground(t.FgSubtle),
			},
			Blurred: textinput.StyleState{
				Text:        base.Foreground(t.FgMuted),
				Placeholder: base.Foreground(t.FgSubtle),
				Prompt:      base.Foreground(t.FgMuted),
				Suggestion:  base.Foreground(t.FgSubtle),
			},
			Cursor: textinput.CursorStyle{
				Color: t.Primary,
				Shape: tea.CursorBlock,
				Blink: true,
			},
		},
	}
}

// Manager manages theme instances.
type Manager struct {
	themes  map[string]*Theme
	current *Theme
}

var defaultManager *Manager

// SetDefaultManager sets the default theme manager.
func SetDefaultManager(m *Manager) {
	defaultManager = m
}

// DefaultManager returns the default theme manager.
func DefaultManager() *Manager {
	if defaultManager == nil {
		defaultManager = NewManager()
	}
	return defaultManager
}

// CurrentTheme returns the current theme.
func CurrentTheme() *Theme {
	return DefaultManager().Current()
}

// NewManager creates a new theme manager with the default theme.
func NewManager() *Manager {
	m := &Manager{
		themes: make(map[string]*Theme),
	}

	t := NewDefaultTheme()
	m.Register(t)
	m.current = m.themes[t.Name]

	return m
}

// Register adds a theme to the manager.
func (m *Manager) Register(theme *Theme) {
	m.themes[theme.Name] = theme
}

// Current returns the current theme.
func (m *Manager) Current() *Theme {
	return m.current
}

// SetTheme sets the current theme by name.
func (m *Manager) SetTheme(name string) error {
	if theme, ok := m.themes[name]; ok {
		m.current = theme
		return nil
	}
	return fmt.Errorf("theme %s not found", name)
}

// ParseHex converts hex string to color.
func ParseHex(hex string) color.Color {
	var r, g, b uint8
	//nolint:errcheck // Simple hex parsing; invalid input returns black.
	fmt.Sscanf(hex, "#%02x%02x%02x", &r, &g, &b)
	return color.RGBA{R: r, G: g, B: b, A: 255}
}

// ForegroundGrad creates a gradient across the string.
func ForegroundGrad(input string, bold bool, color1, color2 color.Color) []string {
	if input == "" {
		return []string{""}
	}
	t := CurrentTheme()
	if len(input) == 1 {
		style := t.S().Base.Foreground(color1)
		if bold {
			style = style.Bold(true)
		}
		return []string{style.Render(input)}
	}

	var clusters []string
	gr := uniseg.NewGraphemes(input)
	for gr.Next() {
		clusters = append(clusters, string(gr.Runes()))
	}

	ramp := blendColors(len(clusters), color1, color2)
	for i, c := range ramp {
		style := t.S().Base.Foreground(c)
		if bold {
			style = style.Bold(true)
		}
		clusters[i] = style.Render(clusters[i])
	}
	return clusters
}

// ApplyForegroundGrad renders a string with a horizontal gradient foreground.
func ApplyForegroundGrad(input string, color1, color2 color.Color) string {
	if input == "" {
		return ""
	}
	var o strings.Builder
	clusters := ForegroundGrad(input, false, color1, color2)
	for _, c := range clusters {
		fmt.Fprint(&o, c)
	}
	return o.String()
}

// ApplyBoldForegroundGrad renders a string with a bold horizontal gradient foreground.
func ApplyBoldForegroundGrad(input string, color1, color2 color.Color) string {
	if input == "" {
		return ""
	}
	var o strings.Builder
	clusters := ForegroundGrad(input, true, color1, color2)
	for _, c := range clusters {
		fmt.Fprint(&o, c)
	}
	return o.String()
}

// blendColors returns a slice of colors blended between the given stops.
func blendColors(size int, stops ...color.Color) []color.Color {
	if len(stops) < 2 {
		return nil
	}

	stopsPrime := make([]colorful.Color, len(stops))
	for i, k := range stops {
		stopsPrime[i], _ = colorful.MakeColor(k)
	}

	numSegments := len(stopsPrime) - 1
	blended := make([]color.Color, 0, size)

	segmentSizes := make([]int, numSegments)
	baseSize := size / numSegments
	remainder := size % numSegments

	for i := range numSegments {
		segmentSizes[i] = baseSize
		if i < remainder {
			segmentSizes[i]++
		}
	}

	for i := range numSegments {
		c1 := stopsPrime[i]
		c2 := stopsPrime[i+1]
		segmentSize := segmentSizes[i]

		for j := range segmentSize {
			var t float64
			if segmentSize > 1 {
				t = float64(j) / float64(segmentSize-1)
			}
			c := c1.BlendHcl(c2, t)
			blended = append(blended, c)
		}
	}

	return blended
}
