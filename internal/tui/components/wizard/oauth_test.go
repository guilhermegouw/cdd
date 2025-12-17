package wizard

import (
	"strings"
	"testing"

	"github.com/guilhermegouw/cdd/internal/oauth"
)

func TestNewOAuth2Flow(t *testing.T) {
	flow := NewOAuth2Flow()

	if flow == nil {
		t.Fatal("NewOAuth2Flow() returned nil")
	}

	if flow.state != OAuthStateURL {
		t.Errorf("initial state = %d, want %d (OAuthStateURL)", flow.state, OAuthStateURL)
	}
}

func TestOAuth2Flow_Init(t *testing.T) {
	flow := NewOAuth2Flow()
	cmd := flow.Init()

	// Init should return nil (no async command).
	if cmd != nil {
		t.Error("Init() should return nil")
	}

	// After init, verifier and challenge should be set.
	if flow.verifier == "" {
		t.Error("verifier should be set after Init()")
	}
	if flow.challenge == "" {
		t.Error("challenge should be set after Init()")
	}
	if flow.authURL == "" {
		t.Error("authURL should be set after Init()")
	}
}

func TestOAuth2Flow_IsURLState(t *testing.T) {
	flow := NewOAuth2Flow()
	_ = flow.Init()

	if !flow.IsURLState() {
		t.Error("IsURLState() = false, want true initially")
	}

	flow.state = OAuthStateCode
	if flow.IsURLState() {
		t.Error("IsURLState() = true, want false after state change")
	}
}

func TestOAuth2Flow_IsComplete(t *testing.T) {
	flow := NewOAuth2Flow()
	_ = flow.Init()

	if flow.IsComplete() {
		t.Error("IsComplete() = true, want false initially")
	}

	flow.validationState = OAuthValidationStateValid
	if !flow.IsComplete() {
		t.Error("IsComplete() = false, want true when validation is valid")
	}
}

func TestOAuth2Flow_Token(t *testing.T) {
	flow := NewOAuth2Flow()
	_ = flow.Init()

	if flow.Token() != nil {
		t.Error("Token() should be nil initially")
	}

	flow.token = &oauth.Token{AccessToken: "test-token"}
	if flow.Token() == nil {
		t.Error("Token() should not be nil after setting")
	}
	if flow.Token().AccessToken != "test-token" {
		t.Errorf("Token().AccessToken = %q, want %q", flow.Token().AccessToken, "test-token")
	}
}

func TestOAuth2Flow_SetWidth(t *testing.T) {
	flow := NewOAuth2Flow()
	_ = flow.Init()

	flow.SetWidth(100)
	if flow.width != 100 {
		t.Errorf("width = %d, want %d", flow.width, 100)
	}
}

func TestOAuth2Flow_View_URLState(t *testing.T) {
	flow := NewOAuth2Flow()
	_ = flow.Init()

	view := flow.View()

	// Should contain instructions.
	if !strings.Contains(view, "Enter") {
		t.Error("View() in URL state should mention Enter key")
	}
	if !strings.Contains(view, "browser") {
		t.Error("View() in URL state should mention browser")
	}

	// Should contain URL (shortened).
	if !strings.Contains(view, "claude.ai") {
		t.Error("View() in URL state should show authorization URL")
	}
}

func TestOAuth2Flow_View_CodeState(t *testing.T) {
	flow := NewOAuth2Flow()
	_ = flow.Init()
	flow.state = OAuthStateCode
	flow.validationState = OAuthValidationStateNone

	view := flow.View()

	// Should ask for code.
	if !strings.Contains(view, "code") {
		t.Error("View() in code state should mention code")
	}
}

func TestOAuth2Flow_View_VerifyingState(t *testing.T) {
	flow := NewOAuth2Flow()
	_ = flow.Init()
	flow.state = OAuthStateCode
	flow.validationState = OAuthValidationStateVerifying

	view := flow.View()

	if !strings.Contains(view, "Verifying") {
		t.Error("View() should show 'Verifying' when in verifying state")
	}
}

func TestOAuth2Flow_View_ValidState(t *testing.T) {
	flow := NewOAuth2Flow()
	_ = flow.Init()
	flow.state = OAuthStateCode
	flow.validationState = OAuthValidationStateValid

	view := flow.View()

	if !strings.Contains(view, "Validated") {
		t.Error("View() should show 'Validated' when validation is complete")
	}
}

func TestOAuth2Flow_View_ErrorState(t *testing.T) {
	flow := NewOAuth2Flow()
	_ = flow.Init()
	flow.state = OAuthStateCode
	flow.validationState = OAuthValidationStateError

	view := flow.View()

	if !strings.Contains(view, "Invalid") {
		t.Error("View() should show 'Invalid' when validation failed")
	}
}

func TestOAuth2Flow_View_WithError(t *testing.T) {
	flow := NewOAuth2Flow()
	flow.err = &testError{msg: "test error message"}

	view := flow.View()

	if !strings.Contains(view, "test error message") {
		t.Error("View() should show error message when err is set")
	}
}

func TestOAuth2Flow_HandleConfirm_URLState(t *testing.T) {
	flow := NewOAuth2Flow()
	_ = flow.Init()

	_, cmd := flow.HandleConfirm()

	// State should change to code.
	if flow.state != OAuthStateCode {
		t.Errorf("state = %d, want %d after confirm in URL state", flow.state, OAuthStateCode)
	}

	// Should return a focus command.
	if cmd == nil {
		t.Error("HandleConfirm() should return a command")
	}
}

func TestOAuth2Flow_HandleConfirm_ValidState(t *testing.T) {
	flow := NewOAuth2Flow()
	_ = flow.Init()
	flow.state = OAuthStateCode
	flow.validationState = OAuthValidationStateValid
	flow.token = &oauth.Token{AccessToken: "test-token"}

	_, cmd := flow.HandleConfirm()

	if cmd == nil {
		t.Fatal("HandleConfirm() should return a command when valid")
	}

	// Execute and check result.
	msg := cmd()
	completeMsg, ok := msg.(OAuthCompleteMsg)
	if !ok {
		t.Fatalf("Expected OAuthCompleteMsg, got %T", msg)
	}

	if completeMsg.Token.AccessToken != "test-token" {
		t.Errorf("Token.AccessToken = %q, want %q", completeMsg.Token.AccessToken, "test-token")
	}
}

func TestOAuth2Flow_Update_ValidationComplete(t *testing.T) {
	flow := NewOAuth2Flow()
	_ = flow.Init()
	flow.state = OAuthStateCode

	token := &oauth.Token{AccessToken: "validated-token"}
	msg := OAuthValidationCompletedMsg{
		Token: token,
		State: OAuthValidationStateValid,
	}

	_, _ = flow.Update(msg)

	if flow.validationState != OAuthValidationStateValid {
		t.Errorf("validationState = %d, want %d", flow.validationState, OAuthValidationStateValid)
	}
	if flow.token.AccessToken != "validated-token" {
		t.Errorf("token.AccessToken = %q, want %q", flow.token.AccessToken, "validated-token")
	}
}

func TestOAuth2Flow_Update_ValidationError(t *testing.T) {
	flow := NewOAuth2Flow()
	_ = flow.Init()
	flow.state = OAuthStateCode
	flow.codeInput.Blur()

	msg := OAuthValidationCompletedMsg{
		State: OAuthValidationStateError,
	}

	_, _ = flow.Update(msg)

	if flow.validationState != OAuthValidationStateError {
		t.Errorf("validationState = %d, want %d", flow.validationState, OAuthValidationStateError)
	}

	// Code input should be focused again for retry.
	if !flow.codeInput.Focused() {
		t.Error("codeInput should be focused after validation error")
	}
}

func TestOAuth2Flow_Cursor(t *testing.T) {
	flow := NewOAuth2Flow()
	_ = flow.Init()

	// In URL state, cursor should be nil.
	if flow.Cursor() != nil {
		t.Error("Cursor() should be nil in URL state")
	}

	flow.state = OAuthStateCode
	// In code state, cursor should come from input.
	// The exact cursor depends on input state, but should not panic.
	_ = flow.Cursor()
}

func TestOAuth2Flow_DisplayURL(t *testing.T) {
	flow := NewOAuth2Flow()
	_ = flow.Init()

	displayURL := flow.displayURL()

	// Should end with ... (query params stripped).
	if !strings.HasSuffix(displayURL, "...") {
		t.Errorf("displayURL() = %q, should end with ...", displayURL)
	}

	// Should contain the base URL.
	if !strings.Contains(displayURL, "claude.ai") {
		t.Error("displayURL() should contain claude.ai")
	}
}

func TestOAuth2Flow_DisplayURL_NoQueryParams(t *testing.T) {
	flow := NewOAuth2Flow()
	flow.authURL = "https://example.com/path"

	displayURL := flow.displayURL()

	// Without query params, should not end with ...
	if strings.HasSuffix(displayURL, "...") {
		t.Error("displayURL() should not end with ... when no query params")
	}
}

func TestOAuth2Flow_DisplayURL_InvalidURL(t *testing.T) {
	flow := NewOAuth2Flow()
	flow.authURL = "://invalid"

	displayURL := flow.displayURL()

	// Should return the original URL on parse error.
	if displayURL != "://invalid" {
		t.Errorf("displayURL() = %q, want %q for invalid URL", displayURL, "://invalid")
	}
}

func TestOAuthState_Constants(t *testing.T) {
	if OAuthStateURL == OAuthStateCode {
		t.Error("OAuthStateURL and OAuthStateCode should be different")
	}
}

func TestOAuthValidationState_Constants(t *testing.T) {
	states := []OAuthValidationState{
		OAuthValidationStateNone,
		OAuthValidationStateVerifying,
		OAuthValidationStateValid,
		OAuthValidationStateError,
	}

	// All should be distinct.
	seen := make(map[OAuthValidationState]bool)
	for _, s := range states {
		if seen[s] {
			t.Errorf("Duplicate validation state: %d", s)
		}
		seen[s] = true
	}
}

func TestOAuthValidationCompletedMsg_Fields(t *testing.T) {
	token := &oauth.Token{AccessToken: "test"}
	msg := OAuthValidationCompletedMsg{
		Token: token,
		State: OAuthValidationStateValid,
	}

	if msg.Token.AccessToken != "test" {
		t.Errorf("Token.AccessToken = %q, want %q", msg.Token.AccessToken, "test")
	}
	if msg.State != OAuthValidationStateValid {
		t.Errorf("State = %d, want %d", msg.State, OAuthValidationStateValid)
	}
}

func TestOAuthCompleteMsg_Fields(t *testing.T) {
	token := &oauth.Token{AccessToken: "complete-token"}
	msg := OAuthCompleteMsg{Token: token}

	if msg.Token.AccessToken != "complete-token" {
		t.Errorf("Token.AccessToken = %q, want %q", msg.Token.AccessToken, "complete-token")
	}
}

// testError is a simple error type for testing.
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
