package wizard

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestNewAuthMethodChooser(t *testing.T) {
	chooser := NewAuthMethodChooser("Anthropic")

	if chooser == nil {
		t.Fatal("NewAuthMethodChooser() returned nil")
	}

	if chooser.providerName != "Anthropic" {
		t.Errorf("providerName = %q, want %q", chooser.providerName, "Anthropic")
	}

	// Default should be OAuth.
	if chooser.selected != AuthMethodOAuth2 {
		t.Errorf("selected = %d, want %d (OAuth2)", chooser.selected, AuthMethodOAuth2)
	}
}

func TestAuthMethodChooser_Init(t *testing.T) {
	chooser := NewAuthMethodChooser("Test")

	cmd := chooser.Init()
	if cmd != nil {
		t.Error("Init() should return nil")
	}
}

func TestAuthMethodChooser_Update_Navigation(t *testing.T) {
	//nolint:govet // Field order optimized for test readability.
	tests := []struct {
		name         string
		initialState AuthMethod
		key          string
		wantState    AuthMethod
	}{
		{
			name:         "left key selects OAuth when on API Key",
			initialState: AuthMethodAPIKey,
			key:          "left",
			wantState:    AuthMethodOAuth2,
		},
		{
			name:         "h key selects OAuth when on API Key",
			initialState: AuthMethodAPIKey,
			key:          "h",
			wantState:    AuthMethodOAuth2,
		},
		{
			name:         "right key selects API Key when on OAuth",
			initialState: AuthMethodOAuth2,
			key:          "right",
			wantState:    AuthMethodAPIKey,
		},
		{
			name:         "l key selects API Key when on OAuth",
			initialState: AuthMethodOAuth2,
			key:          "l",
			wantState:    AuthMethodAPIKey,
		},
		{
			name:         "tab toggles from OAuth to API Key",
			initialState: AuthMethodOAuth2,
			key:          "tab",
			wantState:    AuthMethodAPIKey,
		},
		{
			name:         "tab toggles from API Key to OAuth",
			initialState: AuthMethodAPIKey,
			key:          "tab",
			wantState:    AuthMethodOAuth2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chooser := NewAuthMethodChooser("Test")
			chooser.selected = tt.initialState

			// Use testKeyMsg to simulate key presses.
			msg := tea.KeyPressMsg(tea.Key{Code: -1, Text: tt.key})

			_, _ = chooser.Update(msg)

			if chooser.selected != tt.wantState {
				t.Errorf("selected = %d, want %d", chooser.selected, tt.wantState)
			}
		})
	}
}

func TestAuthMethodChooser_Update_Enter(t *testing.T) {
	chooser := NewAuthMethodChooser("Test")
	chooser.selected = AuthMethodAPIKey

	msg := tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter})
	_, cmd := chooser.Update(msg)

	if cmd == nil {
		t.Fatal("Update(enter) should return a command")
	}

	// Execute the command and check the message.
	resultMsg := cmd()
	selectedMsg, ok := resultMsg.(AuthMethodSelectedMsg)
	if !ok {
		t.Fatalf("Expected AuthMethodSelectedMsg, got %T", resultMsg)
	}

	if selectedMsg.Method != AuthMethodAPIKey {
		t.Errorf("AuthMethodSelectedMsg.Method = %d, want %d", selectedMsg.Method, AuthMethodAPIKey)
	}
}

func TestAuthMethodChooser_Update_NonKeyMsg(t *testing.T) {
	chooser := NewAuthMethodChooser("Test")
	initialState := chooser.selected

	// Send a non-key message.
	_, cmd := chooser.Update("not a key message")

	if cmd != nil {
		t.Error("Update(non-key) should return nil cmd")
	}
	if chooser.selected != initialState {
		t.Error("State should not change for non-key messages")
	}
}

func TestAuthMethodChooser_View(t *testing.T) {
	chooser := NewAuthMethodChooser("Anthropic")
	chooser.SetWidth(80)

	view := chooser.View()

	// Should contain provider name.
	if !strings.Contains(view, "Anthropic") {
		t.Error("View() should contain provider name")
	}

	// Should contain both options.
	if !strings.Contains(view, "Claude Account") {
		t.Error("View() should contain 'Claude Account' option")
	}
	if !strings.Contains(view, "API Key") {
		t.Error("View() should contain 'API Key' option")
	}

	// Should contain help text.
	if !strings.Contains(view, "Tab") {
		t.Error("View() should contain help text mentioning Tab")
	}
}

func TestAuthMethodChooser_SetWidth(t *testing.T) {
	chooser := NewAuthMethodChooser("Test")

	chooser.SetWidth(100)
	if chooser.width != 100 {
		t.Errorf("width = %d, want %d", chooser.width, 100)
	}

	chooser.SetWidth(50)
	if chooser.width != 50 {
		t.Errorf("width = %d, want %d", chooser.width, 50)
	}
}

func TestAuthMethodChooser_ToggleChoice(t *testing.T) {
	chooser := NewAuthMethodChooser("Test")

	// Start with OAuth.
	if chooser.selected != AuthMethodOAuth2 {
		t.Errorf("Initial selected = %d, want %d", chooser.selected, AuthMethodOAuth2)
	}

	chooser.toggleChoice()
	if chooser.selected != AuthMethodAPIKey {
		t.Errorf("After first toggle: selected = %d, want %d", chooser.selected, AuthMethodAPIKey)
	}

	chooser.toggleChoice()
	if chooser.selected != AuthMethodOAuth2 {
		t.Errorf("After second toggle: selected = %d, want %d", chooser.selected, AuthMethodOAuth2)
	}
}

func TestAuthMethod_Constants(t *testing.T) {
	// Verify constants are distinct.
	if AuthMethodOAuth2 == AuthMethodAPIKey {
		t.Error("AuthMethodOAuth2 and AuthMethodAPIKey should be different")
	}

	// OAuth2 should be 0 (default).
	if AuthMethodOAuth2 != 0 {
		t.Errorf("AuthMethodOAuth2 = %d, want 0", AuthMethodOAuth2)
	}
}

func TestAuthMethodSelectedMsg_Fields(t *testing.T) {
	msg := AuthMethodSelectedMsg{
		Method: AuthMethodAPIKey,
	}

	if msg.Method != AuthMethodAPIKey {
		t.Errorf("Method = %d, want %d", msg.Method, AuthMethodAPIKey)
	}
}
