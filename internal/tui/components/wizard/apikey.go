//nolint:goconst // Key strings are standard keyboard identifiers.
package wizard

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/guilhermegouw/cdd/internal/config"
	"github.com/guilhermegouw/cdd/internal/tui/styles"
	"github.com/guilhermegouw/cdd/internal/tui/util"
)

// APIKeyEnteredMsg is sent when an API key is entered.
type APIKeyEnteredMsg struct {
	APIKey string
}

// APIKeyInput handles API key input from the user.
type APIKeyInput struct {
	providerName string
	input        textinput.Model
	width        int
	envVarMode   bool
}

// NewAPIKeyInput creates a new API key input component.
func NewAPIKeyInput(providerName string) *APIKeyInput {
	t := styles.CurrentTheme()

	ti := textinput.New()
	ti.Placeholder = "Enter API key or $ENV_VAR..."
	ti.Prompt = "> "
	ti.SetStyles(t.S().TextInput)
	ti.Focus()
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '*'

	return &APIKeyInput{
		input:        ti,
		providerName: providerName,
	}
}

// Init initializes the component.
func (a *APIKeyInput) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages.
func (a *APIKeyInput) Update(msg tea.Msg) (util.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case keyEnter:
			value := strings.TrimSpace(a.input.Value())
			if value != "" {
				return a, util.CmdHandler(APIKeyEnteredMsg{
					APIKey: value,
				})
			}
		case "tab":
			// Toggle between password mode and visible mode.
			if a.input.EchoMode == textinput.EchoPassword {
				a.input.EchoMode = textinput.EchoNormal
			} else {
				a.input.EchoMode = textinput.EchoPassword
			}
			return a, nil
		}
	}

	var cmd tea.Cmd
	a.input, cmd = a.input.Update(msg)

	// Check if input looks like an env var reference.
	a.envVarMode = strings.HasPrefix(a.input.Value(), "$")
	if a.envVarMode {
		a.input.EchoMode = textinput.EchoNormal
	}

	return a, cmd
}

// View renders the API key input.
func (a *APIKeyInput) View() string {
	t := styles.CurrentTheme()

	title := t.S().Title.Render(fmt.Sprintf("Enter %s API Key", a.providerName))

	inputView := a.input.View()

	// Help text.
	helpParts := []string{"Enter to confirm", "Tab to show/hide"}
	help := t.S().Muted.Render(strings.Join(helpParts, " | "))

	// Hint about env vars.
	hint := t.S().Subtle.Render("Tip: Use $ENV_VAR to reference an environment variable")

	// Config path info.
	configPath := config.GlobalConfigPath()
	configInfo := t.S().Muted.Render(fmt.Sprintf("Config will be saved to: %s", configPath))

	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		inputView,
		"",
		hint,
		"",
		help,
		"",
		configInfo,
	)
}

// Cursor returns the cursor position.
func (a *APIKeyInput) Cursor() *tea.Cursor {
	return a.input.Cursor()
}

// SetWidth sets the input width.
func (a *APIKeyInput) SetWidth(width int) {
	a.width = width
	a.input.SetWidth(width - 4)
}

// Value returns the current input value.
func (a *APIKeyInput) Value() string {
	return a.input.Value()
}

// SetProviderName updates the provider name.
func (a *APIKeyInput) SetProviderName(name string) {
	a.providerName = name
}

// Reset clears the input.
func (a *APIKeyInput) Reset() {
	a.input.SetValue("")
	a.input.Focus()
}
