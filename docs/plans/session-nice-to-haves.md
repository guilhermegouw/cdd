# Session Management - Nice to Have Features

## 1. Search/Filter Functionality

### Overview
Allow users to filter sessions by keyword, searching both titles and message content.

### User Flow
1. User opens `/sessions` modal
2. Presses `/` to enter search mode
3. Text input appears at top of modal
4. As user types, sessions filter in real-time
5. Press `Esc` to clear search and show all sessions
6. Press `Enter` to select from filtered results

### Implementation

**Files to modify:**
- `internal/tui/components/sessions/list.go` - Add search input state and rendering
- `internal/tui/components/sessions/modal.go` - Handle search mode transitions

**Changes to list.go:**
```go
// Add to SessionList struct
searchMode  bool
searchInput textinput.Model

// In Update(), handle "/" key (already stubbed)
case "/":
    l.searchMode = true
    return l, l.searchInput.Focus()

// In search mode, handle typing and Esc
if l.searchMode {
    switch keyMsg.String() {
    case "esc":
        l.searchMode = false
        l.searchText = ""
        l.Refresh() // Show all sessions
    case "enter":
        l.searchMode = false
        // Keep filtered results, allow selection
    default:
        l.searchInput, cmd = l.searchInput.Update(msg)
        l.Search(l.searchInput.Value()) // Filter sessions
    }
}
```

**Changes to View():**
```go
// If in search mode, render search input at top
if l.searchMode {
    searchView := l.searchInput.View()
    content = searchView + "\n" + content
}
```

### Already Done
- `SearchWithPreview()` method in store - searches by title
- Could extend to search message content with new SQL query

### Estimated Scope
~50-70 lines of code

---

## 2. Session Preview Panel

### Overview
Show detailed preview of selected session before opening, displayed alongside the session list.

### User Flow
1. User opens `/sessions` modal
2. List shows on left, preview panel on right
3. As user navigates with j/k, preview updates
4. Preview shows: full title, dates, message count, first/last messages

### Implementation

**Files to modify:**
- `internal/tui/components/sessions/preview.go` - New file for preview component
- `internal/tui/components/sessions/modal.go` - Split layout rendering
- `internal/session/store.go` - Add method to get session with messages

**New preview.go:**
```go
type Preview struct {
    session  *session.SessionWithPreview
    messages []message.Message // First few messages
    width    int
    height   int
}

func (p *Preview) SetSession(sess *session.SessionWithPreview) {
    p.session = sess
    // Load first 3-5 messages for preview
}

func (p *Preview) View() string {
    // Render:
    // - Title (full, not truncated)
    // - Created: Jan 10, 2026
    // - Updated: 5 mins ago
    // - Messages: 4
    // - Preview of conversation
}
```

**Layout in modal.go:**
```go
// Split modal into two columns
// Left: Session list (60% width)
// Right: Preview panel (40% width)

listView := m.sessionList.View()
previewView := m.preview.View()

content := lipgloss.JoinHorizontal(lipgloss.Top, listView, previewView)
```

### New Store Method
```sql
-- Get messages for preview (first N messages)
SELECT id, role, parts, created_at
FROM messages
WHERE session_id = ?
ORDER BY created_at ASC
LIMIT 5;
```

### Estimated Scope
~100-150 lines of code (new file + modifications)

---

## Priority Recommendation

1. **Search/Filter** - Higher value, simpler to implement
   - Users with many sessions will need this
   - Foundation already exists (SearchWithPreview)

2. **Preview Panel** - Nice polish, more complex
   - Requires layout changes
   - New data fetching
   - Can be added later

## Dependencies
- None - both features build on existing session infrastructure
