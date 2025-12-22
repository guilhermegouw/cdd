# TUI and Setup Wizard Documentation

This document provides comprehensive documentation for the Terminal User Interface (TUI) and setup wizard, built with Bubble Tea v2.

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
  - [Page-Based Navigation](#page-based-navigation)
  - [Component Structure](#component-structure)
  - [Message Flow](#message-flow)
- [Welcome Screen](#welcome-screen)
- [Setup Wizard](#setup-wizard)
  - [Wizard Steps](#wizard-steps)
  - [Provider Selection](#provider-selection)
  - [Authentication Method](#authentication-method)
  - [OAuth2 Flow](#oauth2-flow)
  - [API Key Input](#api-key-input)
  - [Model Selection](#model-selection)
  - [Completion](#completion)
- [Theme System](#theme-system)
  - [CDD Theme](#cdd-theme)
  - [Color Palette](#color-palette)
  - [Gradient Rendering](#gradient-rendering)
  - [Style Definitions](#style-definitions)
- [Components](#components)
  - [Logo](#logo)
  - [Text Input](#text-input)
  - [Spinner](#spinner)
- [Keyboard Navigation](#keyboard-navigation)
- [File Structure](#file-structure)
- [Dependencies](#dependencies)

---

## Overview

The TUI provides a polished terminal interface for CDD, featuring:

- **Welcome screen**: CDD-themed branding with ASCII logo
- **Setup wizard**: Multi-step configuration flow
- **OAuth2 integration**: Browser-based authentication for Claude Account
- **Theme system**: CDD-inspired green aesthetic with gradient support
- **Responsive layout**: Adapts to terminal dimensions

The TUI launches automatically on the `cdd` command and shows the setup wizard on first run or when no valid configuration exists.

---

## Architecture

### Page-Based Navigation

**Implementation**: `internal/tui/page/page.go`

The TUI uses a page-based navigation system:

```go
type ID string

const (
    Welcome ID = "welcome"  // Welcome/splash screen
    Wizard  ID = "wizard"   // Setup wizard
    Main    ID = "main"     // Main application (future)
    Chat    ID = "chat"     // Chat interface
)
```

Page transitions are handled via messages:

```go
type ChangeMsg struct {
    Page ID
}
```

### Component Structure

```
TUI Model
├── Welcome Screen
│   └── Logo Component
├── Wizard
│   ├── ProviderList
│   ├── AuthMethodChooser (Anthropic only)
│   ├── OAuth2Flow
│   ├── APIKeyInput
│   ├── ModelList (Large)
│   └── ModelList (Small)
└── Chat Page
    ├── Message List
    ├── Input Component
    └── Markdown Renderer
```

### Message Flow

The TUI follows Bubble Tea's Elm architecture:

```
┌─────────────────────────────────────────────────┐
│                    TUI Model                     │
├─────────────────────────────────────────────────┤
│                                                 │
│  Init() ─────► Initial Command                  │
│                                                 │
│  Update(msg) ──┬─► WindowSizeMsg → resize       │
│                ├─► KeyMsg → handle keys         │
│                ├─► StartWizardMsg → init wizard │
│                ├─► CompleteMsg → save config    │
│                └─► route to current page        │
│                                                 │
│  View() ─────► Render current page              │
│                                                 │
└─────────────────────────────────────────────────┘
```

---

## Welcome Screen

**Implementation**: `internal/tui/components/welcome/welcome.go`

The welcome screen displays CDD-themed branding:

```
███╗   ███╗ █████╗ ████████╗██████╗ ██╗██╗  ██╗
████╗ ████║██╔══██╗╚══██╔══╝██╔══██╗██║╚██╗██╔╝
██╔████╔██║███████║   ██║   ██████╔╝██║ ╚███╔╝
██║╚██╔╝██║██╔══██║   ██║   ██╔══██╗██║ ██╔██╗
██║ ╚═╝ ██║██║  ██║   ██║   ██║  ██║██║██╔╝ ██╗
╚═╝     ╚═╝╚═╝  ╚═╝   ╚═╝   ╚═╝  ╚═╝╚═╝╚═╝  ╚═╝

Wake up, Neo...

The CDD has you...

Follow the white rabbit.

Let's configure your AI assistant.

Press Enter to begin setup • q to quit
```

**Features**:
- Centered layout adapting to terminal size
- Gradient-colored ASCII logo
- CDD movie references
- Simple keyboard navigation

**Key bindings**:
| Key | Action |
|-----|--------|
| `Enter`, `Space` | Start setup wizard |
| `q`, `Ctrl+C` | Quit |

---

## Setup Wizard

**Implementation**: `internal/tui/components/wizard/wizard.go`

The wizard guides users through initial configuration with a multi-step flow.

### Wizard Steps

```go
const (
    StepProvider Step = iota    // Select LLM provider
    StepAuthMethod              // Choose OAuth or API Key (Anthropic)
    StepOAuth                   // OAuth2 flow
    StepAPIKey                  // API key input
    StepLargeModel              // Select large model
    StepSmallModel              // Select small model
    StepComplete                // Show summary
)
```

**Progress indicator**:
```
Provider → Auth → OAuth → Large Model → Small Model
    ↑
 current
```

### Provider Selection

**Implementation**: `internal/tui/components/wizard/provider.go`

Lists available providers from catwalk metadata:

```
Select a provider:

  ▸ Anthropic (Claude)
    OpenAI (GPT-4)
    Google (Gemini)
    ...

Use ↑/↓ to navigate, Enter to select
```

**Features**:
- Scrollable list for many providers
- Provider names from catwalk metadata
- Keyboard navigation with arrow keys

### Authentication Method

**Implementation**: `internal/tui/components/wizard/method.go`

For Anthropic, users can choose between OAuth and API Key:

```
How would you like to authenticate with Anthropic?

  ▸ Claude Account (OAuth)
    Recommended for personal use

    API Key
    For API access with separate billing
```

**AuthMethod enum**:
```go
const (
    AuthMethodOAuth2 AuthMethod = iota
    AuthMethodAPIKey
)
```

### OAuth2 Flow

**Implementation**: `internal/tui/components/wizard/oauth.go`

Two-stage OAuth flow:

**Stage 1 - URL Display**:
```
Press Enter to open the authorization URL in your browser:

https://claude.ai/oauth/authorize...
```

**Stage 2 - Code Entry**:
```
Enter the code you received:

> [code input with cursor]
```

**Validation states**:
| State | Display |
|-------|---------|
| None | `> ` prompt |
| Verifying | Spinner animation |
| Valid | ✓ checkmark |
| Error | ✗ error icon |

**Browser opening**: Uses platform-specific commands silently:
- Linux: `xdg-open`
- macOS: `open`
- Windows: `rundll32 url.dll,FileProtocolHandler`

### API Key Input

**Implementation**: `internal/tui/components/wizard/apikey.go`

Secure text input for API keys:

```
Enter your Anthropic API key:

> sk-ant-api03-...

Tip: Your key starts with 'sk-ant-'
```

**Features**:
- Password masking option
- Validation feedback
- Provider-specific hints

### Model Selection

**Implementation**: `internal/tui/components/wizard/model.go`

Select models for each tier:

```
Select your large model for complex tasks:

  ▸ Claude Sonnet 4 (claude-sonnet-4-20250514)
    Best balance of intelligence and speed

    Claude Opus 4 (claude-opus-4-0-20250514)
    Most capable, slower

    Claude 3.5 Sonnet (claude-3-5-sonnet-latest)
    Previous generation
```

**Features**:
- Model descriptions from catwalk
- Pre-selects provider's default model
- Separate selection for large and small tiers

### Completion

Shows a summary of the configuration:

```
✓ Setup Complete!

Provider: Anthropic
Authentication: OAuth (Claude Account)
Large Model: Claude Sonnet 4
Small Model: Claude 3.5 Haiku

Configuration saved to: ~/.config/cdd/cdd.json

Press any key to continue...
```

---

## Theme System

**Implementation**: `internal/tui/styles/`

### Default Theme

**Implementation**: `internal/tui/styles/default.go`

An ocean-inspired dark theme with calming blue tones.

### Color Palette

| Color | Hex | Usage |
|-------|-----|-------|
| Primary | `#5eb5f7` | Clear ocean blue, main accents |
| Secondary | `#7ec8e8` | Light sky blue, subtitles |
| Tertiary | `#2d4a5e` | Deep ocean, backgrounds |
| Accent | `#8fd4f4` | Bright water, highlights |
| BgBase | `#0f1419` | Deep sea dark background |
| BgSubtle | `#1a2028` | Slightly lighter background |
| BgOverlay | `#232a32` | Overlay/modal background |
| FgBase | `#c5d1de` | Soft white-blue text |
| FgMuted | `#7a8b99` | Muted blue-gray text |
| FgSubtle | `#4d5b66` | Subtle blue-gray text |
| Border | `#2d4a5e` | Default border color |
| BorderFocus | `#5eb5f7` | Focused border color |
| Success | `#7ec8e8` | Light blue (calm success) |
| Error | `#f4726d` | Coral red |
| Warning | `#f4c56d` | Sandy amber |
| Info | `#5eb5f7` | Ocean blue |

### Gradient Rendering

**Implementation**: `internal/tui/styles/theme.go:176-273`

The theme system supports horizontal gradients for text:

```go
func ForegroundGrad(input string, bold bool, color1, color2 color.Color) []string
func ApplyForegroundGrad(input string, color1, color2 color.Color) string
func ApplyBoldForegroundGrad(input string, color1, color2 color.Color) string
```

**How it works**:
1. Splits string into grapheme clusters (proper Unicode handling)
2. Generates color ramp between start and end colors using HCL blending
3. Applies each color to corresponding character
4. Joins styled characters

**Example** (logo gradient):
```go
styles.ApplyForegroundGrad(logo, t.Primary, t.Secondary)
// Renders logo from bright green (#00ff41) to dark green (#008f11)
```

### Style Definitions

**Pre-built styles** (`Styles` struct):

| Style | Properties |
|-------|------------|
| `Base` | Default foreground |
| `Title` | Accent color, bold |
| `Subtitle` | Secondary color, bold |
| `Text` | Base foreground |
| `Muted` | Muted foreground |
| `Subtle` | Subtle foreground |
| `Success` | Success color (green) |
| `Error` | Error color (red) |
| `Warning` | Warning color (yellow) |
| `Info` | Info color (cyan) |
| `TextInput` | Cursor, prompt, placeholder styles |

**Usage**:
```go
t := styles.CurrentTheme()
title := t.S().Title.Render("Welcome")
error := t.S().Error.Render("Something went wrong")
```

---

## Components

### Logo

**Implementation**: `internal/tui/components/logo/logo.go`

ASCII art wordmark with gradient coloring.

**Functions**:
| Function | Description |
|----------|-------------|
| `Render()` | Full logo with gradient |
| `RenderSmall()` | Compact logo for narrow terminals |
| `RenderWithTagline()` | Logo + "Wake up, Neo..." |
| `Width()` | Logo width in characters |
| `Height()` | Logo height in lines |

### Text Input

Uses Bubble Tea's `textinput` component with theme-aware styling:

```go
input := textinput.New()
input.SetStyles(t.S().TextInput)
```

**Cursor configuration**:
- Shape: Block
- Color: Primary (bright green)
- Blinking: Enabled

### Spinner

Used during OAuth validation:

```go
spinner.New(
    spinner.WithSpinner(spinner.Dot),
    spinner.WithStyle(t.S().Base.Foreground(t.Primary)),
)
```

---

## Keyboard Navigation

### Global Keys

| Key | Action |
|-----|--------|
| `Ctrl+C` | Quit (always) |
| `q` | Quit (when allowed) |

### Wizard Navigation

| Key | Action |
|-----|--------|
| `↑` / `k` | Previous item |
| `↓` / `j` | Next item |
| `Enter` | Select / Confirm |
| `Esc` | Go back one step |

### Text Input

| Key | Action |
|-----|--------|
| `←` / `→` | Move cursor |
| `Backspace` | Delete character |
| `Enter` | Submit |

---

## File Structure

```
internal/tui/
├── tui.go                     # Main TUI model and entry point
├── keys.go                    # Global key bindings
├── page/
│   ├── page.go               # Page IDs and navigation messages
│   └── chat/
│       ├── chat.go           # Chat page model
│       ├── input.go          # Chat input component
│       └── markdown.go       # Markdown renderer
├── util/
│   └── util.go               # Utility types (Model interface, InfoMsg)
├── styles/
│   ├── theme.go              # Theme struct, manager, gradient functions
│   ├── default.go            # Default ocean theme colors
│   └── icons.go              # Unicode icons (✓, ✗, etc.)
└── components/
    ├── logo/
    │   └── logo.go           # ASCII logo rendering
    ├── welcome/
    │   └── welcome.go        # Welcome screen
    └── wizard/
        ├── wizard.go         # Main wizard orchestration
        ├── keys.go           # Wizard-specific key constants
        ├── provider.go       # Provider selection list
        ├── method.go         # Auth method chooser
        ├── oauth.go          # OAuth2 flow component
        ├── apikey.go         # API key input component
        └── model.go          # Model selection list
```

---

## Dependencies

| Package | Purpose |
|---------|---------|
| `charm.land/bubbletea/v2` | TUI framework (Elm architecture) |
| `charm.land/bubbles/v2` | Pre-built components (textinput, spinner) |
| `charm.land/lipgloss/v2` | Terminal styling and layout |
| `github.com/lucasb-eyer/go-colorful` | Color manipulation for gradients |
| `github.com/rivo/uniseg` | Unicode grapheme cluster handling |

---

## Running the TUI

The TUI launches automatically when running `cdd`:

```bash
# Build and run
task build
./cdd

# Or run directly
task run
```

**Entry point** (`internal/tui/tui.go`):

```go
// AgentFactory is a function that creates an agent from the current config.
type AgentFactory func() (*agent.DefaultAgent, error)

func Run(providers []catwalk.Provider, isFirstRun bool, ag *agent.DefaultAgent, factory AgentFactory) error {
    styles.NewManager()
    model := New(providers, isFirstRun, ag, factory)
    p := tea.NewProgram(model)
    model.program = p
    if model.chatPage != nil {
        model.chatPage.SetProgram(p)
    }
    _, err := p.Run()
    return err
}
```

**Program options**:
- Alt screen mode (set in View)
- Mouse support (cell motion)
- Agent factory for post-wizard agent creation

---

## Related Documentation

- [Provider System](./provider-system.md) - Provider configuration and OAuth details
