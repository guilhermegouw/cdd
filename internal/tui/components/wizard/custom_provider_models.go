// Package wizard provides custom provider wizard components.
//
//nolint:gocritic // appendCombine is stylistic preference; rangeValCopy is acceptable here.
package wizard

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/catwalk/pkg/catwalk"

	"github.com/guilhermegouw/cdd/internal/config"
	"github.com/guilhermegouw/cdd/internal/tui/styles"
	"github.com/guilhermegouw/cdd/internal/tui/util"
)

// CustomProviderModelsCompleteMsg is sent when custom provider models are complete.
type CustomProviderModelsCompleteMsg struct {
	Provider config.CustomProvider
}

// CustomProviderModels handles model definition for custom providers.
type CustomProviderModels struct {
	provider config.CustomProvider

	// Input fields for model.
	modelNameInput    textinput.Model
	modelIDInput      textinput.Model
	contextInput      textinput.Model
	costInInput       textinput.Model
	costOutInput      textinput.Model
	maxTokensInput    textinput.Model

	// State.
	models    []catwalk.Model
	step      int // 0: name, 1: id, 2: context, 3: cost_in, 4: cost_out, 5: max_tokens, 6: confirm
	editIndex int // -1 when adding new model

	width int
}

// NewCustomProviderModels creates a new custom provider models component.
func NewCustomProviderModels(provider config.CustomProvider) *CustomProviderModels {
	t := styles.CurrentTheme()

	modelNameInput := textinput.New()
	modelNameInput.Placeholder = "My Model"
	modelNameInput.Prompt = "> "
	modelNameInput.SetStyles(t.S().TextInput)
	modelNameInput.Focus()

	modelIDInput := textinput.New()
	modelIDInput.Placeholder = "my-model"
	modelIDInput.Prompt = "> "
	modelIDInput.SetStyles(t.S().TextInput)

	contextInput := textinput.New()
	contextInput.Placeholder = "128000"
	contextInput.Prompt = "> "
	contextInput.SetStyles(t.S().TextInput)
	contextInput.SetValue("128000")

	costInInput := textinput.New()
	costInInput.Placeholder = "0.01"
	costInInput.Prompt = "> "
	costInInput.SetStyles(t.S().TextInput)

	costOutInput := textinput.New()
	costOutInput.Placeholder = "0.03"
	costOutInput.Prompt = "> "
	costOutInput.SetStyles(t.S().TextInput)

	maxTokensInput := textinput.New()
	maxTokensInput.Placeholder = "4096"
	maxTokensInput.Prompt = "> "
	maxTokensInput.SetStyles(t.S().TextInput)
	maxTokensInput.SetValue("4096")

	return &CustomProviderModels{
		provider:         provider,
		modelNameInput:   modelNameInput,
		modelIDInput:     modelIDInput,
		contextInput:     contextInput,
		costInInput:      costInInput,
		costOutInput:     costOutInput,
		maxTokensInput:   maxTokensInput,
		models:           []catwalk.Model{},
		step:             0,
		editIndex:        -1,
		width:            70,
	}
}

// Init initializes the component.
func (c *CustomProviderModels) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages.
func (c *CustomProviderModels) Update(msg tea.Msg) (util.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return c, nil
	}

	switch keyMsg.String() {
	case keyEnter:
		return c.handleEnter()
	case "tab":
		return c.handleTab()
	case "esc":
		// Cancel adding this model.
		return c.cancelModel()
	}

	// Update focused input.
	var cmd tea.Cmd
	switch c.step {
	case 0:
		c.modelNameInput, cmd = c.modelNameInput.Update(msg)
	case 1:
		c.modelIDInput, cmd = c.modelIDInput.Update(msg)
	case 2:
		c.contextInput, cmd = c.contextInput.Update(msg)
	case 3:
		c.costInInput, cmd = c.costInInput.Update(msg)
	case 4:
		c.costOutInput, cmd = c.costOutInput.Update(msg)
	case 5:
		c.maxTokensInput, cmd = c.maxTokensInput.Update(msg)
	}

	return c, cmd
}

func (c *CustomProviderModels) handleEnter() (util.Model, tea.Cmd) {
	switch c.step {
	case 0: // Name
		if strings.TrimSpace(c.modelNameInput.Value()) != "" {
			c.step = 1
			c.modelIDInput.Focus()
			c.modelNameInput.Blur()
			// Auto-generate ID from name if empty.
			if c.modelIDInput.Value() == "" {
				generateID := strings.ToLower(strings.TrimSpace(c.modelNameInput.Value()))
				generateID = strings.ReplaceAll(generateID, " ", "-")
				generateID = strings.ReplaceAll(generateID, "_", "-")
				c.modelIDInput.SetValue(generateID)
			}
		}
	case 1: // ID
		if strings.TrimSpace(c.modelIDInput.Value()) != "" {
			c.step = 2
			c.contextInput.Focus()
			c.modelIDInput.Blur()
		}
	case 2: // Context Window
		c.step = 3
		c.costInInput.Focus()
		c.contextInput.Blur()
	case 3: // Cost In
		c.step = 4
		c.costOutInput.Focus()
		c.costInInput.Blur()
	case 4: // Cost Out
		c.step = 5
		c.maxTokensInput.Focus()
		c.costOutInput.Blur()
	case 5: // Max Tokens
		c.step = 6
		c.maxTokensInput.Blur()
	case 6: // Confirm/Add Another
		// Add the model.
		model := c.buildModel()
		c.models = append(c.models, model)
		c.resetInputs()
		c.step = 0
		c.modelNameInput.Focus()
	case 7: // Finish
		// Save all models and complete.
		c.provider.Models = c.models
		return c, util.CmdHandler(CustomProviderModelsCompleteMsg{Provider: c.provider})
	}
	return c, nil
}

func (c *CustomProviderModels) handleTab() (util.Model, tea.Cmd) {
	if c.step == 6 {
		// Skip to finish if we have at least one model.
		if len(c.models) > 0 {
			c.step = 7
		}
	} else if c.step == 7 {
		c.step = 6
	}
	return c, nil
}

func (c *CustomProviderModels) cancelModel() (util.Model, tea.Cmd) {
	if c.step == 6 || c.step == 7 {
		// Cancel - finish with current models.
		if len(c.models) > 0 {
			c.provider.Models = c.models
			return c, util.CmdHandler(CustomProviderModelsCompleteMsg{Provider: c.provider})
		}
	}
	return c, nil
}

func (c *CustomProviderModels) buildModel() catwalk.Model {
	contextWindow := int64(128000)
	if ctx, err := strconv.Atoi(c.contextInput.Value()); err == nil {
		contextWindow = int64(ctx)
	}

	costIn := 0.0
	if cost, err := strconv.ParseFloat(c.costInInput.Value(), 64); err == nil {
		costIn = cost
	}

	costOut := 0.0
	if cost, err := strconv.ParseFloat(c.costOutInput.Value(), 64); err == nil {
		costOut = cost
	}

	maxTokens := int64(4096)
	if mt, err := strconv.Atoi(c.maxTokensInput.Value()); err == nil {
		maxTokens = int64(mt)
	}

	return catwalk.Model{
		ID:                   strings.TrimSpace(c.modelIDInput.Value()),
		Name:                 strings.TrimSpace(c.modelNameInput.Value()),
		ContextWindow:        contextWindow,
		CostPer1MIn:          costIn,
		CostPer1MOut:         costOut,
		CostPer1MInCached:    costIn * 0.1, // Default cached to 10% of input cost
		CostPer1MOutCached:   costOut,
		DefaultMaxTokens:     maxTokens,
	}
}

func (c *CustomProviderModels) resetInputs() {
	c.modelNameInput.Reset()
	c.modelIDInput.Reset()
	c.contextInput.SetValue("128000")
	c.costInInput.Reset()
	c.costOutInput.Reset()
	c.maxTokensInput.SetValue("4096")
}

// View renders the models configuration.
func (c *CustomProviderModels) View() string {
	t := styles.CurrentTheme()

	title := t.S().Title.Render(fmt.Sprintf("Define Models for %s", c.provider.Name))

	var content []string

	// Existing models list.
	if len(c.models) > 0 {
		content = append(content, c.renderModelList())
		content = append(content, "")
	}

	// Current step input.
	content = append(content, c.renderCurrentStep())

	// Help text.
	help := c.getHelpText()
	content = append(content, "")
	content = append(content, help)

	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		strings.Join(content, "\n"),
	)
}

func (c *CustomProviderModels) renderModelList() string {
	t := styles.CurrentTheme()

	lines := []string{t.S().Text.Render("Configured Models:")}

	for i := range c.models {
		modelInfo := fmt.Sprintf("  %d. %s (%s)", i+1, c.models[i].Name, c.models[i].ID)
		lines = append(lines, t.S().Subtle.Render(modelInfo))
	}

	return strings.Join(lines, "\n")
}

func (c *CustomProviderModels) renderCurrentStep() string {
	switch c.step {
	case 0:
		return c.renderField("Model Name", c.modelNameInput, "A friendly name for this model")
	case 1:
		return c.renderField("Model ID", c.modelIDInput, "Unique identifier (e.g., gpt-4)")
	case 2:
		return c.renderField("Context Window", c.contextInput, "Maximum context size in tokens (default: 128000)")
	case 3:
		return c.renderField("Cost per 1M Input", c.costInInput, "Cost per 1M input tokens (optional)")
	case 4:
		return c.renderField("Cost per 1M Output", c.costOutInput, "Cost per 1M output tokens (optional)")
	case 5:
		return c.renderField("Default Max Tokens", c.maxTokensInput, "Default max tokens for responses (default: 4096)")
	case 6:
		return c.renderConfirmStep()
	case 7:
		return c.renderFinishStep()
	}
	return ""
}

func (c *CustomProviderModels) renderField(label string, input textinput.Model, help string) string {
	t := styles.CurrentTheme()

	fieldLabel := t.S().Success.Bold(true).Render(label + ":")
	inputView := input.View()
	helpText := t.S().Subtle.Render(help)

	return fieldLabel + "\n" + inputView + "\n" + helpText
}

func (c *CustomProviderModels) renderConfirmStep() string {
	t := styles.CurrentTheme()

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Primary).
		Padding(1)

	lines := []string{
		t.S().Title.Render("Model Summary"),
		"",
		t.S().Text.Render(fmt.Sprintf("Name: %s", c.modelNameInput.Value())),
		t.S().Text.Render(fmt.Sprintf("ID: %s", c.modelIDInput.Value())),
		t.S().Text.Render(fmt.Sprintf("Context: %s tokens", c.contextInput.Value())),
	}

	if c.costInInput.Value() != "" {
		lines = append(lines, t.S().Text.Render(fmt.Sprintf("Cost In: $%s / 1M tokens", c.costInInput.Value())))
	}
	if c.costOutInput.Value() != "" {
		lines = append(lines, t.S().Text.Render(fmt.Sprintf("Cost Out: $%s / 1M tokens", c.costOutInput.Value())))
	}
	if c.maxTokensInput.Value() != "" {
		lines = append(lines, t.S().Text.Render(fmt.Sprintf("Max Tokens: %s", c.maxTokensInput.Value())))
	}

	lines = append(lines, "")
	lines = append(lines, t.S().Success.Render("Press Enter to add this model"))
	lines = append(lines, t.S().Muted.Render("Press Tab to finish (skip adding more models)"))

	return boxStyle.Render(strings.Join(lines, "\n"))
}

func (c *CustomProviderModels) renderFinishStep() string {
	t := styles.CurrentTheme()

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Success).
		Padding(1)

	lines := []string{
		t.S().Success.Bold(true).Render(fmt.Sprintf("Configured %d model(s)", len(c.models))),
		"",
	}

	for i, m := range c.models {
		lines = append(lines, t.S().Text.Render(fmt.Sprintf("%d. %s", i+1, m.Name)))
	}

	lines = append(lines, "")
	lines = append(lines, t.S().Success.Render("Press Enter to save and continue"))
	lines = append(lines, t.S().Muted.Render("Press Tab to add more models"))

	return boxStyle.Render(strings.Join(lines, "\n"))
}

func (c *CustomProviderModels) getHelpText() string {
	t := styles.CurrentTheme()

	switch c.step {
	case 0, 1, 2, 3, 4, 5:
		return t.S().Muted.Render("Enter to continue • Tab to skip optional fields")
	case 6:
		return t.S().Muted.Render("Enter to add model • Tab to finish")
	case 7:
		return t.S().Muted.Render("Enter to save • Tab to add more models")
	default:
		return ""
	}
}

// SetSize sets the component width.
func (c *CustomProviderModels) SetSize(width, height int) {
	c.width = width
	maxWidth := width - 4
	c.modelNameInput.SetWidth(maxWidth)
	c.modelIDInput.SetWidth(maxWidth)
	c.contextInput.SetWidth(maxWidth)
	c.costInInput.SetWidth(maxWidth)
	c.costOutInput.SetWidth(maxWidth)
	c.maxTokensInput.SetWidth(maxWidth)
}

// Cursor returns the cursor position.
func (c *CustomProviderModels) Cursor() *tea.Cursor {
	switch c.step {
	case 0:
		return c.modelNameInput.Cursor()
	case 1:
		return c.modelIDInput.Cursor()
	case 2:
		return c.contextInput.Cursor()
	case 3:
		return c.costInInput.Cursor()
	case 4:
		return c.costOutInput.Cursor()
	case 5:
		return c.maxTokensInput.Cursor()
	}
	return nil
}
