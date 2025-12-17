// Package util provides shared utilities for TUI components.
package util //nolint:revive // "util" is a common and meaningful name for shared utilities.

import (
	"log/slog"
	"time"

	tea "charm.land/bubbletea/v2"
)

// Cursor is an interface for components that have a cursor.
type Cursor interface {
	Cursor() *tea.Cursor
}

// Model is the interface for TUI components.
type Model interface {
	Init() tea.Cmd
	Update(tea.Msg) (Model, tea.Cmd)
	View() string
}

// CmdHandler wraps a message in a command.
func CmdHandler(msg tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return msg
	}
}

// ReportError reports an error as an InfoMsg.
func ReportError(err error) tea.Cmd {
	slog.Error("Error reported", "error", err)
	return CmdHandler(InfoMsg{
		Type: InfoTypeError,
		Msg:  err.Error(),
	})
}

// InfoType represents the type of info message.
type InfoType int

// Info message types.
const (
	InfoTypeInfo InfoType = iota
	InfoTypeSuccess
	InfoTypeWarn
	InfoTypeError
)

// ReportInfo reports an info message.
func ReportInfo(info string) tea.Cmd {
	return CmdHandler(InfoMsg{
		Type: InfoTypeInfo,
		Msg:  info,
	})
}

// ReportSuccess reports a success message.
func ReportSuccess(info string) tea.Cmd {
	return CmdHandler(InfoMsg{
		Type: InfoTypeSuccess,
		Msg:  info,
	})
}

// ReportWarn reports a warning message.
func ReportWarn(warn string) tea.Cmd {
	return CmdHandler(InfoMsg{
		Type: InfoTypeWarn,
		Msg:  warn,
	})
}

// InfoMsg is a message for displaying info to the user.
type InfoMsg struct {
	Msg  string
	Type InfoType
	TTL  time.Duration
}

// ClearStatusMsg clears the status bar.
type ClearStatusMsg struct{}
