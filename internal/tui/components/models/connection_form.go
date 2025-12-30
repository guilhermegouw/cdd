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

// FormField constants.
const (
	FieldName FormField = iota
	FieldBaseURL
	FieldModelID
	FieldAPIKey
)

// ConnectionForm is the form for adding/editing connections.
type ConnectionForm struct {
	nameInput    textinput.Model
	baseURLInput textinput.Model
	modelIDInput textinput.Model
	apiKeyInput  textinput.Model
	focused      FormField
	providerID   string
	providerName string
	providerType string
	isCustom     bool
	width        int
	height       int
	isEdit       bool
	editID       string
}

// NewConnectionForm creates a new ConnectionForm.
func NewConnectionForm() *ConnectionForm {
	nameInput := textinput.New()
	nameInput.Placeholder = "Connection name"
	nameInput.CharLimit = 50
	nameInput.Prompt = ""

	baseURLInput := textinput.New()
	baseURLInput.Placeholder = "https://api.example.com/v1"
	baseURLInput.CharLimit = 200
	baseURLInput.Prompt = ""

	modelIDInput := textinput.New()
	modelIDInput.Placeholder = "model-id"
	modelIDInput.CharLimit = 100
	modelIDInput.Prompt = ""

	apiKeyInput := textinput.New()
	apiKeyInput.Placeholder = "API key or $ENV_VAR"
	apiKeyInput.CharLimit = 2000 // JWT tokens can be very long
	apiKeyInput.EchoMode = textinput.EchoPassword
	apiKeyInput.Prompt = ""

	return &ConnectionForm{
		nameInput:    nameInput,
		baseURLInput: baseURLInput,
		modelIDInput: modelIDInput,
		apiKeyInput:  apiKeyInput,
		focused:      FieldName,
	}
}

// Reset clears the form.
func (f *ConnectionForm) Reset() {
	f.nameInput.Reset()
	f.nameInput.Blur()
	f.baseURLInput.Reset()
	f.baseURLInput.Blur()
	f.modelIDInput.Reset()
	f.modelIDInput.Blur()
	f.apiKeyInput.Reset()
	f.apiKeyInput.Blur()
	f.focused = FieldName
	f.providerID = ""
	f.providerName = ""
	f.providerType = ""
	f.isCustom = false
	f.isEdit = false
	f.editID = ""
}

// SetProvider sets the provider for a new connection.
func (f *ConnectionForm) SetProvider(id, name, providerType string, isCustom bool) {
	f.providerID = id
	f.providerName = name
	f.providerType = providerType
	f.isCustom = isCustom
	f.isEdit = false

	// For custom providers, clear the name so user enters their own.
	// For standard providers, default to provider name.
	if isCustom {
		f.nameInput.SetValue("")
	} else {
		f.nameInput.SetValue(name)
	}
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
		case "tab", keyDown:
			return f.nextField()
		case "shift+tab", "up":
			return f.prevField()
		case keyEnter:
			// Submit if on last field (API key).
			if f.focused == FieldAPIKey {
				return f.submit()
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
	case FieldBaseURL:
		f.baseURLInput, cmd = f.baseURLInput.Update(msg)
	case FieldModelID:
		f.modelIDInput, cmd = f.modelIDInput.Update(msg)
	case FieldAPIKey:
		f.apiKeyInput, cmd = f.apiKeyInput.Update(msg)
	}
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return f, tea.Batch(cmds...)
}

func (f *ConnectionForm) submit() (*ConnectionForm, tea.Cmd) {
	name := strings.TrimSpace(f.nameInput.Value())
	apiKey := strings.TrimSpace(f.apiKeyInput.Value())

	if name == "" {
		return f, util.ReportWarn("Connection name is required")
	}
	if apiKey == "" {
		return f, util.ReportWarn("API key is required")
	}

	// For custom providers, require base URL and model ID.
	var baseURL, modelID string
	if f.isCustom {
		baseURL = strings.TrimSpace(f.baseURLInput.Value())
		modelID = strings.TrimSpace(f.modelIDInput.Value())

		if baseURL == "" {
			return f, util.ReportWarn("API endpoint URL is required")
		}
		if modelID == "" {
			return f, util.ReportWarn("Model ID is required")
		}
	}

	return f, util.CmdHandler(FormSubmitMsg{
		Name:     name,
		APIKey:   apiKey,
		BaseURL:  baseURL,
		ModelID:  modelID,
		IsCustom: f.isCustom,
	})
}

//nolint:dupl // nextField and prevField are intentionally similar but serve different purposes.
func (f *ConnectionForm) nextField() (*ConnectionForm, tea.Cmd) {
	switch f.focused {
	case FieldName:
		f.nameInput.Blur()
		if f.isCustom {
			f.focused = FieldBaseURL
			return f, f.baseURLInput.Focus()
		}
		f.focused = FieldAPIKey
		return f, f.apiKeyInput.Focus()
	case FieldBaseURL:
		f.baseURLInput.Blur()
		f.focused = FieldModelID
		return f, f.modelIDInput.Focus()
	case FieldModelID:
		f.modelIDInput.Blur()
		f.focused = FieldAPIKey
		return f, f.apiKeyInput.Focus()
	case FieldAPIKey:
		// Stay on API key - submit handled separately.
		return f, nil
	}
	return f, nil
}

//nolint:dupl // prevField and nextField are intentionally similar but serve different purposes.
func (f *ConnectionForm) prevField() (*ConnectionForm, tea.Cmd) {
	switch f.focused {
	case FieldAPIKey:
		f.apiKeyInput.Blur()
		if f.isCustom {
			f.focused = FieldModelID
			return f, f.modelIDInput.Focus()
		}
		f.focused = FieldName
		return f, f.nameInput.Focus()
	case FieldModelID:
		f.modelIDInput.Blur()
		f.focused = FieldBaseURL
		return f, f.baseURLInput.Focus()
	case FieldBaseURL:
		f.baseURLInput.Blur()
		f.focused = FieldName
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

	// Custom provider fields.
	if f.isCustom {
		// Base URL field.
		if f.focused == FieldBaseURL {
			sb.WriteString(t.S().Primary.Bold(true).Render("API Endpoint"))
		} else {
			sb.WriteString(t.S().Text.Render("API Endpoint"))
		}
		sb.WriteString("\n")
		sb.WriteString("  ")
		sb.WriteString(f.baseURLInput.View())
		sb.WriteString("\n\n")

		// Model ID field.
		if f.focused == FieldModelID {
			sb.WriteString(t.S().Primary.Bold(true).Render("Model ID"))
		} else {
			sb.WriteString(t.S().Text.Render("Model ID"))
		}
		sb.WriteString("\n")
		sb.WriteString("  ")
		sb.WriteString(f.modelIDInput.View())
		sb.WriteString("\n\n")
	}

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
	case FieldBaseURL:
		return f.baseURLInput.Cursor()
	case FieldModelID:
		return f.modelIDInput.Cursor()
	case FieldAPIKey:
		return f.apiKeyInput.Cursor()
	}
	return nil
}
