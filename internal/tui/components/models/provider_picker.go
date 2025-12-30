//nolint:goconst // Key strings are standard keyboard identifiers.
package models

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/catwalk/pkg/catwalk"

	"github.com/guilhermegouw/cdd/internal/tui/styles"
	"github.com/guilhermegouw/cdd/internal/tui/util"
)

// Custom provider type constants.
const (
	CustomProviderOpenAI    = "custom-openai"
	CustomProviderAnthropic = "custom-anthropic"
)

// ProviderOption represents a selectable provider option.
type ProviderOption struct {
	ID       string
	Name     string
	Type     string
	IsCustom bool
}

// ProviderPicker displays a list of providers to choose from.
type ProviderPicker struct {
	providers []catwalk.Provider
	options   []ProviderOption
	cursor    int
	width     int
	height    int
}

// NewProviderPicker creates a new ProviderPicker.
func NewProviderPicker(providers []catwalk.Provider) *ProviderPicker {
	p := &ProviderPicker{
		providers: providers,
		cursor:    0,
	}
	p.buildOptions()
	return p
}

// buildOptions builds the list of provider options including custom providers.
func (p *ProviderPicker) buildOptions() {
	p.options = make([]ProviderOption, 0, len(p.providers)+2)

	// Add standard providers.
	for i := range p.providers {
		p.options = append(p.options, ProviderOption{
			ID:       string(p.providers[i].ID),
			Name:     p.providers[i].Name,
			Type:     string(p.providers[i].Type),
			IsCustom: false,
		})
	}

	// Add custom provider options at the end.
	p.options = append(p.options,
		ProviderOption{
			ID:       CustomProviderOpenAI,
			Name:     "Custom (OpenAI-compatible)",
			Type:     "openai-compat",
			IsCustom: true,
		},
		ProviderOption{
			ID:       CustomProviderAnthropic,
			Name:     "Custom (Anthropic-compatible)",
			Type:     "anthropic",
			IsCustom: true,
		},
	)
}

// Reset resets the cursor to the beginning.
func (p *ProviderPicker) Reset() {
	p.cursor = 0
}

// SetSize sets the component size.
func (p *ProviderPicker) SetSize(width, height int) {
	p.width = width
	p.height = height
}

// Update handles messages.
func (p *ProviderPicker) Update(msg tea.Msg) (*ProviderPicker, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "up", "k":
			if p.cursor > 0 {
				p.cursor--
			}
			return p, nil

		case keyDown, "j":
			if p.cursor < len(p.options)-1 {
				p.cursor++
			}
			return p, nil

		case keyEnter:
			if p.cursor >= 0 && p.cursor < len(p.options) {
				opt := p.options[p.cursor]
				return p, util.CmdHandler(ProviderSelectedMsg{
					ProviderID:   opt.ID,
					ProviderName: opt.Name,
					ProviderType: opt.Type,
					IsCustom:     opt.IsCustom,
				})
			}
			return p, nil
		}
	}

	return p, nil
}

// View renders the provider list.
func (p *ProviderPicker) View() string {
	t := styles.CurrentTheme()

	if len(p.options) == 0 {
		return t.S().Muted.Render("No providers available.")
	}

	var sb strings.Builder

	sb.WriteString(t.S().Muted.Render("Select a provider:"))
	sb.WriteString("\n\n")

	for i := range p.options {
		opt := p.options[i]

		// Build line content.
		var line string
		if opt.IsCustom {
			line = opt.Name
		} else {
			line = opt.Name + " (" + opt.Type + ")"
		}

		// Render with cursor and styling.
		if i == p.cursor {
			sb.WriteString("> ")
			sb.WriteString(t.S().Primary.Bold(true).Render(line))
		} else {
			sb.WriteString("  ")
			sb.WriteString(t.S().Muted.Render(line))
		}
		sb.WriteString("\n")
	}

	// Add help.
	sb.WriteString("\n")
	sb.WriteString(t.S().Muted.Render("[enter] select  [esc] cancel"))

	return sb.String()
}

// Selected returns the currently selected provider.
func (p *ProviderPicker) Selected() *catwalk.Provider {
	if p.cursor >= 0 && p.cursor < len(p.providers) {
		return &p.providers[p.cursor]
	}
	return nil
}
