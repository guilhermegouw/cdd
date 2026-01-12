package sessions

import (
	"charm.land/lipgloss/v2"

	"github.com/guilhermegouw/cdd/internal/tui/styles"
)

// HintMode represents the current mode for hint display.
type HintMode int

const (
	// HintModeNormal shows hints for normal browsing mode.
	HintModeNormal HintMode = iota
	// HintModeSearch shows hints for search mode.
	HintModeSearch
	// HintModeRename shows hints for rename mode.
	HintModeRename
	// HintModeDelete shows hints for delete confirmation mode.
	HintModeDelete
	// HintModeExport shows hints for export mode.
	HintModeExport
)

// HintBar displays context-sensitive keyboard hints.
type HintBar struct {
	mode  HintMode
	width int
}

// NewHintBar creates a new hint bar.
func NewHintBar() *HintBar {
	return &HintBar{
		mode: HintModeNormal,
	}
}

// SetMode sets the current hint mode.
func (h *HintBar) SetMode(mode HintMode) {
	h.mode = mode
}

// SetWidth sets the hint bar width.
func (h *HintBar) SetWidth(width int) {
	h.width = width
}

// View renders the hint bar.
func (h *HintBar) View() string {
	t := styles.CurrentTheme()

	var hints string
	switch h.mode {
	case HintModeNormal:
		hints = "[/] search  [n] new  [enter] open  [r] rename  [d] delete  [esc] close"
	case HintModeSearch:
		hints = "[enter] done  [esc] clear  [↑↓] navigate"
	case HintModeRename:
		hints = "[enter] save  [esc] cancel"
	case HintModeDelete:
		hints = "[y] yes  [n] no  [esc] cancel"
	case HintModeExport:
		hints = "[m] markdown  [esc] cancel"
	}

	hintStyle := t.S().Muted.
		Width(h.width).
		Align(lipgloss.Center)

	return hintStyle.Render(hints)
}
