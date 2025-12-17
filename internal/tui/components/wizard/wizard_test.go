package wizard

import (
	"strings"
	"testing"

	"github.com/charmbracelet/catwalk/pkg/catwalk"
)

func TestNewWizard(t *testing.T) {
	providers := []catwalk.Provider{
		{ID: "anthropic", Name: "Anthropic"},
		{ID: "openai", Name: "OpenAI"},
	}

	w := NewWizard(providers)

	if w == nil {
		t.Fatal("NewWizard() returned nil")
	}

	if w.step != StepProvider {
		t.Errorf("initial step = %d, want %d", w.step, StepProvider)
	}

	if len(w.providers) != 2 {
		t.Errorf("providers count = %d, want 2", len(w.providers))
	}

	if w.providerList == nil {
		t.Error("providerList should be initialized")
	}
}

func TestWizard_Init(t *testing.T) {
	providers := []catwalk.Provider{
		{ID: "anthropic", Name: "Anthropic"},
	}

	w := NewWizard(providers)
	cmd := w.Init()

	// Init should return a command from providerList.Init().
	// The exact command depends on the list implementation.
	_ = cmd
}

func TestWizard_IsComplete(t *testing.T) {
	providers := []catwalk.Provider{
		{ID: "anthropic", Name: "Anthropic"},
	}

	w := NewWizard(providers)

	if w.IsComplete() {
		t.Error("IsComplete() = true, want false initially")
	}

	w.step = StepComplete
	if !w.IsComplete() {
		t.Error("IsComplete() = false, want true when step is StepComplete")
	}
}

func TestWizard_SetSize(t *testing.T) {
	providers := []catwalk.Provider{
		{ID: "anthropic", Name: "Anthropic"},
	}

	w := NewWizard(providers)
	w.SetSize(100, 50)

	if w.width != 100 {
		t.Errorf("width = %d, want 100", w.width)
	}
	if w.height != 50 {
		t.Errorf("height = %d, want 50", w.height)
	}
}

func TestWizard_View_ProviderStep(t *testing.T) {
	providers := []catwalk.Provider{
		{ID: "anthropic", Name: "Anthropic"},
	}

	w := NewWizard(providers)
	w.SetSize(80, 24)
	_ = w.Init()

	view := w.View()

	// Should show progress.
	if !strings.Contains(view, "Provider") {
		t.Error("View() should show 'Provider' in progress")
	}
}

func TestWizard_View_Complete(t *testing.T) {
	providers := []catwalk.Provider{
		{ID: "anthropic", Name: "Anthropic"},
	}

	w := NewWizard(providers)
	w.step = StepComplete
	w.selectedProvider = &providers[0]
	w.selectedLarge = &catwalk.Model{ID: "large", Name: "Large Model"}
	w.selectedSmall = &catwalk.Model{ID: "small", Name: "Small Model"}
	w.apiKey = "test-key"

	view := w.View()

	if !strings.Contains(view, "Complete") {
		t.Error("View() should show 'Complete' when finished")
	}
	if !strings.Contains(view, "Anthropic") {
		t.Error("View() should show provider name")
	}
}

func TestWizard_GoBack(t *testing.T) {
	providers := []catwalk.Provider{
		{ID: catwalk.InferenceProviderAnthropic, Name: "Anthropic"},
	}

	//nolint:govet // Field order optimized for test readability.
	tests := []struct {
		name       string
		fromStep   Step
		expectStep Step
		setup      func(*Wizard)
	}{
		{
			name:       "can't go back from provider",
			fromStep:   StepProvider,
			expectStep: StepProvider,
		},
		{
			name:       "auth method to provider",
			fromStep:   StepAuthMethod,
			expectStep: StepProvider,
		},
		{
			name:       "oauth to auth method",
			fromStep:   StepOAuth,
			expectStep: StepAuthMethod,
		},
		{
			name:       "api key to auth method for anthropic",
			fromStep:   StepAPIKey,
			expectStep: StepAuthMethod,
			setup: func(w *Wizard) {
				w.selectedProvider = &providers[0]
			},
		},
		{
			name:       "large model to api key",
			fromStep:   StepLargeModel,
			expectStep: StepAPIKey,
			setup: func(w *Wizard) {
				w.apiKeyInput = NewAPIKeyInput("Test")
			},
		},
		{
			name:       "small model to large model",
			fromStep:   StepSmallModel,
			expectStep: StepLargeModel,
		},
		{
			name:       "can't go back from complete",
			fromStep:   StepComplete,
			expectStep: StepComplete,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := NewWizard(providers)
			w.step = tt.fromStep
			if tt.setup != nil {
				tt.setup(w)
			}

			w.goBack()

			if w.step != tt.expectStep {
				t.Errorf("step = %d, want %d", w.step, tt.expectStep)
			}
		})
	}
}

func TestWizard_Cursor(t *testing.T) {
	providers := []catwalk.Provider{
		{ID: "anthropic", Name: "Anthropic"},
	}

	w := NewWizard(providers)

	// Provider step - no cursor.
	w.step = StepProvider
	if w.Cursor() != nil {
		t.Error("Cursor() should be nil in provider step")
	}

	// API key step - should have cursor.
	w.step = StepAPIKey
	w.apiKeyInput = NewAPIKeyInput("Test")
	_ = w.apiKeyInput.Init()
	// Cursor depends on input focus state.
	_ = w.Cursor()

	// OAuth step - should have cursor.
	w.step = StepOAuth
	w.oauthFlow = NewOAuth2Flow()
	_ = w.oauthFlow.Init()
	_ = w.Cursor()
}

func TestStep_Constants(t *testing.T) {
	steps := []Step{
		StepProvider,
		StepAuthMethod,
		StepOAuth,
		StepAPIKey,
		StepLargeModel,
		StepSmallModel,
		StepComplete,
	}

	// All should be distinct.
	seen := make(map[Step]bool)
	for _, s := range steps {
		if seen[s] {
			t.Errorf("Duplicate step: %d", s)
		}
		seen[s] = true
	}

	// StepProvider should be 0 (first).
	if StepProvider != 0 {
		t.Errorf("StepProvider = %d, want 0", StepProvider)
	}
}

func TestCompleteMsg_Fields(t *testing.T) {
	msg := CompleteMsg{
		ProviderID:   "anthropic",
		APIKey:       "test-key",
		LargeModelID: "claude-opus",
		SmallModelID: "claude-haiku",
	}

	if msg.ProviderID != "anthropic" {
		t.Errorf("ProviderID = %q, want %q", msg.ProviderID, "anthropic")
	}
	if msg.APIKey != "test-key" {
		t.Errorf("APIKey = %q, want %q", msg.APIKey, "test-key")
	}
	if msg.LargeModelID != "claude-opus" {
		t.Errorf("LargeModelID = %q, want %q", msg.LargeModelID, "claude-opus")
	}
	if msg.SmallModelID != "claude-haiku" {
		t.Errorf("SmallModelID = %q, want %q", msg.SmallModelID, "claude-haiku")
	}
}

func TestWizard_OAuthStepIndex(t *testing.T) {
	providers := []catwalk.Provider{
		{ID: catwalk.InferenceProviderAnthropic, Name: "Anthropic"},
	}

	w := NewWizard(providers)
	w.selectedProvider = &providers[0]
	w.authMethod = AuthMethodOAuth2

	tests := []struct {
		step Step
		want int
	}{
		{StepProvider, 0},
		{StepAuthMethod, 1},
		{StepOAuth, 2},
		{StepAPIKey, 2},
		{StepLargeModel, 3},
		{StepSmallModel, 4},
		{StepComplete, 5},
	}

	for _, tt := range tests {
		w.step = tt.step
		got := w.oauthStepIndex()
		if got != tt.want {
			t.Errorf("oauthStepIndex() for step %d = %d, want %d", tt.step, got, tt.want)
		}
	}
}

func TestWizard_APIKeyStepIndex(t *testing.T) {
	providers := []catwalk.Provider{
		{ID: "openai", Name: "OpenAI"},
	}

	w := NewWizard(providers)

	tests := []struct {
		step Step
		want int
	}{
		{StepProvider, 0},
		{StepAuthMethod, 1},
		{StepAPIKey, 1},
		{StepOAuth, 1},
		{StepLargeModel, 2},
		{StepSmallModel, 3},
		{StepComplete, 4},
	}

	for _, tt := range tests {
		w.step = tt.step
		got := w.apiKeyStepIndex()
		if got != tt.want {
			t.Errorf("apiKeyStepIndex() for step %d = %d, want %d", tt.step, got, tt.want)
		}
	}
}
