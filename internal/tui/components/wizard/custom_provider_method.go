// Package wizard provides custom provider wizard components.
package wizard

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/guilhermegouw/cdd/internal/tui/styles"
	"github.com/guilhermegouw/cdd/internal/tui/util"
)

// ProviderImportMethod represents the import method choice.
type ProviderImportMethod int

// ProviderImportMethod constants.
const (
	ProviderImportMethodManual ProviderImportMethod = iota
	ProviderImportMethodURL
	ProviderImportMethodFile
)

// CustomProviderMethodSelectedMsg is sent when a method is selected.
type CustomProviderMethodSelectedMsg struct {
	Method ProviderImportMethod
}

// CustomProviderMethod lets the user choose how to add a custom provider.
type CustomProviderMethod struct {
	width    int
	selected ProviderImportMethod
}

// NewCustomProviderMethod creates a new custom provider method chooser.
func NewCustomProviderMethod() *CustomProviderMethod {
	return &CustomProviderMethod{
		selected: ProviderImportMethodManual, // Default to manual.
	}
}

// Init initializes the component.
func (c *CustomProviderMethod) Init() tea.Cmd {
	return nil
}

// Update handles messages.
func (c *CustomProviderMethod) Update(msg tea.Msg) (util.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return c, nil
	}

	switch keyMsg.String() {
	case keyUp, keyK:
		if c.selected > 0 {
			c.selected--
		}
	case keyDown, keyJ:
		if c.selected < ProviderImportMethodFile {
			c.selected++
		}
	case keyEnter:
		return c, util.CmdHandler(CustomProviderMethodSelectedMsg{Method: c.selected})
	}
	return c, nil
}

// View renders the method chooser.
func (c *CustomProviderMethod) View() string {
	t := styles.CurrentTheme()

	title := t.S().Title.Render("How would you like to add a custom provider?")
	help := t.S().Muted.Render("Use ↑/↓ to navigate, Enter to select")

	items := []struct {
		label       string
		description string
		method      ProviderImportMethod
	}{
		{"Manual Definition", "Enter provider details manually", ProviderImportMethodManual},
		{"Import from URL", "Import from a URL (providers.json)", ProviderImportMethodURL},
		{"Import from File", "Import from a local file", ProviderImportMethodFile},
	}

	renderedItems := make([]string, len(items))
	for i, item := range items {
		cursor := "  "
		style := t.S().Text
		descStyle := t.S().Muted

		if i == int(c.selected) {
			cursor = t.S().Success.Render(styles.Selected + " ")
			style = t.S().Text.Bold(true)
			descStyle = t.S().Subtle
		}

		renderedItems[i] = cursor + style.Render(item.label) + "\n    " + descStyle.Render(item.description)
	}

	list := ""
	for _, item := range renderedItems {
		list += item + "\n"
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		list,
		"",
		help,
	)
}

// SetWidth sets the component width.
func (c *CustomProviderMethod) SetWidth(width int) {
	c.width = width
}
