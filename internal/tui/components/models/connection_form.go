package models

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/guilhermegouw/cdd/internal/config"
	"github.com/guilhermegouw/cdd/internal/tui/styles"
	"github.com/guilhermegouw/cdd/internal/tui/util"
)

// FormField indicates which field is focused.
type FormField int

const (
	FieldName FormField = iota
	FieldAPIKey
)

// ConnectionForm is the form for adding/editing connections.
type ConnectionForm struct {
	nameInput   textinput.Model
	apiKeyInput textinput.Model
	focused     FormField
	providerID  string
	providerName string
	providerType string
	width       int
	height      int
	isEdit      bool
	editID      string
}

// NewConnectionForm creates a new ConnectionForm.
func NewConnectionForm() *ConnectionForm {
	nameInput := textinput.New()
	nameInput.Placeholder = "Connection name"
	nameInput.CharLimit = 50
	nameInput.Prompt = ""

	apiKeyInput := textinput.New()
	apiKeyInput.Placeholder = "API key or $ENV_VAR"
	apiKeyInput.CharLimit = 200
	apiKeyInput.EchoMode = textinput.EchoPassword
	apiKeyInput.Prompt = ""

	return &ConnectionForm{
		nameInput:   nameInput,
		apiKeyInput: apiKeyInput,
		focused:     FieldName,
	}
}

// Reset clears the form.
func (f *ConnectionForm) Reset() {
	f.nameInput.Reset()
	f.nameInput.Blur()
	f.apiKeyInput.Reset()
	f.apiKeyInput.Blur()
	f.focused = FieldName
	f.providerID = ""
	f.providerName = ""
	f.providerType = ""
	f.isEdit = false
	f.editID = ""
}

// SetProvider sets the provider for a new connection.
func (f *ConnectionForm) SetProvider(id, name, providerType string) {
	f.providerID = id
	f.providerName = name
	f.providerType = providerType
	f.isEdit = false

	// Default connection name to provider name.
	f.nameInput.SetValue(name)
}

// SetConnection sets up the form for editing an existing connection.
func (f *ConnectionForm) SetConnection(conn *config.Connection) {
	f.isEdit = true
	f.editID = conn.ID
	f.providerID = conn.ProviderID
	f.providerName = conn.ProviderID // We don't have the name, use ID
	f.nameInput.SetValue(conn.Name)
	f.apiKeyInput.SetValue(conn.APIKey)
	f.focused = FieldName
}

// Focus focuses the first input.
func (f *ConnectionForm) Focus() tea.Cmd {
	return f.nameInput.Focus()
}

// SetSize sets the component size.
func (f *ConnectionForm) SetSize(width, height int) {
	f.width = width
	f.height = height
}

// Update handles messages.
func (f *ConnectionForm) Update(msg tea.Msg) (*ConnectionForm, tea.Cmd) {
	var cmds []tea.Cmd

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "tab", "down":
			return f.nextField()
		case "shift+tab", "up":
			return f.prevField()
		case "enter":
			// Submit if on API key field.
			if f.focused == FieldAPIKey {
				name := strings.TrimSpace(f.nameInput.Value())
				apiKey := strings.TrimSpace(f.apiKeyInput.Value())

				if name == "" {
					return f, util.ReportWarn("Connection name is required")
				}
				if apiKey == "" {
					return f, util.ReportWarn("API key is required")
				}

				return f, util.CmdHandler(FormSubmitMsg{
					Name:   name,
					APIKey: apiKey,
				})
			}
			// Move to next field.
			return f.nextField()
		}
	}

	// Update focused input.
	var cmd tea.Cmd
	switch f.focused {
	case FieldName:
		f.nameInput, cmd = f.nameInput.Update(msg)
	case FieldAPIKey:
		f.apiKeyInput, cmd = f.apiKeyInput.Update(msg)
	}
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return f, tea.Batch(cmds...)
}

func (f *ConnectionForm) nextField() (*ConnectionForm, tea.Cmd) {
	switch f.focused {
	case FieldName:
		f.focused = FieldAPIKey
		f.nameInput.Blur()
		return f, f.apiKeyInput.Focus()
	case FieldAPIKey:
		// Stay on API key or submit.
		return f, nil
	}
	return f, nil
}

func (f *ConnectionForm) prevField() (*ConnectionForm, tea.Cmd) {
	switch f.focused {
	case FieldAPIKey:
		f.focused = FieldName
		f.apiKeyInput.Blur()
		return f, f.nameInput.Focus()
	case FieldName:
		return f, nil
	}
	return f, nil
}

// View renders the form.
func (f *ConnectionForm) View() string {
	t := styles.CurrentTheme()

	var sb strings.Builder

	// Provider info.
	if f.providerName != "" {
		sb.WriteString(t.S().Muted.Render("Provider: "))
		sb.WriteString(t.S().Primary.Render(f.providerName))
		sb.WriteString("\n\n")
	}

	// Name field.
	if f.focused == FieldName {
		sb.WriteString(t.S().Primary.Bold(true).Render("Name"))
	} else {
		sb.WriteString(t.S().Text.Render("Name"))
	}
	sb.WriteString("\n")
	sb.WriteString("  ")
	sb.WriteString(f.nameInput.View())
	sb.WriteString("\n\n")

	// API Key field.
	if f.focused == FieldAPIKey {
		sb.WriteString(t.S().Primary.Bold(true).Render("API Key"))
	} else {
		sb.WriteString(t.S().Text.Render("API Key"))
	}
	sb.WriteString("\n")
	sb.WriteString("  ")
	sb.WriteString(f.apiKeyInput.View())
	sb.WriteString("\n\n")

	// Hint about env vars.
	sb.WriteString(t.S().Muted.Render("Tip: Use $ENV_VAR to reference environment variables"))
	sb.WriteString("\n\n")

	// Help.
	sb.WriteString(t.S().Muted.Render("[tab] next field  [enter] submit  [esc] cancel"))

	return sb.String()
}

// Cursor returns the cursor for the focused input.
func (f *ConnectionForm) Cursor() *tea.Cursor {
	switch f.focused {
	case FieldName:
		return f.nameInput.Cursor()
	case FieldAPIKey:
		return f.apiKeyInput.Cursor()
	}
	return nil
}
