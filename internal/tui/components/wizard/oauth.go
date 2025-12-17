package wizard

import (
	"context"
	"net/url"
	"os"
	"os/exec"
	"runtime"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/guilhermegouw/cdd/internal/oauth"
	"github.com/guilhermegouw/cdd/internal/oauth/claude"
	"github.com/guilhermegouw/cdd/internal/tui/styles"
	"github.com/guilhermegouw/cdd/internal/tui/util"
)

// OAuthState represents the current state of the OAuth flow.
type OAuthState int

// OAuth states.
const (
	OAuthStateURL OAuthState = iota
	OAuthStateCode
)

// OAuthValidationState represents the validation state.
type OAuthValidationState int

// Validation states.
const (
	OAuthValidationStateNone OAuthValidationState = iota
	OAuthValidationStateVerifying
	OAuthValidationStateValid
	OAuthValidationStateError
)

// OAuthValidationCompletedMsg is sent when OAuth validation completes.
type OAuthValidationCompletedMsg struct {
	Token *oauth.Token
	State OAuthValidationState
}

// OAuthCompleteMsg is sent when OAuth authentication is complete.
type OAuthCompleteMsg struct {
	Token *oauth.Token
}

// OAuth2Flow handles the OAuth2 authentication flow.
//
//nolint:govet // Field order optimized for readability over memory.
type OAuth2Flow struct {
	// Code input.
	codeInput textinput.Model
	spinner   spinner.Model

	// PKCE values.
	err       error
	token     *oauth.Token
	verifier  string
	challenge string
	authURL   string
	width     int

	state           OAuthState
	validationState OAuthValidationState
}

// NewOAuth2Flow creates a new OAuth2 flow component.
func NewOAuth2Flow() *OAuth2Flow {
	return &OAuth2Flow{
		state: OAuthStateURL,
	}
}

// Init initializes the OAuth2 flow.
func (o *OAuth2Flow) Init() tea.Cmd {
	t := styles.CurrentTheme()

	// Generate PKCE challenge.
	verifier, challenge, err := claude.GetChallenge()
	if err != nil {
		o.err = err
		return nil
	}

	// Generate authorization URL.
	authURL, err := claude.AuthorizeURL(verifier, challenge)
	if err != nil {
		o.err = err
		return nil
	}

	o.verifier = verifier
	o.challenge = challenge
	o.authURL = authURL

	// Setup code input.
	o.codeInput = textinput.New()
	o.codeInput.Placeholder = "Paste or type the code here..."
	o.codeInput.Prompt = "> "
	o.codeInput.SetStyles(t.S().TextInput)
	o.codeInput.SetWidth(50)

	// Setup spinner.
	o.spinner = spinner.New(
		spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(t.S().Base.Foreground(t.Primary)),
	)

	return nil
}

// Update handles messages.
func (o *OAuth2Flow) Update(msg tea.Msg) (util.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if m, ok := msg.(OAuthValidationCompletedMsg); ok {
		o.validationState = m.State
		o.token = m.Token
		if o.validationState == OAuthValidationStateError {
			o.codeInput.Focus()
		}
		o.updatePrompt()
	}

	if o.validationState == OAuthValidationStateVerifying {
		var cmd tea.Cmd
		o.spinner, cmd = o.spinner.Update(msg)
		cmds = append(cmds, cmd)
		o.updatePrompt()
	}

	var cmd tea.Cmd
	o.codeInput, cmd = o.codeInput.Update(msg)
	cmds = append(cmds, cmd)

	return o, tea.Batch(cmds...)
}

// HandleConfirm handles the Enter key press.
func (o *OAuth2Flow) HandleConfirm() (util.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch {
	case o.state == OAuthStateURL:
		// Open URL in browser and move to code input.
		// Use silent open to avoid disrupting the TUI.
		openBrowserSilent(o.authURL)
		o.state = OAuthStateCode
		cmds = append(cmds, o.codeInput.Focus())

	case o.validationState == OAuthValidationStateNone || o.validationState == OAuthValidationStateError:
		// Validate the code.
		o.codeInput.Blur()
		o.validationState = OAuthValidationStateVerifying
		cmds = append(cmds, o.spinner.Tick, o.validateCode)

	case o.validationState == OAuthValidationStateValid:
		// Complete the OAuth flow.
		cmds = append(cmds, util.CmdHandler(OAuthCompleteMsg{Token: o.token}))
	}

	o.updatePrompt()
	return o, tea.Batch(cmds...)
}

// View renders the OAuth2 flow.
func (o *OAuth2Flow) View() string {
	t := styles.CurrentTheme()

	if o.err != nil {
		return t.S().Error.Render(o.err.Error())
	}

	switch o.state {
	case OAuthStateURL:
		heading := t.S().Title.Render("Press Enter to open the authorization URL in your browser:")
		displayURL := o.displayURL()
		urlText := t.S().Muted.Render(displayURL)

		return lipgloss.JoinVertical(lipgloss.Left,
			heading,
			"",
			urlText,
		)

	case OAuthStateCode:
		var heading string

		switch o.validationState {
		case OAuthValidationStateNone:
			heading = t.S().Title.Render("Enter the ") +
				t.S().Success.Render("code") +
				t.S().Title.Render(" you received:")
		case OAuthValidationStateVerifying:
			heading = t.S().Title.Render("Verifying...")
		case OAuthValidationStateValid:
			heading = t.S().Success.Render("Validated! Press Enter to continue.")
		case OAuthValidationStateError:
			heading = t.S().Error.Render("Invalid code. Try again?")
		}

		return lipgloss.JoinVertical(lipgloss.Left,
			heading,
			"",
			o.codeInput.View(),
		)

	default:
		return "Unknown state"
	}
}

// SetWidth sets the component width.
func (o *OAuth2Flow) SetWidth(w int) {
	o.width = w
	o.codeInput.SetWidth(w - 4)
}

// Token returns the OAuth token if validated.
func (o *OAuth2Flow) Token() *oauth.Token {
	return o.token
}

// IsComplete returns true if OAuth validation is complete and successful.
func (o *OAuth2Flow) IsComplete() bool {
	return o.validationState == OAuthValidationStateValid
}

// IsURLState returns true if in URL display state.
func (o *OAuth2Flow) IsURLState() bool {
	return o.state == OAuthStateURL
}

// Cursor returns the cursor for the text input.
func (o *OAuth2Flow) Cursor() *tea.Cursor {
	if o.state == OAuthStateCode {
		return o.codeInput.Cursor()
	}
	return nil
}

func (o *OAuth2Flow) validateCode() tea.Msg {
	token, err := claude.ExchangeToken(context.Background(), o.codeInput.Value(), o.verifier)
	if err != nil || token == nil {
		return OAuthValidationCompletedMsg{State: OAuthValidationStateError}
	}
	return OAuthValidationCompletedMsg{State: OAuthValidationStateValid, Token: token}
}

func (o *OAuth2Flow) updatePrompt() {
	switch o.validationState {
	case OAuthValidationStateNone:
		o.codeInput.Prompt = "> "
	case OAuthValidationStateVerifying:
		o.codeInput.Prompt = o.spinner.View() + " "
	case OAuthValidationStateValid:
		o.codeInput.Prompt = styles.CheckIcon + " "
	case OAuthValidationStateError:
		o.codeInput.Prompt = styles.ErrorIcon + " "
	}
}

// displayURL returns a shortened URL for display (without query params).
func (o *OAuth2Flow) displayURL() string {
	parsed, err := url.Parse(o.authURL)
	if err != nil {
		return o.authURL
	}

	if parsed.RawQuery != "" {
		parsed.RawQuery = ""
		return parsed.String() + "..."
	}

	return o.authURL
}

// openBrowserSilent opens a URL in the browser without outputting to stdout/stderr.
// This prevents disruption to the TUI.
func openBrowserSilent(targetURL string) {
	var cmd *exec.Cmd
	ctx := context.Background()

	switch runtime.GOOS {
	case "linux":
		cmd = exec.CommandContext(ctx, "xdg-open", targetURL)
	case "darwin":
		cmd = exec.CommandContext(ctx, "open", targetURL)
	case "windows":
		cmd = exec.CommandContext(ctx, "rundll32", "url.dll,FileProtocolHandler", targetURL)
	default:
		return
	}

	// Redirect stdout and stderr to /dev/null to avoid TUI disruption.
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil

	// Detach from the process group so it doesn't receive signals.
	if f, err := os.Open(os.DevNull); err == nil {
		cmd.Stdout = f
		cmd.Stderr = f
		defer f.Close() //nolint:errcheck // Best effort close.
	}

	_ = cmd.Start() //nolint:errcheck // Best effort open.
}
