# TUI Activity Panel & Chat Improvements

> Clean, focused chat experience with live tool activity feedback inspired by Claude Code.

---

## Table of Contents

1. [Overview](#overview)
2. [Problem Statement](#problem-statement)
3. [Design Goals](#design-goals)
4. [Visual Design](#visual-design)
5. [Component Architecture](#component-architecture)
6. [Implementation Steps](#implementation-steps)
7. [Testing Strategy](#testing-strategy)
8. [Future Enhancements](#future-enhancements)

---

## Overview

This plan introduces an **Activity Panel** component to the TUI chat interface, providing real-time feedback during AI interactions while keeping the message viewport clean and focused on actual content.

### Key Changes

| Area | Current State | Target State |
|------|---------------|--------------|
| Tool feedback | Dumped in message body | Live activity panel |
| Thinking status | Static "Thinking..." | Animated spinner |
| Status bar | Generic shortcuts | Model name + context-aware hints |
| Message content | Cluttered with tool results | Clean response only |

---

## Problem Statement

### Current Issues

1. **Tool results clutter responses** - Full tool outputs (up to 500 chars each) appear at the end of assistant messages, making it hard to see where the actual answer ends.

2. **No visual activity feedback** - Static "Thinking..." text provides no sense of progress or activity.

3. **Lost context** - Users can't easily see which tools were used or what operations happened.

4. **Status bar underutilized** - Shows generic shortcuts, doesn't indicate active model.

### User Experience Goal

The user should:
- See a clean, readable response in the chat
- Know something is happening via animation
- See tool operations as they occur (live feed)
- Access details on-demand if needed (future: modal)

---

## Design Goals

| Goal | Description |
|------|-------------|
| **Clarity** | Separate content from activity/metadata |
| **Feedback** | Always show something is happening during operations |
| **Non-intrusive** | Activity panel hides when idle (zero height) |
| **Performance** | Minimal overhead, efficient updates |
| **Extensibility** | Easy to add more activity types later |

---

## Visual Design

### Complete Layout (During Streaming)

```
┌─────────────────────────────────────────────────────────────────────┐
│                                                                     │
│                         Messages Viewport                           │
│                                                                     │
│  You                                                                │
│  What files are in the agent package?                               │
│                                                                     │
│  Assistant                                                          │
│  Here's what I found in the agent package. The main files are:      │
│  - agent.go - Core interface definitions                            │
│  - loop.go - Main agent loop implementation                         │
│  [response continues...]                                            │
│                                                                     │
├─────────────────────────────────────────────────────────────────────┤
│  ⠋ Thinking...                                                      │
│    ├─ read: internal/agent/loop.go                                  │
│    ├─ read: internal/agent/agent.go                                 │
│    └─ grep: "func.*" in *.go                                        │
├─────────────────────────────────────────────────────────────────────┤
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ Type a message... (Ctrl+J for newline)                        │  │
│  └───────────────────────────────────────────────────────────────┘  │
├─────────────────────────────────────────────────────────────────────┤
│  claude-sonnet-4-20250514        Enter send · Esc cancel · Ctrl+C quit  │
└─────────────────────────────────────────────────────────────────────┘
```

### Layout When Idle

```
┌─────────────────────────────────────────────────────────────────────┐
│                                                                     │
│                         Messages Viewport                           │
│                         (more vertical space)                       │
│                                                                     │
├─────────────────────────────────────────────────────────────────────┤
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ Type a message... (Ctrl+J for newline)                        │  │
│  └───────────────────────────────────────────────────────────────┘  │
├─────────────────────────────────────────────────────────────────────┤
│  claude-sonnet-4-20250514        Enter send · Esc cancel · Ctrl+C quit  │
└─────────────────────────────────────────────────────────────────────┘
```

### Spinner Animation Frames

```go
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
```

### Tool Line Format

```
├─ read: internal/agent/loop.go
├─ grep: "pattern" in *.go
├─ edit: file.go (lines 45-52)  
├─ bash: go test ./...
└─ write: internal/new_file.go
```

Each tool shows:
- **Tool name** - The operation type
- **Context** - Key parameter (path, pattern, command) - truncated if needed

---

## Component Architecture

### New Files

```
internal/tui/page/chat/
├── activity.go        # NEW: Activity panel component
├── activity_test.go   # NEW: Activity panel tests
├── chat.go            # MODIFY: Layout integration
├── messages.go        # MODIFY: Remove tool result rendering
├── status.go          # MODIFY: Add model name, update shortcuts
└── ...
```

### Activity Panel Component

```go
// internal/tui/page/chat/activity.go

package chat

// ActivityPanel shows real-time activity during AI interactions.
type ActivityPanel struct {
    spinner      int           // Current spinner frame index
    thinking     bool          // Whether we're in thinking state
    tools        []ToolActivity // Active/recent tool calls
    width        int
    maxTools     int           // Max visible tools (default: 5)
}

// ToolActivity represents a single tool operation.
type ToolActivity struct {
    Name     string    // Tool name (read, grep, edit, etc.)
    Summary  string    // Brief description (file path, pattern, etc.)
    Status   ToolStatus // pending, running, done, error
}

type ToolStatus int

const (
    ToolStatusPending ToolStatus = iota
    ToolStatusRunning
    ToolStatusDone
    ToolStatusError
)

// Key methods:
// - SetThinking(bool)
// - AddTool(name, summary string)
// - MarkToolDone(name string)
// - Clear()
// - Height() int  // Returns 0 when hidden
// - View() string
// - Update(tea.Msg) (*ActivityPanel, tea.Cmd)
```

### Message Types

```go
// Spinner tick message
type SpinnerTickMsg struct{}

// Tool activity messages (may reuse existing or create new)
type ToolStartedMsg struct {
    Name    string
    Summary string
}

type ToolDoneMsg struct {
    Name string
}
```

### Integration with Chat Model

```go
// chat.go - Updated View layout

func (m *Model) View() string {
    // ... setup ...

    messagesView := m.messages.View()
    activityView := m.activity.View()  // NEW
    inputView := m.input.View()
    statusView := m.status.View()

    // Only add separator + activity if activity has content
    var parts []string
    parts = append(parts, messagesView)
    
    if m.activity.Height() > 0 {
        parts = append(parts, separator, activityView)
    }
    
    parts = append(parts, separator, inputView, statusView)

    return lipgloss.JoinVertical(lipgloss.Left, parts...)
}
```

---

## Implementation Steps

### Phase 1: Activity Panel Foundation
**Files:** `activity.go`, `activity_test.go`

- [x] Create `ActivityPanel` struct with state fields
- [x] Implement spinner animation with tick messages
- [x] Implement `View()` with proper formatting
- [x] Implement `Height()` returning 0 when empty
- [x] Implement `SetThinking()`, `AddTool()`, `Clear()`
- [x] Add unit tests for state management
- [x] Add unit tests for view rendering

### Phase 2: Tool Summary Extraction
**Files:** `activity.go`

- [x] Create `toolSummary(name string, input string) string` function
- [x] Handle each tool type:
  - `read` → file path
  - `grep` → `"pattern" in *.ext`
  - `glob` → `pattern in path`
  - `edit` → `file:lines` or `file (N changes)`
  - `write` → file path
  - `bash` → command (truncated)
- [x] Add tests for summary extraction

### Phase 3: Chat Integration
**Files:** `chat.go`

- [x] Add `activity *ActivityPanel` field to `Model`
- [x] Initialize in `New()`
- [x] Update `View()` layout to include activity panel
- [x] Update `messagesAreaHeight()` to account for activity height
- [x] Wire up `SpinnerTickMsg` handling
- [x] Start spinner on `StreamTextMsg` / thinking state
- [x] Add tools on `StreamToolCallMsg`
- [x] Clear activity on `StreamCompleteMsg`
- [x] Update `SetSize()` to propagate width

### Phase 4: Clean Up Message Rendering
**Files:** `messages.go`

- [x] Remove `renderToolMessage()` or simplify significantly
- [x] Remove tool call display from `renderAssistantMessage()`
- [x] Optionally add subtle indicator: `⚡ 3 tools used`
- [x] Update tests if any

### Phase 5: Status Bar Updates
**Files:** `status.go`, `chat.go`

- [x] Add `modelName string` field to `StatusBar`
- [x] Add `SetModelName(name string)` method
- [x] Update `View()` to show model name on left
- [x] Update shortcuts text: `Enter send · Esc cancel · Ctrl+C quit`
- [x] Pass model name from chat to status bar
- [x] Remove or repurpose "Ready" / "Thinking" status (moved to activity)

### Phase 6: Testing & Polish

- [ ] Manual testing of full flow
- [ ] Test edge cases:
  - Many tools (scrolling/truncation)
  - Long file paths (truncation)
  - Rapid tool calls
  - Cancel during tool execution
- [ ] Verify no flickering or layout jumps
- [ ] Performance check with debug logging

---

## Testing Strategy

### Unit Tests

```go
// activity_test.go

func TestActivityPanel_Height(t *testing.T) {
    p := NewActivityPanel()
    
    // Empty panel has zero height
    assert.Equal(t, 0, p.Height())
    
    // Thinking adds height
    p.SetThinking(true)
    assert.Equal(t, 1, p.Height())
    
    // Tools add height
    p.AddTool("read", "/path/to/file.go")
    assert.Equal(t, 2, p.Height())
}

func TestActivityPanel_SpinnerAnimation(t *testing.T) {
    p := NewActivityPanel()
    p.SetThinking(true)
    
    frame1 := p.View()
    p.Update(SpinnerTickMsg{})
    frame2 := p.View()
    
    assert.NotEqual(t, frame1, frame2, "spinner should animate")
}

func TestToolSummary(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"read", `{"file_path": "/home/user/code/main.go"}`, "main.go"},
        {"grep", `{"pattern": "func.*", "include": "*.go"}`, `"func.*" in *.go`},
        {"bash", `{"command": "go test ./..."}`, "go test ./..."},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := toolSummary(tt.name, tt.input)
            assert.Contains(t, result, tt.expected)
        })
    }
}
```

### Integration Tests

- Test full message flow with activity panel updates
- Verify layout doesn't break at various terminal sizes
- Test rapid state changes (multiple tools in quick succession)

### Manual Testing Checklist

- [ ] Start app, verify activity panel hidden
- [ ] Send message, see "Thinking..." with animation
- [ ] See tool lines appear as tools run
- [ ] See clean response without tool dumps
- [ ] Verify activity clears after response
- [ ] Resize terminal during streaming
- [ ] Cancel mid-stream with Esc
- [ ] Verify model name in status bar

---

## Future Enhancements

### Metadata Modal (Phase 2)

After this foundation is in place, we can add an on-demand modal:

```
┌─────────────────────────────────────────────────┐
│  Message Details                            [x] │
├─────────────────────────────────────────────────┤
│  [Tools]  [Cost]  [Thinking]                    │
├─────────────────────────────────────────────────┤
│  Detailed view of selected tab...               │
└─────────────────────────────────────────────────┘
```

Triggered by keyboard shortcut (e.g., `i` for info) on last message.

### Cost Tracking

- Capture token counts from API response
- Calculate cost based on model pricing
- Display in status bar or modal

### Tool Result Inspection

- Click on tool line to see full input/output
- Or keyboard shortcut to expand tool details

### Session Statistics

- Total tokens used in session
- Total cost
- Tool usage breakdown

---

## Dependencies

### Existing Infrastructure

- `pubsub` / `events` - Already have tool events flowing
- `bridge.TUIBridge` - Already forwarding events to TUI
- `StreamToolCallMsg` - Already exists in chat

### New Dependencies

None - using existing Bubble Tea primitives.

---

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Spinner causing excessive redraws | Use reasonable tick interval (100ms) |
| Layout jumps when activity appears/disappears | Smooth transitions, test thoroughly |
| Long tool lists overflow | Cap at maxTools, show count for overflow |
| Complex tool inputs hard to summarize | Truncate with ellipsis, keep it simple |

---

## Success Criteria

1. ✅ Activity panel shows animated spinner during thinking
2. ✅ Tool operations appear in real-time with clear summaries
3. ✅ Message viewport contains only actual response content
4. ✅ Activity panel hides when idle (no wasted space)
5. ✅ Status bar shows active model name
6. ✅ No visual flickering or layout instability
7. ✅ All existing tests pass
8. ✅ New components have test coverage

---

## Estimated Effort

| Phase | Effort | Cumulative |
|-------|--------|------------|
| Phase 1: Activity Foundation | 2-3 hours | 2-3 hours |
| Phase 2: Tool Summaries | 1-2 hours | 3-5 hours |
| Phase 3: Chat Integration | 2-3 hours | 5-8 hours |
| Phase 4: Clean Messages | 1 hour | 6-9 hours |
| Phase 5: Status Bar | 1 hour | 7-10 hours |
| Phase 6: Testing & Polish | 1-2 hours | 8-12 hours |

**Total: 8-12 hours**
