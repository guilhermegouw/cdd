// Package wizard implements the setup wizard for CDD CLI.
//
//nolint:goconst // Key strings are standard keyboard identifiers.
package wizard

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/catwalk/pkg/catwalk"

	"github.com/guilhermegouw/cdd/internal/config"
	"github.com/guilhermegouw/cdd/internal/oauth"
	"github.com/guilhermegouw/cdd/internal/tui/styles"
	"github.com/guilhermegouw/cdd/internal/tui/util"
)

// Step represents the current step in the wizard.
type Step int

// Wizard steps.
const (
	StepProvider Step = iota
	StepCustomProviderMethod
	StepCustomProviderDefine
	StepCustomProviderModels
	StepAuthMethod
	StepOAuth
	StepAPIKey
	StepLargeModel
	StepSmallModel
	StepComplete
)

// CompleteMsg is sent when the wizard is complete.
type CompleteMsg struct {
	ProviderID   string
	APIKey       string
	LargeModelID string
	SmallModelID string
}

// Wizard manages the setup wizard flow.
type Wizard struct {
	providerList           *ProviderList
	authMethodChoice       *AuthMethodChooser
	oauthFlow              *OAuth2Flow
	apiKeyInput            *APIKeyInput
	largeModel             *ModelList
	smallModel             *ModelList
	customProviderMethod   *CustomProviderMethod
	customProviderDefine   *CustomProviderDefine
	customProviderModels   *CustomProviderModels
	selectedProvider       *catwalk.Provider
	selectedCustomProvider *config.CustomProvider
	selectedLarge          *catwalk.Model
	selectedSmall          *catwalk.Model
	oauthToken             *oauth.Token
	apiKey                 string
	providers              []catwalk.Provider
	height                 int
	width                  int
	step                   Step
	authMethod             AuthMethod
}

// NewWizard creates a new wizard instance.
func NewWizard(providers []catwalk.Provider) *Wizard {
	// Add custom provider option to the list.
	providersWithCustom := AddCustomProviderOption(providers)
	return &Wizard{
		step:         StepProvider,
		providers:    providersWithCustom,
		providerList: NewProviderList(providersWithCustom),
	}
}

// Init initializes the wizard.
func (w *Wizard) Init() tea.Cmd {
	return w.providerList.Init()
}

// Update handles messages.
func (w *Wizard) Update(msg tea.Msg) (util.Model, tea.Cmd) {
	// Handle escape to go back.
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.String() == "esc" {
			w.goBack()
			return w, nil
		}
	}

	switch w.step {
	case StepProvider:
		return w.updateProvider(msg)
	case StepCustomProviderMethod:
		return w.updateCustomProviderMethod(msg)
	case StepCustomProviderDefine:
		return w.updateCustomProviderDefine(msg)
	case StepCustomProviderModels:
		return w.updateCustomProviderModels(msg)
	case StepAuthMethod:
		return w.updateAuthMethod(msg)
	case StepOAuth:
		return w.updateOAuth(msg)
	case StepAPIKey:
		return w.updateAPIKey(msg)
	case StepLargeModel:
		return w.updateLargeModel(msg)
	case StepSmallModel:
		return w.updateSmallModel(msg)
	case StepComplete:
		return w, nil
	}

	return w, nil
}

func (w *Wizard) updateProvider(msg tea.Msg) (util.Model, tea.Cmd) {
	if m, ok := msg.(ProviderSelectedMsg); ok {
		// Check if "Add Custom Provider" was selected.
		if m.Provider.ID == catwalk.InferenceProvider("custom") {
			w.customProviderMethod = NewCustomProviderMethod()
			w.customProviderMethod.SetWidth(w.width)
			w.step = StepCustomProviderMethod
			return w, w.customProviderMethod.Init()
		}

		w.selectedProvider = &m.Provider

		// Check if this is Anthropic - offer OAuth option.
		if m.Provider.ID == catwalk.InferenceProviderAnthropic {
			w.authMethodChoice = NewAuthMethodChooser(m.Provider.Name)
			w.authMethodChoice.SetWidth(w.width)
			w.step = StepAuthMethod
			return w, w.authMethodChoice.Init()
		}

		// For other providers, go directly to API key.
		w.apiKeyInput = NewAPIKeyInput(m.Provider.Name)
		w.apiKeyInput.SetWidth(w.width)
		w.step = StepAPIKey
		return w, w.apiKeyInput.Init()
	}

	_, cmd := w.providerList.Update(msg)
	return w, cmd
}

func (w *Wizard) updateAuthMethod(msg tea.Msg) (util.Model, tea.Cmd) {
	if m, ok := msg.(AuthMethodSelectedMsg); ok {
		w.authMethod = m.Method

		if m.Method == AuthMethodOAuth2 {
			w.oauthFlow = NewOAuth2Flow()
			w.oauthFlow.SetWidth(w.width)
			w.step = StepOAuth
			return w, w.oauthFlow.Init()
		}

		// API Key method.
		w.apiKeyInput = NewAPIKeyInput(w.selectedProvider.Name)
		w.apiKeyInput.SetWidth(w.width)
		w.step = StepAPIKey
		return w, w.apiKeyInput.Init()
	}

	_, cmd := w.authMethodChoice.Update(msg)
	return w, cmd
}

func (w *Wizard) updateCustomProviderMethod(msg tea.Msg) (util.Model, tea.Cmd) {
	if m, ok := msg.(CustomProviderMethodSelectedMsg); ok {
		switch m.Method {
		case ProviderImportMethodManual:
			w.customProviderDefine = NewCustomProviderDefine()
			w.customProviderDefine.SetWidth(w.width)
			w.step = StepCustomProviderDefine
			return w, w.customProviderDefine.Init()
		case ProviderImportMethodURL, ProviderImportMethodFile:
			// TODO: Implement URL and file import in future PR.
			// For now, fall back to manual.
			w.customProviderDefine = NewCustomProviderDefine()
			w.customProviderDefine.SetWidth(w.width)
			w.step = StepCustomProviderDefine
			return w, w.customProviderDefine.Init()
		}
	}

	_, cmd := w.customProviderMethod.Update(msg)
	return w, cmd
}

func (w *Wizard) updateCustomProviderDefine(msg tea.Msg) (util.Model, tea.Cmd) {
	if m, ok := msg.(CustomProviderDefinedMsg); ok {
		w.selectedCustomProvider = &m.Provider
		w.customProviderModels = NewCustomProviderModels(m.Provider)
		w.customProviderModels.SetSize(w.width, w.height)
		w.step = StepCustomProviderModels
		return w, w.customProviderModels.Init()
	}

	_, cmd := w.customProviderDefine.Update(msg)
	return w, cmd
}

func (w *Wizard) updateCustomProviderModels(msg tea.Msg) (util.Model, tea.Cmd) {
	if m, ok := msg.(CustomProviderModelsCompleteMsg); ok {
		w.selectedCustomProvider = &m.Provider

		// Save the custom provider.
		loader := config.NewProviderLoader("")
		manager := loader.GetCustomProviderManager()
		// Ignore error - provider addition failure is non-fatal for wizard flow.
		_ = manager.Add(m.Provider) //nolint:errcheck // Best-effort save, continue with wizard regardless

		// Convert to catwalk provider and continue to API key input.
		provider := m.Provider.ToCatwalkProvider()
		w.selectedProvider = &provider

		// Go to API key input.
		w.apiKeyInput = NewAPIKeyInput(provider.Name)
		w.apiKeyInput.SetWidth(w.width)
		w.step = StepAPIKey
		return w, w.apiKeyInput.Init()
	}

	_, cmd := w.customProviderModels.Update(msg)
	return w, cmd
}

func (w *Wizard) updateOAuth(msg tea.Msg) (util.Model, tea.Cmd) {
	// Handle Enter key for OAuth flow.
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == keyEnter {
		return w.oauthFlow.HandleConfirm()
	}

	if m, ok := msg.(OAuthCompleteMsg); ok {
		w.oauthToken = m.Token
		w.apiKey = m.Token.AccessToken

		// Create model lists with provider's models.
		models := w.selectedProvider.Models
		w.largeModel = NewModelList(models, "large", w.selectedProvider.Name)
		w.smallModel = NewModelList(models, "small", w.selectedProvider.Name)
		w.largeModel.SetSize(w.width, w.height)
		w.smallModel.SetSize(w.width, w.height)

		// Pre-select default models if available.
		if w.selectedProvider.DefaultLargeModelID != "" {
			w.largeModel.SetCursorToModel(w.selectedProvider.DefaultLargeModelID)
		}
		if w.selectedProvider.DefaultSmallModelID != "" {
			w.smallModel.SetCursorToModel(w.selectedProvider.DefaultSmallModelID)
		}

		w.step = StepLargeModel
		return w, w.largeModel.Init()
	}

	_, cmd := w.oauthFlow.Update(msg)
	return w, cmd
}

func (w *Wizard) updateAPIKey(msg tea.Msg) (util.Model, tea.Cmd) {
	if m, ok := msg.(APIKeyEnteredMsg); ok {
		w.apiKey = m.APIKey

		// Create model lists with provider's models.
		models := w.selectedProvider.Models
		w.largeModel = NewModelList(models, "large", w.selectedProvider.Name)
		w.smallModel = NewModelList(models, "small", w.selectedProvider.Name)
		w.largeModel.SetSize(w.width, w.height)
		w.smallModel.SetSize(w.width, w.height)

		// Pre-select default models if available.
		if w.selectedProvider.DefaultLargeModelID != "" {
			w.largeModel.SetCursorToModel(w.selectedProvider.DefaultLargeModelID)
		}
		if w.selectedProvider.DefaultSmallModelID != "" {
			w.smallModel.SetCursorToModel(w.selectedProvider.DefaultSmallModelID)
		}

		w.step = StepLargeModel
		return w, w.largeModel.Init()
	}

	_, cmd := w.apiKeyInput.Update(msg)
	return w, cmd
}

func (w *Wizard) updateLargeModel(msg tea.Msg) (util.Model, tea.Cmd) {
	if m, ok := msg.(ModelSelectedMsg); ok {
		w.selectedLarge = &m.Model
		w.step = StepSmallModel
		return w, w.smallModel.Init()
	}

	_, cmd := w.largeModel.Update(msg)
	return w, cmd
}

func (w *Wizard) updateSmallModel(msg tea.Msg) (util.Model, tea.Cmd) {
	if m, ok := msg.(ModelSelectedMsg); ok {
		w.selectedSmall = &m.Model
		w.step = StepComplete
		cmd := w.saveConfig()
		return w, cmd
	}

	_, cmd := w.smallModel.Update(msg)
	return w, cmd
}

func (w *Wizard) goBack() {
	switch w.step {
	case StepCustomProviderMethod:
		w.step = StepProvider
		w.customProviderMethod = nil
	case StepCustomProviderDefine:
		w.step = StepCustomProviderMethod
		w.customProviderDefine = nil
	case StepCustomProviderModels:
		w.step = StepCustomProviderDefine
		w.customProviderModels = nil
	case StepAuthMethod:
		w.step = StepProvider
		w.authMethodChoice = nil
	case StepOAuth:
		w.step = StepAuthMethod
		w.oauthFlow = nil
	case StepAPIKey:
		// If we came from custom provider, go back to provider selection.
		//nolint:gocritic // ifElseChain is clearer than switch for this mixed condition logic
		if w.selectedCustomProvider != nil {
			w.step = StepProvider
			w.apiKeyInput = nil
			w.selectedCustomProvider = nil
		} else if w.selectedProvider.ID == catwalk.InferenceProviderAnthropic {
			w.step = StepAuthMethod
			w.apiKeyInput = nil
		} else {
			w.step = StepProvider
			w.apiKeyInput = nil
		}
	case StepLargeModel:
		// Go back to API key or OAuth depending on auth method.
		if w.oauthToken != nil {
			w.step = StepOAuth
		} else {
			w.step = StepAPIKey
			if w.apiKeyInput != nil {
				w.apiKeyInput.Reset()
			}
		}
	case StepSmallModel:
		w.step = StepLargeModel
	case StepProvider, StepComplete:
		// Can't go back from first step or complete.
	}
}

func (w *Wizard) saveConfig() tea.Cmd {
	return func() tea.Msg {
		var err error

		if w.oauthToken != nil {
			// Save with OAuth token.
			err = config.SaveWizardResultWithOAuth(
				string(w.selectedProvider.ID),
				w.oauthToken,
				w.selectedLarge.ID,
				w.selectedSmall.ID,
			)
		} else {
			// Save with API key.
			err = config.SaveWizardResult(
				string(w.selectedProvider.ID),
				w.apiKey,
				w.selectedLarge.ID,
				w.selectedSmall.ID,
			)
		}

		if err != nil {
			return util.InfoMsg{
				Type: util.InfoTypeError,
				Msg:  fmt.Sprintf("Failed to save config: %v", err),
			}
		}
		return CompleteMsg{
			ProviderID:   string(w.selectedProvider.ID),
			APIKey:       w.apiKey,
			LargeModelID: w.selectedLarge.ID,
			SmallModelID: w.selectedSmall.ID,
		}
	}
}

// View renders the current wizard step.
func (w *Wizard) View() string {
	t := styles.CurrentTheme()

	// Progress indicator.
	progress := w.renderProgress()

	var content string
	switch w.step {
	case StepProvider:
		content = w.providerList.View()
	case StepCustomProviderMethod:
		content = w.customProviderMethod.View()
	case StepCustomProviderDefine:
		content = w.customProviderDefine.View()
	case StepCustomProviderModels:
		content = w.customProviderModels.View()
	case StepAuthMethod:
		content = w.authMethodChoice.View()
	case StepOAuth:
		content = w.oauthFlow.View()
	case StepAPIKey:
		content = w.apiKeyInput.View()
	case StepLargeModel:
		content = w.largeModel.View()
	case StepSmallModel:
		content = w.smallModel.View()
	case StepComplete:
		content = w.renderComplete()
	}

	// Back hint (except on first step).
	backHint := ""
	if w.step > StepProvider && w.step < StepComplete {
		backHint = t.S().Subtle.Render("Press Esc to go back")
	}

	// Inner content centered.
	innerContent := lipgloss.JoinVertical(lipgloss.Center,
		progress,
		"",
		content,
		"",
		backHint,
	)

	// Box style with border.
	boxWidth := min(w.width-4, 70)
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Border).
		Padding(1, 3).
		Width(boxWidth).
		Align(lipgloss.Center)

	box := boxStyle.Render(innerContent)

	// Center the box on screen.
	return lipgloss.Place(
		w.width, w.height,
		lipgloss.Center, lipgloss.Center,
		box,
	)
}

func (w *Wizard) renderProgress() string {
	t := styles.CurrentTheme()

	// Determine which steps to show based on auth method.
	var steps []string
	var currentStepIndex int

	if w.selectedProvider != nil && w.selectedProvider.ID == catwalk.InferenceProviderAnthropic && w.authMethod == AuthMethodOAuth2 {
		steps = []string{"Provider", "Auth", "OAuth", "Large Model", "Small Model"}
		currentStepIndex = w.oauthStepIndex()
	} else {
		steps = []string{"Provider", "API Key", "Large Model", "Small Model"}
		currentStepIndex = w.apiKeyStepIndex()
	}

	parts := make([]string, 0, len(steps)*2-1)
	for i, step := range steps {
		style := t.S().Subtle
		if i == currentStepIndex {
			style = t.S().Success.Bold(true)
		} else if i < currentStepIndex {
			style = t.S().Muted
		}
		parts = append(parts, style.Render(step))
		if i < len(steps)-1 {
			parts = append(parts, t.S().Subtle.Render(" → "))
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Center, parts...)
}

func (w *Wizard) renderComplete() string {
	t := styles.CurrentTheme()

	title := t.S().Success.Bold(true).Render("Setup Complete!")

	authType := "API Key"
	if w.oauthToken != nil {
		authType = "OAuth (Claude Account)"
	}

	summary := lipgloss.JoinVertical(lipgloss.Left,
		t.S().Text.Render(fmt.Sprintf("Provider: %s", w.selectedProvider.Name)),
		t.S().Text.Render(fmt.Sprintf("Authentication: %s", authType)),
		t.S().Text.Render(fmt.Sprintf("Large Model: %s", w.selectedLarge.Name)),
		t.S().Text.Render(fmt.Sprintf("Small Model: %s", w.selectedSmall.Name)),
	)

	configPath := config.GlobalConfigPath()
	saved := t.S().Muted.Render(fmt.Sprintf("Configuration saved to: %s", configPath))

	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		summary,
		"",
		saved,
		"",
		t.S().Info.Render("Press any key to continue..."),
	)
}

// SetSize sets the wizard size.
func (w *Wizard) SetSize(width, height int) {
	w.width = width
	w.height = height

	if w.providerList != nil {
		w.providerList.SetSize(width, height)
	}
	if w.customProviderMethod != nil {
		w.customProviderMethod.SetWidth(width)
	}
	if w.customProviderDefine != nil {
		w.customProviderDefine.SetWidth(width)
	}
	if w.customProviderModels != nil {
		w.customProviderModels.SetSize(width, height)
	}
	if w.authMethodChoice != nil {
		w.authMethodChoice.SetWidth(width)
	}
	if w.oauthFlow != nil {
		w.oauthFlow.SetWidth(width)
	}
	if w.apiKeyInput != nil {
		w.apiKeyInput.SetWidth(width)
	}
	if w.largeModel != nil {
		w.largeModel.SetSize(width, height)
	}
	if w.smallModel != nil {
		w.smallModel.SetSize(width, height)
	}
}

// IsComplete returns true if the wizard is complete.
func (w *Wizard) IsComplete() bool {
	return w.step == StepComplete
}

// SelectedLargeModel returns the selected large model, or nil if not yet selected.
func (w *Wizard) SelectedLargeModel() *catwalk.Model {
	return w.selectedLarge
}

// Cursor returns the cursor position for the current step.
func (w *Wizard) Cursor() *tea.Cursor {
	if w.step == StepAPIKey && w.apiKeyInput != nil {
		return w.apiKeyInput.Cursor()
	}
	if w.step == StepOAuth && w.oauthFlow != nil {
		return w.oauthFlow.Cursor()
	}
	if w.step == StepCustomProviderDefine && w.customProviderDefine != nil {
		return w.customProviderDefine.Cursor()
	}
	if w.step == StepCustomProviderModels && w.customProviderModels != nil {
		return w.customProviderModels.Cursor()
	}
	return nil
}

func (w *Wizard) oauthStepIndex() int {
	switch w.step {
	case StepProvider:
		return 0
	case StepCustomProviderMethod, StepCustomProviderDefine, StepCustomProviderModels:
		return 0 // Custom provider steps are not shown in OAuth progress.
	case StepAuthMethod:
		return 1
	case StepOAuth, StepAPIKey:
		return 2
	case StepLargeModel:
		return 3
	case StepSmallModel:
		return 4
	case StepComplete:
		return 5
	}
	return 0
}

func (w *Wizard) apiKeyStepIndex() int {
	switch w.step {
	case StepProvider:
		return 0
	case StepCustomProviderMethod, StepCustomProviderDefine, StepCustomProviderModels:
		return 0 // Custom provider steps are not shown in API key progress.
	case StepAuthMethod, StepAPIKey, StepOAuth:
		return 1
	case StepLargeModel:
		return 2
	case StepSmallModel:
		return 3
	case StepComplete:
		return 4
	}
	return 0
}
