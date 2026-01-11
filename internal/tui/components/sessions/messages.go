package sessions

// ModalClosedMsg is sent when the modal is closed.
type ModalClosedMsg struct{}

// SwitchSessionMsg is sent to switch to a different session.
type SwitchSessionMsg struct {
	SessionID string
}

// SessionSelectedMsg is sent when a session is selected from the list.
type SessionSelectedMsg struct {
	SessionID string
}

// RenameSessionMsg is sent to start renaming a session.
type RenameSessionMsg struct {
	SessionID    string
	CurrentTitle string
}

// DeleteSessionMsg is sent to confirm session deletion.
type DeleteSessionMsg struct {
	SessionID string
}

// ExportSessionMsg is sent to start export flow.
type ExportSessionMsg struct {
	SessionID string
}

// ExportMarkdownMsg is sent to export session to markdown.
type ExportMarkdownMsg struct {
	SessionID string
}

// NewSessionMsg is sent to create a new session.
type NewSessionMsg struct{}

// GenerateTitleMsg is sent to request LLM title generation.
type GenerateTitleMsg struct {
	SessionID string
}

// RequestTitleGenerationMsg is sent to the parent to request LLM title generation.
type RequestTitleGenerationMsg struct {
	SessionID string
}

// TitleGeneratedMsg is sent when LLM generates a title.
type TitleGeneratedMsg struct {
	SessionID string
	Title     string
}
