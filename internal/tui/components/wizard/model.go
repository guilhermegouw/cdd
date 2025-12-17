package wizard

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/catwalk/pkg/catwalk"

	"github.com/guilhermegouw/cdd/internal/tui/styles"
	"github.com/guilhermegouw/cdd/internal/tui/util"
)

// ModelSelectedMsg is sent when a model is selected.
type ModelSelectedMsg struct {
	Model catwalk.Model
	Tier  string // "large" or "small"
}

// ModelList displays a list of models to select from.
type ModelList struct {
	tier     string
	provider string
	models   []catwalk.Model
	cursor   int
	width    int
	height   int
}

// NewModelList creates a new model list component.
func NewModelList(models []catwalk.Model, tier, provider string) *ModelList {
	return &ModelList{
		models:   models,
		cursor:   0,
		tier:     tier,
		provider: provider,
	}
}

// Init initializes the component.
func (m *ModelList) Init() tea.Cmd {
	return nil
}

// Update handles messages.
func (m *ModelList) Update(msg tea.Msg) (util.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case keyUp, keyK:
		if m.cursor > 0 {
			m.cursor--
		}
	case keyDown, keyJ:
		if m.cursor < len(m.models)-1 {
			m.cursor++
		}
	case keyEnter:
		if len(m.models) > 0 {
			return m, util.CmdHandler(ModelSelectedMsg{
				Model: m.models[m.cursor],
				Tier:  m.tier,
			})
		}
	}
	return m, nil
}

// View renders the model list.
func (m *ModelList) View() string {
	t := styles.CurrentTheme()

	tierDisplay := "Large"
	tierDesc := "for complex reasoning tasks"
	if m.tier == "small" {
		tierDisplay = "Small"
		tierDesc = "for faster, simpler tasks"
	}

	title := t.S().Title.Render(fmt.Sprintf("Select %s Model", tierDisplay))
	subtitle := t.S().Muted.Render(fmt.Sprintf("(%s)", tierDesc))
	help := t.S().Muted.Render("Use ↑/↓ to navigate, Enter to select")

	items := make([]string, 0, len(m.models))
	for i := range m.models {
		cursor := "  "
		style := t.S().Text

		if i == m.cursor {
			cursor = t.S().Success.Render(styles.Selected + " ")
			style = t.S().Text.Bold(true)
		}

		name := style.Render(m.models[i].Name)
		id := t.S().Subtle.Render(fmt.Sprintf(" (%s)", m.models[i].ID))
		items = append(items, cursor+name+id)
	}

	list := strings.Join(items, "\n")

	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		subtitle,
		"",
		list,
		"",
		help,
	)
}

// SetSize sets the component size.
func (m *ModelList) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// SelectedModel returns the currently selected model.
func (m *ModelList) SelectedModel() *catwalk.Model {
	if len(m.models) == 0 {
		return nil
	}
	return &m.models[m.cursor]
}

// SetModels updates the list of models.
func (m *ModelList) SetModels(models []catwalk.Model) {
	m.models = models
	m.cursor = 0
}

// SetCursorToModel moves cursor to a specific model by ID.
func (m *ModelList) SetCursorToModel(modelID string) {
	for i := range m.models {
		if m.models[i].ID == modelID {
			m.cursor = i
			return
		}
	}
}
