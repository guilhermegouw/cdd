// Package page defines page types for the TUI navigation.
package page

// ID identifies a page in the TUI.
type ID string

const (
	// Welcome is the welcome/splash screen.
	Welcome ID = "welcome"
	// Wizard is the setup wizard.
	Wizard ID = "wizard"
	// Main is the main application page.
	Main ID = "main"
)

// ChangeMsg is used to change the current page.
type ChangeMsg struct {
	Page ID
}
