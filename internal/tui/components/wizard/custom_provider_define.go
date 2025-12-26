// Package wizard provides custom provider wizard components.
//
//nolint:goconst,gocritic // Key strings are standard keyboard identifiers; appendCombine is stylistic preference.
package wizard

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/catwalk/pkg/catwalk"

	"github.com/guilhermegouw/cdd/internal/config"
	"github.com/guilhermegouw/cdd/internal/tui/styles"
	"github.com/guilhermegouw/cdd/internal/tui/util"
)

// CustomProviderDefinedMsg is sent when custom provider definition is complete.
type CustomProviderDefinedMsg struct {
	Provider config.CustomProvider
}

// CustomProviderDefine handles manual custom provider definition.
type CustomProviderDefine struct {
	// Input fields.
	nameInput        textinput.Model
	idInput          textinput.Model
	typeInput        textinput.Model
	apiEndpointInput textinput.Model

	// Provider types for selection.
	providerTypes []catwalk.Type
	typeIndex     int

	// Headers.
	headers        map[string]string
	headerKeyInput textinput.Model
	headerValInput textinput.Model
	headerMode     bool // true when adding a header
	headerIndex    int  // for editing existing headers

	width int
	step  int // 0: name, 1: id, 2: type, 3: endpoint, 4: headers, 5: confirm
}

// NewCustomProviderDefine creates a new custom provider definition component.
func NewCustomProviderDefine() *CustomProviderDefine {
	t := styles.CurrentTheme()

	// Initialize all inputs.
	nameInput := textinput.New()
	nameInput.Placeholder = "My Custom Provider"
	nameInput.Prompt = "> "
	nameInput.SetStyles(t.S().TextInput)
	nameInput.Focus()

	idInput := textinput.New()
	idInput.Placeholder = "my-custom-provider"
	idInput.Prompt = "> "
	idInput.SetStyles(t.S().TextInput)
	idInput.CharLimit = 50

	typeInput := textinput.New()
	typeInput.Prompt = "> "
	typeInput.SetStyles(t.S().TextInput)
	typeInput.SetValue("openai-compat")
	typeInput.CharLimit = 20

	apiEndpointInput := textinput.New()
	apiEndpointInput.Placeholder = "https://api.example.com/v1"
	apiEndpointInput.Prompt = "> "
	apiEndpointInput.SetStyles(t.S().TextInput)

	headerKeyInput := textinput.New()
	headerKeyInput.Placeholder = "Header name"
	headerKeyInput.Prompt = "> "
	headerKeyInput.SetStyles(t.S().TextInput)

	headerValInput := textinput.New()
	headerValInput.Placeholder = "Header value"
	headerValInput.Prompt = "> "
	headerValInput.SetStyles(t.S().TextInput)

	providerTypes := []catwalk.Type{
		catwalk.TypeOpenAICompat,
		catwalk.TypeOpenAI,
		catwalk.TypeAnthropic,
		catwalk.TypeGoogle,
		catwalk.TypeAzure,
		catwalk.TypeBedrock,
		catwalk.TypeVertexAI,
		catwalk.TypeOpenRouter,
	}

	return &CustomProviderDefine{
		nameInput:        nameInput,
		idInput:          idInput,
		typeInput:        typeInput,
		apiEndpointInput: apiEndpointInput,
		headerKeyInput:   headerKeyInput,
		headerValInput:   headerValInput,
		providerTypes:    providerTypes,
		typeIndex:        0,
		headers:          make(map[string]string),
		width:            60,
		step:             0,
	}
}

// Init initializes the component.
func (c *CustomProviderDefine) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages.
//
//nolint:gocyclo // TUI update handler requires handling many message types
func (c *CustomProviderDefine) Update(msg tea.Msg) (util.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return c, nil
	}

	switch keyMsg.String() {
	case keyEnter:
		return c.handleEnter()
	case "tab":
		return c.handleTab()
	case keyUp:
		if c.step == 2 { // Type selection
			if c.typeIndex > 0 {
				c.typeIndex--
			}
			c.typeInput.SetValue(string(c.providerTypes[c.typeIndex]))
		}
	case keyDown:
		if c.step == 2 { // Type selection
			if c.typeIndex < len(c.providerTypes)-1 {
				c.typeIndex++
			}
			c.typeInput.SetValue(string(c.providerTypes[c.typeIndex]))
		}
	case "esc":
		// Cancel - go back without saving.
		return c, nil
	}

	// Update focused input.
	var cmd tea.Cmd
	switch c.step {
	case 0:
		c.nameInput, cmd = c.nameInput.Update(msg)
	case 1:
		c.idInput, cmd = c.idInput.Update(msg)
	case 2:
		c.typeInput, cmd = c.typeInput.Update(msg)
	case 3:
		c.apiEndpointInput, cmd = c.apiEndpointInput.Update(msg)
	case 4:
		if c.headerMode {
			if c.headerIndex == 0 { // Key
				c.headerKeyInput, cmd = c.headerKeyInput.Update(msg)
			} else { // Value
				c.headerValInput, cmd = c.headerValInput.Update(msg)
			}
		}
	}

	return c, cmd
}

func (c *CustomProviderDefine) handleEnter() (util.Model, tea.Cmd) {
	switch c.step {
	case 0: // Name
		if strings.TrimSpace(c.nameInput.Value()) != "" {
			c.step = 1
			c.idInput.Focus()
			c.nameInput.Blur()
			// Auto-generate ID from name.
			if c.idInput.Value() == "" {
				generateID := strings.ToLower(strings.TrimSpace(c.nameInput.Value()))
				generateID = strings.ReplaceAll(generateID, " ", "-")
				generateID = strings.ReplaceAll(generateID, "_", "-")
				// Remove non-alphanumeric chars except hyphen.
				c.idInput.SetValue(generateID)
			}
		}
	case 1: // ID
		if strings.TrimSpace(c.idInput.Value()) != "" {
			c.step = 2
			c.typeInput.Focus()
			c.idInput.Blur()
		}
	case 2: // Type
		c.step = 3
		c.apiEndpointInput.Focus()
		c.typeInput.Blur()
	case 3: // API Endpoint
		c.step = 4
		c.headerMode = true
		c.headerIndex = 0
		c.headerKeyInput.Focus()
		c.apiEndpointInput.Blur()
	case 4: // Headers
		if c.headerMode {
			if c.headerIndex == 0 {
				// Moving to value input.
				if key := strings.TrimSpace(c.headerKeyInput.Value()); key != "" {
					c.headerIndex = 1
					c.headerKeyInput.Blur()
					c.headerValInput.Focus()
				}
			} else {
				// Save header and go back to key.
				key := strings.TrimSpace(c.headerKeyInput.Value())
				val := strings.TrimSpace(c.headerValInput.Value())
				if key != "" {
					c.headers[key] = val
				}
				c.headerKeyInput.SetValue("")
				c.headerValInput.SetValue("")
				c.headerIndex = 0
				c.headerKeyInput.Focus()
				c.headerValInput.Blur()
			}
		}
	case 5: // Confirm
		// Build the provider and finish.
		provider := c.buildProvider()
		return c, util.CmdHandler(CustomProviderDefinedMsg{Provider: provider})
	}
	return c, nil
}

func (c *CustomProviderDefine) handleTab() (util.Model, tea.Cmd) {
	if c.step == 4 && c.headerMode {
		// Skip headers and move to confirm.
		c.headerMode = false
		c.step = 5
		c.headerKeyInput.Blur()
		c.headerValInput.Blur()
	} else if c.step == 5 {
		// Go back to headers.
		c.step = 4
		c.headerMode = true
		c.headerIndex = 0
		c.headerKeyInput.Focus()
	}
	return c, nil
}

func (c *CustomProviderDefine) buildProvider() config.CustomProvider {
	providerType := catwalk.Type(c.typeInput.Value())

	return config.CustomProvider{
		Name:           strings.TrimSpace(c.nameInput.Value()),
		ID:             strings.TrimSpace(c.idInput.Value()),
		Type:           providerType,
		APIEndpoint:    strings.TrimSpace(c.apiEndpointInput.Value()),
		DefaultHeaders: c.headers,
		Models:         []catwalk.Model{}, // Models will be added in next step.
	}
}

// View renders the custom provider definition.
func (c *CustomProviderDefine) View() string {
	t := styles.CurrentTheme()

	title := t.S().Title.Render("Define Custom Provider")

	var fields []string

	// Step 0: Name
	fields = append(fields, c.renderField("Provider Name", c.nameInput, c.step == 0,
		"A friendly name for this provider"))

	// Step 1: ID
	fields = append(fields, c.renderField("Provider ID", c.idInput, c.step == 1,
		"Unique identifier (e.g., my-custom-provider)"))

	// Step 2: Type
	typeHelp := "Use ↑/↓ to select from: openai-compat, openai, anthropic, google, azure, bedrock, vertexai, openrouter"
	if c.step == 2 {
		typeHelp = t.S().Success.Render("↑/↓ to change type | Enter to continue")
	}
	fields = append(fields, c.renderField("Provider Type", c.typeInput, c.step == 2, typeHelp))

	// Step 3: API Endpoint
	fields = append(fields, c.renderField("API Endpoint", c.apiEndpointInput, c.step == 3,
		"Base URL for the API (e.g., https://api.example.com/v1)"))

	// Step 4: Headers
	if c.step == 4 {
		headerSection := c.renderHeaders()
		fields = append(fields, headerSection)
	} else {
		headerCount := len(c.headers)
		headerText := "optional"
		if headerCount > 0 {
			headerText = fmt.Sprintf("%d configured", headerCount)
		}
		fields = append(fields, c.renderReadOnlyField("Default Headers", headerText, c.step == 4))
	}

	// Step 5: Confirm/Review
	if c.step == 5 {
		fields = append(fields, c.renderSummary())
	}

	// Help text at each step.
	help := c.getHelpText()

	content := strings.Join(fields, "\n\n")

	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		content,
		"",
		help,
	)
}

func (c *CustomProviderDefine) renderField(label string, input textinput.Model, focused bool, help string) string {
	t := styles.CurrentTheme()

	labelStyle := t.S().Text
	if focused {
		labelStyle = t.S().Success.Bold(true)
	}

	fieldLabel := labelStyle.Render(label + ":")
	inputView := input.View()
	helpText := t.S().Subtle.Render(help)

	return fieldLabel + "\n" + inputView + "\n" + helpText
}

func (c *CustomProviderDefine) renderReadOnlyField(label, value string, focused bool) string {
	t := styles.CurrentTheme()

	labelStyle := t.S().Text
	if focused {
		labelStyle = t.S().Success.Bold(true)
	}

	fieldLabel := labelStyle.Render(label + ":")
	valueText := t.S().Muted.Render(value)

	return fieldLabel + "\n" + valueText
}

func (c *CustomProviderDefine) renderHeaders() string {
	t := styles.CurrentTheme()

	labelStyle := t.S().Success.Bold(true)

	var headerLines []string
	if len(c.headers) > 0 {
		headerLines = append(headerLines, "")
		headerLines = append(headerLines, t.S().Muted.Render("Configured headers:"))
		for k, v := range c.headers {
			headerLines = append(headerLines, t.S().Subtle.Render(fmt.Sprintf("  %s: %s", k, v)))
		}
		headerLines = append(headerLines, "")
	}

	keyLabel := labelStyle.Render("Header Key:")
	if c.headerIndex == 0 {
		keyLabel = t.S().Success.Bold(true).Render("Header Key:")
	}
	valLabel := labelStyle.Render("Header Value:")
	if c.headerIndex == 1 {
		valLabel = t.S().Success.Bold(true).Render("Header Value:")
	}

	headerLines = append(headerLines, keyLabel)
	headerLines = append(headerLines, c.headerKeyInput.View())
	headerLines = append(headerLines, valLabel)
	headerLines = append(headerLines, c.headerValInput.View())

	return strings.Join(headerLines, "\n")
}

func (c *CustomProviderDefine) renderSummary() string {
	t := styles.CurrentTheme()

	summaryStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Primary).
		Padding(1)

	lines := []string{
		t.S().Title.Render("Review Configuration"),
		"",
		t.S().Text.Render(fmt.Sprintf("Name: %s", c.nameInput.Value())),
		t.S().Text.Render(fmt.Sprintf("ID: %s", c.idInput.Value())),
		t.S().Text.Render(fmt.Sprintf("Type: %s", c.typeInput.Value())),
		t.S().Text.Render(fmt.Sprintf("API Endpoint: %s", c.apiEndpointInput.Value())),
	}

	if len(c.headers) > 0 {
		lines = append(lines, "")
		lines = append(lines, t.S().Text.Render("Headers:"))
		for k, v := range c.headers {
			lines = append(lines, t.S().Subtle.Render(fmt.Sprintf("  %s: %s", k, v)))
		}
	}

	lines = append(lines, "")
	lines = append(lines, t.S().Success.Render("Press Enter to continue to model configuration"))
	lines = append(lines, t.S().Muted.Render("Press Tab to add more headers"))

	return summaryStyle.Render(strings.Join(lines, "\n"))
}

func (c *CustomProviderDefine) getHelpText() string {
	t := styles.CurrentTheme()

	switch c.step {
	case 0, 1, 2, 3:
		return t.S().Muted.Render("Enter to continue • Tab to skip")
	case 4:
		if c.headerMode {
			return t.S().Muted.Render("Enter to add header • Tab to finish headers")
		}
		return t.S().Muted.Render("Tab to finish headers")
	case 5:
		return t.S().Muted.Render("Enter to confirm • Esc to cancel")
	default:
		return ""
	}
}

// SetWidth sets the component width.
func (c *CustomProviderDefine) SetWidth(width int) {
	c.width = width
	maxWidth := width - 4
	c.nameInput.SetWidth(maxWidth)
	c.idInput.SetWidth(maxWidth)
	c.typeInput.SetWidth(maxWidth)
	c.apiEndpointInput.SetWidth(maxWidth)
	c.headerKeyInput.SetWidth(maxWidth / 2)
	c.headerValInput.SetWidth(maxWidth / 2)
}

// Cursor returns the cursor position.
func (c *CustomProviderDefine) Cursor() *tea.Cursor {
	switch c.step {
	case 0:
		return c.nameInput.Cursor()
	case 1:
		return c.idInput.Cursor()
	case 2:
		return c.typeInput.Cursor()
	case 3:
		return c.apiEndpointInput.Cursor()
	case 4:
		if c.headerIndex == 0 {
			return c.headerKeyInput.Cursor()
		}
		return c.headerValInput.Cursor()
	}
	return nil
}
