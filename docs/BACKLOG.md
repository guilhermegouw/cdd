# CDD Backlog

Bugs and enhancements identified during code review sessions.

## Enhancements

### High Priority

| ID | Module | Description | Notes |
|----|--------|-------------|-------|
| E001 | `debug` | Add log levels (DEBUG, INFO, WARN, ERROR) | Filter noise during development |
| E002 | `debug` | Add source location (file:line) | Jump directly to code from logs |
| E003 | `tui` | Model switcher during session | Ctrl+M to change models on the fly |

### Medium Priority

| ID | Module | Description | Notes |
|----|--------|-------------|-------|
| E004 | `debug` | Add timing helpers | Measure operation duration |
| E005 | `debug` | Structured logging fields | `Log("msg", "key", value)` format |
| E006 | `config` | Extract `finalizeConfig()` | Reduce duplication in Load/LoadFromFile |
| E007 | `tui` | Show current model in status bar | User visibility |

### Low Priority

| ID | Module | Description | Notes |
|----|--------|-------------|-------|
| E008 | `config` | Split `configureProviders` | Reduce cyclomatic complexity |
| E009 | `debug` | Log rotation | Prevent disk fill |

## Bugs

| ID | Module | Description | Severity |
|----|--------|-------------|----------|
| - | - | None identified yet | - |

## Technical Debt

| ID | Module | Description | Notes |
|----|--------|-------------|-------|
| T001 | `providers.go` | Awkward `_ = cacheErr` pattern | Line 44, simplify error handling |

## Completed

| ID | Description | Date |
|----|-------------|------|
| - | Removed `var debugMode` global from cmd | 2025-12-22 |
| - | Fixed stderr consistency for debug messages | 2025-12-22 |
| - | Moved `defaultSystemPrompt` to agent package | 2025-12-22 |

---

## Review Progress

| Package | Reviewed | Documented | Refactored |
|---------|----------|------------|------------|
| `cmd` | Yes | - | Yes |
| `debug` | Yes | Yes | - |
| `config` | Yes | Yes | - |
| `tools` | No | No | No |
| `provider` | No | No | No |
| `agent` | No | No | No |
| `tui` | No | No | No |
