// Package message provides message management with persistence.
package message

import (
	"time"
)

// Role represents the role of a message sender.
type Role string

// Role constants.
const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
	RoleTool      Role = "tool"
)

// Message represents a conversation message.
type Message struct {
	ID        string
	SessionID string
	Role      Role
	Parts     []Part
	Model     string
	Provider  string
	IsSummary bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// PartType represents the type of a message part.
type PartType string

// Part type constants.
const (
	PartTypeText       PartType = "text"
	PartTypeReasoning  PartType = "reasoning"
	PartTypeToolCall   PartType = "tool_call"
	PartTypeToolResult PartType = "tool_result"
)

// Part represents a content part of a message.
type Part struct {
	Type       PartType    `json:"type"`
	Text       string      `json:"text,omitempty"`
	Reasoning  string      `json:"reasoning,omitempty"`
	ToolCall   *ToolCall   `json:"tool_call,omitempty"`
	ToolResult *ToolResult `json:"tool_result,omitempty"`
}

// ToolCall represents a tool invocation.
type ToolCall struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Input string `json:"input"`
}

// ToolResult represents the result of a tool invocation.
type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Name       string `json:"name"`
	Content    string `json:"content"`
	IsError    bool   `json:"is_error"`
}

// TextContent returns the concatenated text content from all text parts.
func (m *Message) TextContent() string {
	for _, p := range m.Parts {
		if p.Type == PartTypeText {
			return p.Text
		}
	}
	return ""
}

// ReasoningContent returns the reasoning content from the message parts.
func (m *Message) ReasoningContent() string {
	for _, p := range m.Parts {
		if p.Type == PartTypeReasoning {
			return p.Reasoning
		}
	}
	return ""
}

// ToolCalls returns all tool calls from the message parts.
func (m *Message) ToolCalls() []*ToolCall {
	var calls []*ToolCall
	for _, p := range m.Parts {
		if p.Type == PartTypeToolCall && p.ToolCall != nil {
			calls = append(calls, p.ToolCall)
		}
	}
	return calls
}

// ToolResults returns all tool results from the message parts.
func (m *Message) ToolResults() []*ToolResult {
	var results []*ToolResult
	for _, p := range m.Parts {
		if p.Type == PartTypeToolResult && p.ToolResult != nil {
			results = append(results, p.ToolResult)
		}
	}
	return results
}

// NewTextPart creates a new text part.
func NewTextPart(text string) Part {
	return Part{
		Type: PartTypeText,
		Text: text,
	}
}

// NewReasoningPart creates a new reasoning part.
func NewReasoningPart(reasoning string) Part {
	return Part{
		Type:      PartTypeReasoning,
		Reasoning: reasoning,
	}
}

// NewToolCallPart creates a new tool call part.
func NewToolCallPart(id, name, input string) Part {
	return Part{
		Type: PartTypeToolCall,
		ToolCall: &ToolCall{
			ID:    id,
			Name:  name,
			Input: input,
		},
	}
}

// NewToolResultPart creates a new tool result part.
func NewToolResultPart(toolCallID, name, content string, isError bool) Part {
	return Part{
		Type: PartTypeToolResult,
		ToolResult: &ToolResult{
			ToolCallID: toolCallID,
			Name:       name,
			Content:    content,
			IsError:    isError,
		},
	}
}
