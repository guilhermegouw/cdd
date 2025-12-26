// Package models provides the models/connections management modal component.
package models

import "github.com/guilhermegouw/cdd/internal/config"

// Modal control messages.
type (
	// CloseModalMsg requests closing the modal.
	CloseModalMsg struct{}

	// ModalClosedMsg is sent when the modal is closed.
	ModalClosedMsg struct{}
)

// Navigation messages.
type (
	// StartAddConnectionMsg starts the add connection flow.
	StartAddConnectionMsg struct{}

	// EditConnectionMsg requests editing a connection.
	EditConnectionMsg struct {
		ID string
	}

	// DeleteConnectionMsg requests deleting a connection.
	DeleteConnectionMsg struct {
		ID string
	}

	// ConfirmDeleteMsg confirms deletion of a connection.
	ConfirmDeleteMsg struct {
		ID string
	}

	// CancelDeleteMsg cancels the delete confirmation.
	CancelDeleteMsg struct{}

	// BackMsg navigates back to the previous step.
	BackMsg struct{}
)

// Connection CRUD messages.
type (
	// ConnectionAddedMsg is sent when a connection is added.
	ConnectionAddedMsg struct {
		Connection config.Connection
	}

	// ConnectionUpdatedMsg is sent when a connection is updated.
	ConnectionUpdatedMsg struct {
		Connection config.Connection
	}

	// ConnectionDeletedMsg is sent when a connection is deleted.
	ConnectionDeletedMsg struct {
		ID string
	}

	// ConnectionSelectedMsg is sent when a connection is selected.
	ConnectionSelectedMsg struct {
		Connection config.Connection
	}
)

// Provider selection messages.
type (
	// ProviderSelectedMsg is sent when a provider is selected for a new connection.
	ProviderSelectedMsg struct {
		ProviderID   string
		ProviderName string
		ProviderType string
	}
)

// Model selection messages.
type (
	// SelectLargeModelMsg starts the large model selection flow.
	SelectLargeModelMsg struct{}

	// SelectSmallModelMsg starts the small model selection flow.
	SelectSmallModelMsg struct{}

	// ModelSelectedMsg is sent when a model is selected for a tier.
	ModelSelectedMsg struct {
		Tier         config.SelectedModelType
		ConnectionID string
		ModelID      string
	}

	// ModelSwitchedMsg is sent to the parent when the active model should be switched.
	// The parent (chat page) should reload the model using modelFactory.
	ModelSwitchedMsg struct {
		Tier         config.SelectedModelType
		ConnectionID string
		ModelID      string
		ModelName    string
	}
)

// Form messages.
type (
	// FormSubmitMsg is sent when a form is submitted.
	FormSubmitMsg struct {
		Name   string
		APIKey string
	}

	// FormCancelMsg is sent when a form is cancelled.
	FormCancelMsg struct{}
)
