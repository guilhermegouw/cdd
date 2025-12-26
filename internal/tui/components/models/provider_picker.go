//nolint:goconst // Key strings are standard keyboard identifiers.
package models

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/catwalk/pkg/catwalk"

	"github.com/guilhermegouw/cdd/internal/tui/styles"
	"github.com/guilhermegouw/cdd/internal/tui/util"
)

// ProviderPicker displays a list of providers to choose from.
type ProviderPicker struct {
	providers []catwalk.Provider
	cursor    int
	width     int
	height    int
}

// NewProviderPicker creates a new ProviderPicker.
func NewProviderPicker(providers []catwalk.Provider) *ProviderPicker {
	return &ProviderPicker{
		providers: providers,
		cursor:    0,
	}
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
			if p.cursor < len(p.providers)-1 {
				p.cursor++
			}
			return p, nil

		case keyEnter:
			if p.cursor >= 0 && p.cursor < len(p.providers) {
				provider := p.providers[p.cursor]
				return p, util.CmdHandler(ProviderSelectedMsg{
					ProviderID:   string(provider.ID),
					ProviderName: provider.Name,
					ProviderType: string(provider.Type),
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

	if len(p.providers) == 0 {
		return t.S().Muted.Render("No providers available.")
	}

	var sb strings.Builder

	sb.WriteString(t.S().Muted.Render("Select a provider:"))
	sb.WriteString("\n\n")

	for i := range p.providers {
		// Build line content.
		line := p.providers[i].Name + " (" + string(p.providers[i].Type) + ")"

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
