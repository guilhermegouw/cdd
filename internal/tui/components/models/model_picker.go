package models

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/catwalk/pkg/catwalk"

	"github.com/guilhermegouw/cdd/internal/config"
	"github.com/guilhermegouw/cdd/internal/tui/styles"
	"github.com/guilhermegouw/cdd/internal/tui/util"
)

// ModelPicker displays a list of models for a connection.
type ModelPicker struct {
	cfg        *config.Config
	connection *config.Connection
	models     []catwalk.Model
	cursor     int
	width      int
	height     int
}

// NewModelPicker creates a new ModelPicker.
func NewModelPicker(cfg *config.Config) *ModelPicker {
	return &ModelPicker{
		cfg:    cfg,
		cursor: 0,
	}
}

// SetConnection sets the connection to pick models from.
func (p *ModelPicker) SetConnection(conn *config.Connection) {
	p.connection = conn
	p.cursor = 0
	p.models = nil

	// First try provider config (may have user-configured models).
	if provider, ok := p.cfg.Providers[conn.ProviderID]; ok && len(provider.Models) > 0 {
		p.models = provider.Models
		return
	}

	// Fall back to known providers from catwalk.
	known := p.cfg.KnownProviders()
	for i := range known {
		if string(known[i].ID) == conn.ProviderID {
			p.models = known[i].Models
			return
		}
	}
}

// SetSize sets the component size.
func (p *ModelPicker) SetSize(width, height int) {
	p.width = width
	p.height = height
}

// Update handles messages.
func (p *ModelPicker) Update(msg tea.Msg) (*ModelPicker, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "up", "k":
			if p.cursor > 0 {
				p.cursor--
			}
			return p, nil

		case keyDown, "j":
			if p.cursor < len(p.models)-1 {
				p.cursor++
			}
			return p, nil

		case keyEnter:
			if p.cursor >= 0 && p.cursor < len(p.models) && p.connection != nil {
				model := p.models[p.cursor]
				return p, util.CmdHandler(ModelSelectedMsg{
					ConnectionID: p.connection.ID,
					ModelID:      model.ID,
				})
			}
			return p, nil
		}
	}

	return p, nil
}

// View renders the model list.
func (p *ModelPicker) View() string {
	t := styles.CurrentTheme()

	if p.connection == nil {
		return t.S().Muted.Render("No connection selected.")
	}

	if len(p.models) == 0 {
		return t.S().Muted.Render("No models available for this provider.")
	}

	var sb strings.Builder

	sb.WriteString(t.S().Muted.Render("Connection: "))
	sb.WriteString(t.S().Primary.Render(p.connection.Name))
	sb.WriteString("\n\n")

	sb.WriteString(t.S().Muted.Render("Select a model:"))
	sb.WriteString("\n\n")

	for i := range p.models {
		modelName := p.models[i].Name
		if modelName == "" {
			modelName = p.models[i].ID
		}

		// Build the line content.
		var line string
		if p.models[i].Name != "" && p.models[i].Name != p.models[i].ID {
			line = modelName + " (" + p.models[i].ID + ")"
		} else {
			line = modelName
		}

		// Render with cursor and styling.
		if i == p.cursor {
			sb.WriteString("> ")
			sb.WriteString(t.S().Primary.Bold(true).Render(line))
		} else {
			sb.WriteString("  ")
			sb.WriteString(t.S().Muted.Render(line))
		}
		sb.WriteString("\n")
	}

	// Add help.
	sb.WriteString("\n")
	sb.WriteString(t.S().Muted.Render("[enter] select  [esc] back"))

	return sb.String()
}

// Selected returns the currently selected model.
func (p *ModelPicker) Selected() *catwalk.Model {
	if p.cursor >= 0 && p.cursor < len(p.models) {
		return &p.models[p.cursor]
	}
	return nil
}
