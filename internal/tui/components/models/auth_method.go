package models

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/guilhermegouw/cdd/internal/tui/styles"
	"github.com/guilhermegouw/cdd/internal/tui/util"
)

// AuthMethodChooser lets the user choose between OAuth and API Key authentication.
type AuthMethodChooser struct {
	providerName string
	width        int
	useOAuth     bool // true = OAuth, false = API Key
}

// NewAuthMethodChooser creates a new auth method chooser.
func NewAuthMethodChooser() *AuthMethodChooser {
	return &AuthMethodChooser{
		useOAuth: true, // Default to OAuth
	}
}

// SetProvider sets the provider name for display.
func (a *AuthMethodChooser) SetProvider(name string) {
	a.providerName = name
	a.useOAuth = true // Reset to default
}

// SetSize sets the component width.
func (a *AuthMethodChooser) SetSize(width, _ int) {
	a.width = width
}

// Update handles messages.
func (a *AuthMethodChooser) Update(msg tea.Msg) (*AuthMethodChooser, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return a, nil
	}

	switch keyMsg.String() {
	case "left", "h":
		a.useOAuth = true
	case "right", "l":
		a.useOAuth = false
	case keyTab:
		a.useOAuth = !a.useOAuth
	case keyEnter:
		return a, util.CmdHandler(AuthMethodSelectedMsg{UseOAuth: a.useOAuth})
	}
	return a, nil
}

// View renders the auth method chooser.
func (a *AuthMethodChooser) View() string {
	t := styles.CurrentTheme()

	title := t.S().Text.Render("How would you like to authenticate?")

	// Fixed compact box dimensions
	boxWidth := 22
	boxHeight := 3

	// Style for boxes
	selectedBox := lipgloss.NewStyle().
		Width(boxWidth).
		Height(boxHeight).
		Align(lipgloss.Center).
		AlignVertical(lipgloss.Center).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Primary)

	unselectedBox := lipgloss.NewStyle().
		Width(boxWidth).
		Height(boxHeight).
		Align(lipgloss.Center).
		AlignVertical(lipgloss.Center).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.FgMuted)

	selectedText := t.S().Text.Bold(true)
	unselectedText := t.S().Muted

	var oauthBox, apiKeyBox string
	if a.useOAuth {
		oauthBox = selectedBox.Render(selectedText.Render("Claude Account"))
		apiKeyBox = unselectedBox.Render(unselectedText.Render("API Key"))
	} else {
		oauthBox = unselectedBox.Render(unselectedText.Render("Claude Account"))
		apiKeyBox = selectedBox.Render(selectedText.Render("API Key"))
	}

	boxes := lipgloss.JoinHorizontal(lipgloss.Center, oauthBox, " ", apiKeyBox)

	help := t.S().Muted.Render("[tab] or [left]/[right] to switch  [enter] select  [esc] back")

	return lipgloss.JoinVertical(lipgloss.Center,
		title,
		"",
		boxes,
		"",
		help,
	)
}
