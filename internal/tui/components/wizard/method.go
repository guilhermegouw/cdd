package wizard

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/guilhermegouw/cdd/internal/tui/styles"
	"github.com/guilhermegouw/cdd/internal/tui/util"
)

// AuthMethod represents the authentication method choice.
type AuthMethod int

// Authentication methods.
const (
	AuthMethodOAuth2 AuthMethod = iota
	AuthMethodAPIKey
)

// AuthMethodSelectedMsg is sent when an auth method is selected.
type AuthMethodSelectedMsg struct {
	Method AuthMethod
}

// AuthMethodChooser lets the user choose between OAuth and API Key authentication.
type AuthMethodChooser struct {
	providerName string
	width        int
	selected     AuthMethod
}

// NewAuthMethodChooser creates a new auth method chooser.
func NewAuthMethodChooser(providerName string) *AuthMethodChooser {
	return &AuthMethodChooser{
		selected:     AuthMethodOAuth2, // Default to OAuth.
		providerName: providerName,
	}
}

// Init initializes the component.
func (a *AuthMethodChooser) Init() tea.Cmd {
	return nil
}

// Update handles messages.
func (a *AuthMethodChooser) Update(msg tea.Msg) (util.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return a, nil
	}

	switch keyMsg.String() {
	case "left", "h":
		a.selected = AuthMethodOAuth2
	case "right", "l":
		a.selected = AuthMethodAPIKey
	case "tab":
		a.toggleChoice()
	case keyEnter:
		return a, util.CmdHandler(AuthMethodSelectedMsg{Method: a.selected})
	}
	return a, nil
}

// View renders the auth method chooser.
func (a *AuthMethodChooser) View() string {
	t := styles.CurrentTheme()

	title := t.S().Title.Render("How would you like to authenticate?")

	// Fixed compact box dimensions.
	boxWidth := 24
	boxHeight := 3

	// Style for boxes.
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
	if a.selected == AuthMethodOAuth2 {
		oauthBox = selectedBox.Render(selectedText.Render("Claude Account"))
		apiKeyBox = unselectedBox.Render(unselectedText.Render("API Key"))
	} else {
		oauthBox = unselectedBox.Render(unselectedText.Render("Claude Account"))
		apiKeyBox = selectedBox.Render(selectedText.Render("API Key"))
	}

	boxes := lipgloss.JoinHorizontal(lipgloss.Center, oauthBox, " ", apiKeyBox)

	help := t.S().Muted.Render("Tab or ←/→ to switch • Enter to select")

	return lipgloss.JoinVertical(lipgloss.Center,
		title,
		"",
		boxes,
		"",
		help,
	)
}

// SetWidth sets the component width.
func (a *AuthMethodChooser) SetWidth(w int) {
	a.width = w
}

func (a *AuthMethodChooser) toggleChoice() {
	if a.selected == AuthMethodOAuth2 {
		a.selected = AuthMethodAPIKey
	} else {
		a.selected = AuthMethodOAuth2
	}
}
