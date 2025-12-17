// Package logo renders the CDD wordmark.
package logo

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/guilhermegouw/cdd/internal/tui/styles"
)

// ASCII art for CDD logo.
const cddLogo = `
 ██████╗██████╗ ██████╗
██╔════╝██╔══██╗██╔══██╗
██║     ██║  ██║██║  ██║
██║     ██║  ██║██║  ██║
╚██████╗██████╔╝██████╔╝
 ╚═════╝╚═════╝ ╚═════╝
`

// Smaller logo for narrow spaces.
const cddLogoSmall = `
╔═╗╔╦╗╔╦╗
║   ║║ ║║
╚═╝═╩╝═╩╝
`

// Render returns the CDD logo with the current theme colors.
func Render() string {
	t := styles.CurrentTheme()
	logo := strings.TrimPrefix(cddLogo, "\n")

	// Apply gradient from primary to secondary color.
	return styles.ApplyForegroundGrad(logo, t.Primary, t.Secondary)
}

// RenderSmall returns a smaller version of the logo.
func RenderSmall() string {
	t := styles.CurrentTheme()
	logo := strings.TrimPrefix(cddLogoSmall, "\n")
	return styles.ApplyForegroundGrad(logo, t.Primary, t.Secondary)
}

// RenderWithTagline returns the logo with a tagline.
func RenderWithTagline() string {
	t := styles.CurrentTheme()
	logo := Render()

	tagline := t.S().Muted.Render("Context-Driven Development")

	return lipgloss.JoinVertical(lipgloss.Center, logo, "", tagline)
}

// Width returns the width of the full logo.
func Width() int {
	return lipgloss.Width(cddLogo)
}

// Height returns the height of the full logo.
func Height() int {
	return lipgloss.Height(cddLogo)
}
