package message

import (
	"encoding/json"
	"testing"
)

func TestMessage_TextContent(t *testing.T) {
	tests := []struct {
		name  string
		parts []Part
		want  string
	}{
		{
			name:  "returns text from text part",
			parts: []Part{NewTextPart("Hello world")},
			want:  "Hello world",
		},
		{
			name:  "returns first text part",
			parts: []Part{NewReasoningPart("thinking"), NewTextPart("result")},
			want:  "result",
		},
		{
			name:  "returns empty when no text part",
			parts: []Part{NewReasoningPart("thinking")},
			want:  "",
		},
		{
			name:  "returns empty for nil parts",
			parts: nil,
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Message{Parts: tt.parts}
			if got := m.TextContent(); got != tt.want {
				t.Errorf("TextContent() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMessage_ReasoningContent(t *testing.T) {
	tests := []struct {
		name  string
		parts []Part
		want  string
	}{
		{
			name:  "returns reasoning from reasoning part",
			parts: []Part{NewReasoningPart("thinking deeply")},
			want:  "thinking deeply",
		},
		{
			name:  "returns empty when no reasoning part",
			parts: []Part{NewTextPart("text only")},
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Message{Parts: tt.parts}
			if got := m.ReasoningContent(); got != tt.want {
				t.Errorf("ReasoningContent() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMessage_ToolCalls(t *testing.T) {
	t.Run("returns tool calls", func(t *testing.T) {
		m := &Message{
			Parts: []Part{
				NewToolCallPart("call-1", "read_file", `{"path": "/tmp/test"}`),
				NewTextPart("some text"),
				NewToolCallPart("call-2", "write_file", `{"path": "/tmp/out"}`),
			},
		}

		calls := m.ToolCalls()
		if len(calls) != 2 {
			t.Fatalf("ToolCalls() returned %d calls, want 2", len(calls))
		}

		if calls[0].ID != "call-1" { //nolint:goconst // Test literals are intentionally readable
			t.Errorf("calls[0].ID = %q, want %q", calls[0].ID, "call-1")
		}
		if calls[0].Name != "read_file" { //nolint:goconst // Test literals are intentionally readable
			t.Errorf("calls[0].Name = %q, want %q", calls[0].Name, "read_file")
		}
		if calls[1].ID != "call-2" {
			t.Errorf("calls[1].ID = %q, want %q", calls[1].ID, "call-2")
		}
	})

	t.Run("returns empty when no tool calls", func(t *testing.T) {
		m := &Message{Parts: []Part{NewTextPart("text only")}}
		if calls := m.ToolCalls(); len(calls) != 0 {
			t.Errorf("ToolCalls() returned %d calls, want 0", len(calls))
		}
	})
}

func TestMessage_ToolResults(t *testing.T) {
	t.Run("returns tool results", func(t *testing.T) {
		m := &Message{
			Parts: []Part{
				NewToolResultPart("call-1", "read_file", "file contents", false),
				NewToolResultPart("call-2", "write_file", "error: permission denied", true),
			},
		}

		results := m.ToolResults()
		if len(results) != 2 {
			t.Fatalf("ToolResults() returned %d results, want 2", len(results))
		}

		if results[0].ToolCallID != "call-1" {
			t.Errorf("results[0].ToolCallID = %q, want %q", results[0].ToolCallID, "call-1")
		}
		if results[0].IsError {
			t.Error("results[0].IsError = true, want false")
		}
		if !results[1].IsError {
			t.Error("results[1].IsError = false, want true")
		}
	})
}

func TestNewTextPart(t *testing.T) {
	part := NewTextPart("hello")

	if part.Type != PartTypeText {
		t.Errorf("Type = %q, want %q", part.Type, PartTypeText)
	}
	if part.Text != "hello" {
		t.Errorf("Text = %q, want %q", part.Text, "hello")
	}
}

func TestNewReasoningPart(t *testing.T) {
	part := NewReasoningPart("thinking")

	if part.Type != PartTypeReasoning {
		t.Errorf("Type = %q, want %q", part.Type, PartTypeReasoning)
	}
	if part.Reasoning != "thinking" {
		t.Errorf("Reasoning = %q, want %q", part.Reasoning, "thinking")
	}
}

func TestNewToolCallPart(t *testing.T) {
	part := NewToolCallPart("id-1", "read_file", `{"path": "/tmp"}`)

	if part.Type != PartTypeToolCall {
		t.Errorf("Type = %q, want %q", part.Type, PartTypeToolCall)
	}
	if part.ToolCall == nil {
		t.Fatal("ToolCall is nil")
	}
	if part.ToolCall.ID != "id-1" { //nolint:goconst // Test literals are intentionally readable
		t.Errorf("ToolCall.ID = %q, want %q", part.ToolCall.ID, "id-1")
	}
	if part.ToolCall.Name != "read_file" {
		t.Errorf("ToolCall.Name = %q, want %q", part.ToolCall.Name, "read_file")
	}
	if part.ToolCall.Input != `{"path": "/tmp"}` {
		t.Errorf("ToolCall.Input = %q, want %q", part.ToolCall.Input, `{"path": "/tmp"}`)
	}
}

func TestNewToolResultPart(t *testing.T) {
	t.Run("success result", func(t *testing.T) {
		part := NewToolResultPart("call-1", "read_file", "contents", false)

		if part.Type != PartTypeToolResult {
			t.Errorf("Type = %q, want %q", part.Type, PartTypeToolResult)
		}
		if part.ToolResult == nil {
			t.Fatal("ToolResult is nil")
		}
		if part.ToolResult.ToolCallID != "call-1" {
			t.Errorf("ToolCallID = %q, want %q", part.ToolResult.ToolCallID, "call-1")
		}
		if part.ToolResult.IsError {
			t.Error("IsError = true, want false")
		}
	})

	t.Run("error result", func(t *testing.T) {
		part := NewToolResultPart("call-1", "read_file", "error msg", true)
		if !part.ToolResult.IsError {
			t.Error("IsError = false, want true")
		}
	})
}

func TestPart_JSONSerialization(t *testing.T) {
	parts := []Part{
		NewTextPart("hello"),
		NewReasoningPart("thinking"),
		NewToolCallPart("id-1", "tool", "input"),
		NewToolResultPart("id-1", "tool", "output", false),
	}

	// Serialize
	data, err := json.Marshal(parts)
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}

	// Deserialize
	var decoded []Part
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	if len(decoded) != len(parts) {
		t.Fatalf("decoded %d parts, want %d", len(decoded), len(parts))
	}

	// Verify each part
	if decoded[0].Type != PartTypeText || decoded[0].Text != "hello" {
		t.Errorf("text part mismatch: %+v", decoded[0])
	}
	if decoded[1].Type != PartTypeReasoning || decoded[1].Reasoning != "thinking" {
		t.Errorf("reasoning part mismatch: %+v", decoded[1])
	}
	if decoded[2].Type != PartTypeToolCall || decoded[2].ToolCall.ID != "id-1" {
		t.Errorf("tool call part mismatch: %+v", decoded[2])
	}
	if decoded[3].Type != PartTypeToolResult || decoded[3].ToolResult.ToolCallID != "id-1" {
		t.Errorf("tool result part mismatch: %+v", decoded[3])
	}
}
