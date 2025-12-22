# Debug Module

The debug module provides development logging for CDD CLI. It offers a simple, thread-safe file-based logger that can be enabled at runtime.

## Overview

| Aspect | Details |
|--------|---------|
| Location | `internal/debug/debug.go` |
| Size | ~112 lines |
| Purpose | Development-time logging to file |
| Thread Safety | Yes (via `sync.Mutex`) |

## Architecture

```mermaid
graph TB
    subgraph "Package State"
        enabled[enabled bool]
        logFile[logFile *os.File]
        mu[mu sync.Mutex]
        logPath[logPath string]
    end

    subgraph "Public API"
        Enable[Enable]
        Disable[Disable]
        Log[Log]
        Event[Event]
        Error[Error]
        IsEnabled[IsEnabled]
        LogPath[LogPath]
    end

    Enable --> enabled
    Enable --> logFile
    Enable --> logPath
    Disable --> enabled
    Disable --> logFile
    Log --> logFile
    Event --> Log
    Error --> Log
    IsEnabled --> enabled
    LogPath --> logPath

    mu -.->|protects| enabled
    mu -.->|protects| logFile
    mu -.->|protects| logPath
```

## Data Flow

```mermaid
sequenceDiagram
    participant User
    participant CLI as cmd/root.go
    participant Debug as debug.Enable()
    participant FS as File System
    participant App as Application
    participant Log as debug.Log()

    User->>CLI: cdd --debug
    CLI->>Debug: Enable(path)
    Debug->>FS: MkdirAll (create directory)
    Debug->>FS: OpenFile (create/append)
    Debug->>FS: Write session header
    Debug-->>CLI: nil (success)

    loop During Execution
        App->>Log: Log(format, args...)
        Log->>FS: WriteString + Sync
    end

    CLI->>Debug: Disable() [via defer]
    Debug->>FS: Close file
```

## Lifecycle

```mermaid
stateDiagram-v2
    [*] --> Disabled: Package initialized

    Disabled --> Enabled: Enable(path) called
    Enabled --> Disabled: Disable() called
    Enabled --> Enabled: Log()/Event()/Error() writes to file

    Disabled --> Disabled: Log() calls ignored (no-op)

    Enabled --> [*]: Application exits
```

## Entry Point

The debug module is activated from `cmd/root.go`:

```go
func runTUI(cmd *cobra.Command, _ []string) error {
    debugMode, err := cmd.Flags().GetBool("debug")
    if err != nil {
        return fmt.Errorf("getting debug flag: %w", err)
    }
    if debugMode {
        logPath := filepath.Join(xdg.DataHome, "cdd", "debug.log")
        if debugErr := debug.Enable(logPath); debugErr != nil {
            fmt.Fprintf(os.Stderr, "Warning: Failed to enable debug logging: %v\n", debugErr)
        } else {
            defer debug.Disable()
            fmt.Fprintf(os.Stderr, "Debug: %s\n", logPath)
        }
    }
    // ... rest of function
}
```

## API Reference

### Enable(path string) error

Opens the log file and enables logging.

```mermaid
flowchart TD
    A[Enable called] --> B{Already enabled?}
    B -->|Yes| C[Return nil]
    B -->|No| D[Create directory]
    D --> E{Success?}
    E -->|No| F[Return error]
    E -->|Yes| G[Open file append mode]
    G --> H{Success?}
    H -->|No| I[Return error]
    H -->|Yes| J[Set package state]
    J --> K[Write session header]
    K --> L[Return nil]
```

### Disable()

Closes the log file and disables logging.

```go
func Disable() {
    mu.Lock()
    defer mu.Unlock()

    if !enabled {
        return
    }

    if logFile != nil {
        _ = logFile.Close()
        logFile = nil
    }
    enabled = false
}
```

### Log(format string, args ...any)

Writes a timestamped message to the log file.

```go
// Example usage
debug.Log("User clicked button: %s", buttonName)
debug.Log("Request completed in %dms", elapsed)
```

**Output format:**
```
[15:04:05.123] User clicked button: submit
[15:04:05.456] Request completed in 42ms
```

### Event(component, eventType, details string)

Convenience wrapper for logging TUI events.

```go
// Example usage
debug.Event("chat", "keypress", "ctrl+c")
debug.Event("wizard", "step", "provider selection")
```

**Output format:**
```
[15:04:05.123] [chat] keypress: ctrl+c
[15:04:05.456] [wizard] step: provider selection
```

### Error(component string, err error, context string)

Convenience wrapper for logging errors.

```go
// Example usage
debug.Error("agent", err, "failed to send message")
```

**Output format:**
```
[15:04:05.123] [agent] ERROR: failed to send message - connection timeout
```

### IsEnabled() bool

Returns whether debug logging is currently enabled.

### LogPath() string

Returns the path to the current log file.

## Thread Safety

All functions are protected by a mutex to ensure thread-safe access from multiple goroutines:

```mermaid
sequenceDiagram
    participant G1 as Goroutine 1
    participant M as Mutex
    participant G2 as Goroutine 2
    participant File as Log File

    G1->>M: Lock()
    activate M
    G2->>M: Lock() [blocks]
    G1->>File: Write "Message A"
    G1->>M: Unlock()
    deactivate M
    activate M
    Note over G2: Now acquired
    G2->>File: Write "Message B"
    G2->>M: Unlock()
    deactivate M
```

## File Location

Default log path: `~/.local/share/cdd/debug.log`

This follows the XDG Base Directory Specification using `xdg.DataHome`.

## Usage

Enable debug logging when running CDD:

```bash
cdd --debug
```

View logs in real-time:

```bash
tail -f ~/.local/share/cdd/debug.log
```

## Design Decisions

1. **Package-level state**: Appropriate for a singleton logger pattern
2. **Sync after each write**: Ensures logs are visible immediately for `tail -f`
3. **Append mode**: Preserves logs across sessions
4. **Ignored errors on write**: Debug logging should never crash the app
5. **Mutex protection**: Safe for concurrent goroutine access
