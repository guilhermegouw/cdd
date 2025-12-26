package wizard

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/catwalk/pkg/catwalk"

	"github.com/guilhermegouw/cdd/internal/tui/styles"
	"github.com/guilhermegouw/cdd/internal/tui/util"
)

// ProviderSelectedMsg is sent when a provider is selected.
type ProviderSelectedMsg struct {
	Provider catwalk.Provider
}

// ProviderList displays a list of providers to select from.
type ProviderList struct {
	providers []catwalk.Provider
	cursor    int
	width     int
	height    int
}

// NewProviderList creates a new provider list component.
func NewProviderList(providers []catwalk.Provider) *ProviderList {
	return &ProviderList{
		providers: providers,
		cursor:    0,
	}
}

// AddCustomProviderOption adds a "Add Custom Provider" option to the list.
func AddCustomProviderOption(providers []catwalk.Provider) []catwalk.Provider {
	// Create a pseudo-provider for the custom option.
	customOption := catwalk.Provider{
		ID:   catwalk.InferenceProvider("custom"),
		Name: "➕ Add Custom Provider",
		Type: catwalk.TypeOpenAICompat, // Arbitrary type for display
	}
	return append(providers, customOption)
}

// Init initializes the component.
func (p *ProviderList) Init() tea.Cmd {
	return nil
}

// Update handles messages.
func (p *ProviderList) Update(msg tea.Msg) (util.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return p, nil
	}

	switch keyMsg.String() {
	case keyUp, keyK:
		if p.cursor > 0 {
			p.cursor--
		}
	case keyDown, keyJ:
		if p.cursor < len(p.providers)-1 {
			p.cursor++
		}
	case keyEnter:
		if len(p.providers) > 0 {
			return p, util.CmdHandler(ProviderSelectedMsg{
				Provider: p.providers[p.cursor],
			})
		}
	}
	return p, nil
}

// View renders the provider list.
func (p *ProviderList) View() string {
	t := styles.CurrentTheme()

	title := t.S().Title.Render("Select a Provider")
	help := t.S().Muted.Render("Use ↑/↓ to navigate, Enter to select")

	items := make([]string, 0, len(p.providers))
	for i := range p.providers {
		cursor := "  "
		style := t.S().Text

		if i == p.cursor {
			cursor = t.S().Success.Render(styles.Selected + " ")
			style = t.S().Text.Bold(true)
		}

		name := style.Render(p.providers[i].Name)
		desc := t.S().Muted.Render(fmt.Sprintf(" (%s)", p.providers[i].Type))
		items = append(items, cursor+name+desc)
	}

	list := strings.Join(items, "\n")

	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		list,
		"",
		help,
	)
}

// SetSize sets the component size.
func (p *ProviderList) SetSize(width, height int) {
	p.width = width
	p.height = height
}

// SelectedProvider returns the currently selected provider.
func (p *ProviderList) SelectedProvider() *catwalk.Provider {
	if len(p.providers) == 0 {
		return nil
	}
	return &p.providers[p.cursor]
}
