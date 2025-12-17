# Socrates CLI - Code Architecture

> A clean-room implementation inspired by Crush's proven patterns, built for independence and flexibility.

---

## Table of Contents

1. [Architecture Principles](#1-architecture-principles)
2. [High-Level Overview](#2-high-level-overview)
3. [Package Structure](#3-package-structure)
4. [Layer Responsibilities](#4-layer-responsibilities)
5. [Key Design Decisions](#5-key-design-decisions)
6. [Component Details](#6-component-details)
7. [Data Flow](#7-data-flow)
8. [Extension Points](#8-extension-points)

---

## 1. Architecture Principles

### Inspired by Crush (Patterns to Adopt)

We implement these patterns ourselves, using Crush as reference:

- **Elm Architecture (TEA)** - Model → Update → View for TUI
- **Pub/Sub Events** - Decoupled service communication
- **Service Layer** - Each domain has its own service with broker
- **Tool Interface** - Common interface for all agent tools

### New for Socrates (Innovations)

- **Phase-Based Agent** - Agent behavior changes based on current phase
- **Dual Agent Pattern** - PhaseAgent for workflow, AgentInstance for debates
- **Slash Command Router** - First-class support for `/commands`
- **Model Tier Abstraction** - Small/Mid/Big instead of specific models

### Guiding Principles

```
1. INSPIRED-BY, NOT DEPENDENT-ON
   - Use Crush patterns as reference
   - Implement our own versions from scratch
   - Full ownership, full flexibility
   - Free to evolve in our own direction

2. SIMPLIFY WHERE POSSIBLE
   - Only implement what we need
   - Crush has features we don't need (LSP, MCP, etc.)
   - Start minimal, add complexity when required

3. INTERFACE-FIRST DESIGN
   - Define interfaces before implementations
   - Easy to swap implementations later
   - Testable by design

4. CONFIGURATION OVER CODE
   - Phases defined in config, not hardcoded
   - Model tiers in config
   - Tool permissions in config
```

---

## 2. High-Level Overview

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                              SOCRATES CLI                                    │
├──────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────────┐ │
│  │                         PRESENTATION LAYER                              │ │
│  │                                                                         │ │
│  │  ┌──────────────┐    ┌──────────────────────────────────────────────┐   │ │
│  │  │   CLI        │    │                    TUI                       │   │ │
│  │  │  (Cobra)     │───▶│  ┌──────────┐  ┌──────────┐  ┌────────────┐  │   │ │
│  │  │              │    │  │ ChatPage │  │ Debate   │  │  Dialogs   │  │   │ │
│  │  │ • run        │    │  │          │  │ Page     │  │            │  │   │ │
│  │  │ • login      │    │  └────┬─────┘  └────┬─────┘  │ • Permis.  │  │   │ │
│  │  │ • version    │    │       └──────┬──────┘        │ • Models   │  │   │ │
│  │  └──────────────┘    │              │               │ • Sessions │  │   │ │
│  │                      │              ▼               └────────────┘  │   │ │
│  │                      │       ┌────────────┐                         │   │ │
│  │                      │       │  AppModel  │ ◀── Bubble Tea          │   │ │
│  │                      │       └─────┬──────┘                         │   │ │
│  │                      └─────────────┼────────────────────────────────┘   │ │
│  └────────────────────────────────────┼────────────────────────────────────┘ │
│                                       │                                      │
│  ┌────────────────────────────────────┼────────────────────────────────────┐ │
│  │                         APPLICATION LAYER                               │ │
│  │                                    │                                    │ │
│  │  ┌─────────────────────────────────┼─────────────────────────────────┐  │ │
│  │  │                              App                                  │  │ │
│  │  │                                                                   │  │ │
│  │  │  ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌────────────┐   │  │ │
│  │  │  │   Agent    │  │   Slash    │  │   Phase    │  │   Debate   │   │  │ │
│  │  │  │            │  │   Router   │  │   System   │  │   Room     │   │  │ │
│  │  │  │ • Loop     │  │            │  │            │  │            │   │  │ │
│  │  │  │ • Tools    │  │ • /socrates│  │ • Socrates │  │ • Factory  │   │  │ │
│  │  │  │ • Stream   │  │ • /plan    │  │ • Planner  │  │ • Strategy │   │  │ │
│  │  │  │            │  │ • /exec    │  │ • Executor │  │ • Synth.   │   │  │ │
│  │  │  └────────────┘  │ • /debate  │  │ • Chat     │  └────────────┘   │  │ │
│  │  │                  │ • /init    │  └────────────┘                   │  │ │
│  │  │                  └────────────┘                                   │  │ │
│  │  └───────────────────────────────────────────────────────────────────┘  │ │
│  └─────────────────────────────────────────────────────────────────────────┘ │
│                                       │                                      │
│  ┌────────────────────────────────────┼────────────────────────────────────┐ │
│  │                          DOMAIN LAYER                                   │ │
│  │                                    │                                    │ │
│  │  ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌────────────┐         │ │
│  │  │  Session   │  │  Message   │  │ Permission │  │   History  │         │ │
│  │  │  Service   │  │  Service   │  │  Service   │  │  Service   │         │ │
│  │  │            │  │            │  │            │  │            │         │ │
│  │  │ + Broker   │  │ + Broker   │  │ + Broker   │  │ + Broker   │         │ │
│  │  └────────────┘  └────────────┘  └────────────┘  └────────────┘         │ │
│  │                                                                         │ │
│  └─────────────────────────────────────────────────────────────────────────┘ │
│                                       │                                      │
│  ┌────────────────────────────────────┼────────────────────────────────────┐ │
│  │                       INFRASTRUCTURE LAYER                              │ │
│  │                                                                         │ │
│  │  ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌────────────┐         │ │
│  │  │  Provider  │  │   Tools    │  │  Database  │  │   PubSub   │         │ │
│  │  │            │  │            │  │            │  │            │         │ │
│  │  │ • Anthropic│  │ • read     │  │ • SQLite   │  │ • Broker   │         │ │
│  │  │ • OpenAI   │  │ • write    │  │ • Queries  │  │ • Events   │         │ │
│  │  │ • Tiers    │  │ • edit     │  │ • Migrate  │  │            │         │ │
│  │  │            │  │ • bash     │  │            │  │            │         │ │
│  │  │            │  │ • glob/grep│  │            │  │            │         │ │
│  │  └────────────┘  └────────────┘  └────────────┘  └────────────┘         │ │
│  │                                                                         │ │
│  └─────────────────────────────────────────────────────────────────────────┘ │
│                                                                              │
└──────────────────────────────────────────────────────────────────────────────┘
```

### Layer Summary

| Layer | Responsibility | Key Packages |
|-------|----------------|--------------|
| **Presentation** | User interaction (CLI args, TUI) | `cmd/`, `tui/` |
| **Application** | Orchestration, workflows, commands | `app/`, `agent/`, `phase/`, `slash/`, `debate/` |
| **Domain** | Business logic, services | `session/`, `message/`, `permission/`, `history/` |
| **Infrastructure** | External systems, persistence | `provider/`, `tools/`, `database/`, `pubsub/` |

---

## 3. Package Structure

```
socrates-cli/
├── main.go                          # Entry point
├── go.mod                           # Dependencies
├── go.sum
│
├── cmd/                             # CLI commands (Cobra)
│   ├── root.go                      # Root command, global flags
│   ├── run.go                       # Start interactive session
│   ├── login.go                     # OAuth login
│   └── version.go                   # Version info
│
├── internal/
│   │
│   │━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
│   │ INFRASTRUCTURE LAYER
│   │━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
│   │
│   ├── pubsub/                      # Event system
│   │   ├── broker.go                # Generic Broker[T]
│   │   └── events.go                # Event types (Created, Updated, Deleted)
│   │
│   ├── database/                    # Persistence layer
│   │   ├── db.go                    # Connection management
│   │   ├── migrations/              # SQL migrations (Goose)
│   │   │   └── 001_initial.sql
│   │   └── queries/                 # SQL queries (SQLC)
│   │       ├── sessions.sql
│   │       ├── messages.sql
│   │       └── files.sql
│   │
│   ├── provider/                    # LLM providers
│   │   ├── provider.go              # Provider interface
│   │   ├── registry.go              # Provider registry
│   │   ├── anthropic.go             # Anthropic implementation
│   │   ├── openai.go                # OpenAI implementation
│   │   └── tier.go                  # Model tier abstraction
│   │
│   ├── tools/                       # Agent tools
│   │   ├── tool.go                  # Tool interface
│   │   ├── registry.go              # Tool registry
│   │   ├── executor.go              # Tool execution with permissions
│   │   ├── read.go                  # Read file contents
│   │   ├── write.go                 # Write/create files
│   │   ├── edit.go                  # Edit file sections
│   │   ├── glob.go                  # Find files by pattern
│   │   ├── grep.go                  # Search file contents
│   │   ├── ls.go                    # List directory
│   │   ├── bash.go                  # Shell execution
│   │   ├── job.go                   # Background job management
│   │   └── git.go                   # Git operations
│   │
│   │━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
│   │ DOMAIN LAYER
│   │━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
│   │
│   ├── session/                     # Session management
│   │   ├── service.go               # Session service + broker
│   │   ├── session.go               # Session model
│   │   └── events.go                # Session events
│   │
│   ├── message/                     # Message management
│   │   ├── service.go               # Message service + broker
│   │   ├── message.go               # Message model
│   │   ├── content.go               # Content parts (text, tool_call, etc.)
│   │   └── events.go                # Message events
│   │
│   ├── permission/                  # Permission system
│   │   ├── service.go               # Permission service + broker
│   │   ├── permission.go            # Permission model
│   │   ├── mode.go                  # Approval modes (paranoid/balanced/yolo)
│   │   └── events.go                # Permission events
│   │
│   ├── history/                     # File history tracking
│   │   ├── service.go               # History service + broker
│   │   └── file.go                  # File snapshot model
│   │
│   │━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
│   │ APPLICATION LAYER
│   │━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
│   │
│   ├── app/                         # Application orchestration
│   │   ├── app.go                   # Main App struct, wiring
│   │   ├── events.go                # Event forwarding to TUI
│   │   └── lifecycle.go             # Startup, shutdown
│   │
│   ├── agent/                       # Agent system
│   │   ├── agent.go                 # Agent interface
│   │   ├── loop.go                  # ReAct loop implementation
│   │   ├── stream.go                # Response streaming
│   │   └── phase_agent.go           # Phase-aware agent
│   │
│   ├── phase/                       # CDD Phase System
│   │   ├── phase.go                 # Phase interface
│   │   ├── registry.go              # Phase registry
│   │   ├── chat.go                  # Default chat phase
│   │   ├── socrates.go              # Requirements gathering
│   │   ├── planner.go               # Implementation planning
│   │   └── executor.go              # Code execution
│   │
│   ├── slash/                       # Slash Command Router
│   │   ├── router.go                # Command router
│   │   ├── command.go               # Command interface
│   │   ├── parser.go                # Parse "/cmd args" syntax
│   │   └── commands/                # Built-in commands
│   │       ├── help.go              # /help
│   │       ├── socrates.go          # /socrates
│   │       ├── plan.go              # /plan
│   │       ├── exec.go              # /exec
│   │       ├── debate.go            # /debate
│   │       ├── init.go              # /init
│   │       ├── new.go               # /new
│   │       ├── commit.go            # /commit
│   │       └── clear.go             # /clear
│   │
│   ├── debate/                      # Debate Room
│   │   ├── room.go                  # DebateRoom orchestrator
│   │   ├── instance.go              # AgentInstance (spawnable)
│   │   ├── factory.go               # Creates agent instances
│   │   ├── transcript.go            # Debate message history
│   │   ├── synthesis.go             # Generate recommendation
│   │   └── strategy/                # Debate strategies
│   │       ├── strategy.go          # Strategy interface
│   │       ├── parallel.go          # All respond at once
│   │       ├── roundrobin.go        # Take turns
│   │       └── adversarial.go       # Pro/con pairs
│   │
│   ├── config/                      # Configuration
│   │   ├── config.go                # Config struct
│   │   ├── load.go                  # Load from file/env
│   │   ├── defaults.go              # Default values
│   │   ├── tiers.go                 # Tier configuration
│   │   └── phases.go                # Phase configuration
│   │
│   ├── template/                    # Document Templates
│   │   ├── template.go              # Template interface
│   │   ├── loader.go                # Load templates
│   │   └── builtin.go               # Built-in templates
│   │
│   │━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
│   │ PRESENTATION LAYER
│   │━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
│   │
│   └── tui/                         # Bubble Tea TUI
│       ├── tui.go                   # Root model (tea.Model)
│       ├── keymap.go                # Key bindings
│       ├── styles.go                # Lipgloss styles
│       │
│       ├── page/                    # Pages
│       │   ├── page.go              # Page interface
│       │   ├── chat/                # Chat page
│       │   │   ├── chat.go          # Chat model
│       │   │   ├── messages.go      # Message list component
│       │   │   ├── input.go         # Input editor
│       │   │   └── sidebar.go       # Session sidebar
│       │   │
│       │   └── debate/              # Debate page
│       │       ├── debate.go        # Debate model
│       │       ├── panel.go         # Agent panel
│       │       └── transcript.go    # Debate transcript
│       │
│       ├── dialog/                  # Modal dialogs
│       │   ├── dialog.go            # Dialog interface
│       │   ├── permission.go        # Permission approval
│       │   ├── model.go             # Model/provider picker
│       │   ├── session.go           # Session picker
│       │   └── quit.go              # Quit confirmation
│       │
│       └── component/               # Reusable components
│           ├── spinner.go           # Loading spinner
│           ├── list.go              # Filterable list
│           └── status.go            # Status bar
│
├── templates/                       # Document templates (files)
│   ├── cdd.md.tmpl
│   ├── feature_spec.md.tmpl
│   ├── bug_plan.md.tmpl
│   └── enhancement.md.tmpl
│
└── scripts/
    ├── build.sh
    └── release.sh
```

### Package Dependency Rules

```
┌─────────────────────────────────────────────────────────────────┐
│                    DEPENDENCY DIRECTION                         │
│                                                                 │
│  Presentation ──▶ Application ──▶ Domain ──▶ Infrastructure     │
│                                                                 │
│  ✅ tui/ can import app/, phase/, slash/                        │
│  ✅ app/ can import session/, message/, provider/               │
│  ✅ session/ can import database/, pubsub/                      │
│                                                                 │
│  ❌ pubsub/ cannot import session/                              │
│  ❌ database/ cannot import app/                                │
│  ❌ provider/ cannot import tui/                                │
└─────────────────────────────────────────────────────────────────┘
```

---

## 4. Layer Responsibilities

### Infrastructure Layer

The foundation that everything else builds on. No business logic here.

#### PubSub (internal/pubsub/)

```go
// Generic event broker for decoupled communication
type Broker[T any] struct {
    subs map[chan Event[T]]struct{}
    mu   sync.RWMutex
}

type Event[T any] struct {
    Type    EventType  // Created, Updated, Deleted
    Payload T
}

func (b *Broker[T]) Subscribe(ctx context.Context) <-chan Event[T]
func (b *Broker[T]) Publish(eventType EventType, payload T)
```

**Responsibility:** Enable services to communicate without direct coupling.

#### Database (internal/database/)

```go
// SQLite connection with SQLC-generated queries
type DB struct {
    conn    *sql.DB
    queries *Queries  // SQLC generated
}

func (db *DB) Migrate() error  // Run Goose migrations
```

**Responsibility:** Persist sessions, messages, file history.

#### Provider (internal/provider/)

```go
// Provider abstracts LLM communication
type Provider interface {
    CreateMessage(ctx context.Context, req Request) (*Response, error)
    StreamMessage(ctx context.Context, req Request) (<-chan Event, error)
    Name() string
    Model() string
}

// Tier simplifies model selection
type Tier string
const (
    TierSmall Tier = "small"  // Fast, cheap
    TierMid   Tier = "mid"    // Balanced
    TierBig   Tier = "big"    // Maximum capability
)
```

**Responsibility:** Abstract LLM providers, handle streaming, manage model tiers.

#### Tools (internal/tools/)

```go
// Tool is an action the agent can take
type Tool interface {
    Name() string
    Description() string
    Schema() map[string]any
    Execute(ctx context.Context, params map[string]any) (Result, error)
}

// Registry holds all available tools
type Registry struct {
    tools map[string]Tool
}
```

**Responsibility:** Provide file, shell, and git operations to the agent.

---

### Domain Layer

Business entities and their services. Each service owns its data and publishes events.

#### Session Service (internal/session/)

```go
type Session struct {
    ID        string
    Title     string
    CreatedAt time.Time
    UpdatedAt time.Time
}

type Service struct {
    db     *database.DB
    broker *pubsub.Broker[Session]
}

func (s *Service) Create(ctx context.Context, title string) (Session, error)
func (s *Service) Get(ctx context.Context, id string) (Session, error)
func (s *Service) List(ctx context.Context) ([]Session, error)
func (s *Service) Delete(ctx context.Context, id string) error
```

**Responsibility:** Manage conversation sessions, publish session events.

#### Message Service (internal/message/)

```go
type Message struct {
    ID        string
    SessionID string
    Role      string      // user, assistant, system
    Content   []Part      // text, tool_call, tool_result
    CreatedAt time.Time
}

type Service struct {
    db     *database.DB
    broker *pubsub.Broker[Message]
}
```

**Responsibility:** Store conversation history, publish message events.

#### Permission Service (internal/permission/)

```go
type Permission struct {
    ID       string
    ToolName string
    Params   map[string]any
    Status   Status  // Pending, Approved, Denied
}

type Mode string
const (
    ModeParanoid Mode = "paranoid"  // Ask for everything
    ModeBalanced Mode = "balanced"  // Auto-approve safe tools
    ModeYOLO     Mode = "yolo"      // Skip all approvals
)

type Service struct {
    mode   Mode
    broker *pubsub.Broker[Permission]
}

// Request blocks until user responds (via channel)
func (s *Service) Request(ctx context.Context, req Permission) (bool, error)
```

**Responsibility:** Gate tool execution, handle approval modes.

---

### Application Layer

Orchestration, workflows, and coordination logic.

#### App (internal/app/)

```go
type App struct {
    // Infrastructure
    db       *database.DB
    provider provider.Provider

    // Services
    sessions    *session.Service
    messages    *message.Service
    permissions *permission.Service

    // Application components
    agent       *agent.PhaseAgent
    slashRouter *slash.Router
    phases      *phase.Registry

    // TUI
    program *tea.Program
}

func (a *App) Run(ctx context.Context) error {
    // 1. Initialize services
    // 2. Setup event forwarding
    // 3. Start TUI
}
```

**Responsibility:** Wire everything together, manage lifecycle.

#### Phase System (internal/phase/)

```go
type Phase interface {
    Name() string
    SystemPrompt() string
    AllowedTools() []string
    OnEnter(ctx context.Context) error
    OnExit(ctx context.Context) error
}

type PhaseAgent struct {
    agent    *agent.Agent
    current  Phase
    registry *Registry
}

func (a *PhaseAgent) SwitchPhase(name string) error {
    if a.current != nil {
        a.current.OnExit(ctx)
    }

    phase := a.registry.Get(name)
    a.current = phase
    a.current.OnEnter(ctx)

    // Update agent behavior
    a.agent.SetSystemPrompt(phase.SystemPrompt())
    a.agent.SetToolFilter(phase.AllowedTools())

    return nil
}
```

**Responsibility:** Define CDD workflow phases, switch agent behavior.

#### Slash Router (internal/slash/)

```go
type Command interface {
    Name() string
    Description() string
    Execute(ctx context.Context, args string) error
}

type Router struct {
    commands map[string]Command
}

func (r *Router) Handle(input string) (handled bool, err error) {
    if !strings.HasPrefix(input, "/") {
        return false, nil
    }

    name, args := parseCommand(input)
    cmd, ok := r.commands[name]
    if !ok {
        return true, fmt.Errorf("unknown command: /%s", name)
    }

    return true, cmd.Execute(ctx, args)
}
```

**Responsibility:** Parse and dispatch `/commands`.

#### Debate Room (internal/debate/)

```go
type AgentInstance struct {
    ID       string
    Provider provider.Provider
    Persona  string
    History  []message.Message
}

type Room struct {
    Topic      string
    Agents     []*AgentInstance
    Moderator  *AgentInstance
    Strategy   Strategy
    Transcript []DebateMessage
}

type Strategy interface {
    Run(ctx context.Context, room *Room) (*Outcome, error)
}
```

**Responsibility:** Orchestrate multi-agent discussions.

---

### Presentation Layer

User interface, no business logic.

#### TUI (internal/tui/)

```go
type Model struct {
    app         *app.App
    currentPage page.Page
    dialog      dialog.Dialog
    width       int
    height      int
}

func (m Model) Init() tea.Cmd
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd)
func (m Model) View() string
```

**Responsibility:** Render UI, capture user input, forward to app.

---

## 5. Key Design Decisions

### Decision 1: Clean-Room Implementation

**Approach:** Build our own implementation inspired by Crush's patterns.

```
Crush (Reference)              Socrates (Our Implementation)
─────────────────              ─────────────────────────────
pubsub/broker.go      ──▶      pubsub/broker.go (similar pattern, our code)
session/service.go    ──▶      session/service.go (our version)
tui/page/chat/        ──▶      tui/page/chat/ (our version)
```

**Why:**
- Full ownership and flexibility
- Free to evolve in our own direction
- Can simplify where Crush is over-engineered for our needs
- No dependency coupling or licensing concerns

---

### Decision 2: Interface-First Design

**Approach:** Define interfaces before implementations.

```go
// Define the contract first
type Provider interface {
    CreateMessage(ctx context.Context, req Request) (*Response, error)
    StreamMessage(ctx context.Context, req Request) (<-chan Event, error)
}

// Then implement
type AnthropicProvider struct { ... }
type OpenAIProvider struct { ... }

// Easy to add more later
type OllamaProvider struct { ... }  // Local models
type BedrockProvider struct { ... } // AWS
```

**Why:**
- Easy to swap implementations
- Testable (mock interfaces)
- Clear contracts between layers

---

### Decision 3: Slash Commands Before Agent

**Approach:** Route through slash router before sending to agent.

```
User Input
    │
    ▼
┌─────────────────┐
│  Slash Router   │──── /socrates ────▶ Switch to Socrates phase
│                 │──── /debate ──────▶ Start debate room
│                 │──── /help ────────▶ Show help
└────────┬────────┘
         │ (not a command)
         ▼
┌─────────────────┐
│  Phase Agent    │──── Apply phase system prompt
│                 │──── Filter tools by phase
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Agent Loop     │──── ReAct loop
│                 │──── Tool execution
└─────────────────┘
```

**Why:** Commands are first-class, not hacks. Clean separation of concerns.

---

### Decision 4: Phase-Aware Agent

**Approach:** Agent behavior changes based on current phase.

```go
type PhaseAgent struct {
    agent    *Agent        // Core agent loop
    current  Phase         // Current phase
    registry *Registry     // Available phases
}

// Phase defines behavior
type Phase interface {
    Name() string
    SystemPrompt() string   // Different prompt per phase
    AllowedTools() []string // Tool filtering
}

// Switching phases changes agent behavior
func (a *PhaseAgent) SwitchPhase(name string) error {
    phase := a.registry.Get(name)

    a.agent.SetSystemPrompt(phase.SystemPrompt())
    a.agent.SetToolFilter(phase.AllowedTools())

    a.current = phase
    return nil
}
```

**Why:** CDD workflow requires different behaviors at each stage.

---

### Decision 5: Isolated Debate Room

**Approach:** Debate room spawns independent agents, separate from main chat.

```go
// Main chat uses PhaseAgent (single agent, phase-switching)
// Debate uses AgentInstances (multiple agents, independent)

type Room struct {
    agents    []*AgentInstance  // Multiple independent agents
    moderator *AgentInstance    // Synthesizes results
    // NOT connected to the main PhaseAgent
}

func (r *Room) Start(topic string) {
    // Spawn fresh agents for this debate
    for i := 0; i < agentCount; i++ {
        r.agents = append(r.agents, r.factory.NewInstance())
    }

    // Run debate (parallel exploration)
    outcome := r.strategy.Run(ctx, r)

    // Return result to main session
    return outcome
}
```

**Why:** Debates are parallel exploration, not sequential conversation.

---

### Decision 6: Model Tiers in Config

**Approach:** Abstract model selection to tiers.

```json
{
  "tiers": {
    "small": {
      "anthropic": "claude-3-5-haiku-20241022",
      "openai": "gpt-4o-mini"
    },
    "mid": {
      "anthropic": "claude-sonnet-4-20250514",
      "openai": "gpt-4o"
    },
    "big": {
      "anthropic": "claude-opus-4-20250514",
      "openai": "o1"
    }
  },
  "phases": {
    "socrates": { "tier": "mid" },
    "planner": { "tier": "big" },
    "executor": { "tier": "mid" }
  },
  "debate": {
    "default_tier": "mid",
    "moderator_tier": "big"
  }
}
```

**Why:**
- Users think in capabilities, not model names
- Easy to update when new models release
- Cost control per workflow stage

---

### Decision 7: Event-Driven Services

**Approach:** Services publish events, don't call each other.

```go
// BAD: Direct coupling
func (s *SessionService) Create() {
    session := ...
    s.messageService.NotifyNewSession(session)  // Direct call
}

// GOOD: Event-driven
func (s *SessionService) Create() {
    session := ...
    s.broker.Publish(Created, session)  // Publish event
}

// TUI subscribes to events
func (m *Model) Init() tea.Cmd {
    return tea.Batch(
        m.subscribeToSessions(),
        m.subscribeToMessages(),
        m.subscribeToPermissions(),
    )
}
```

**Why:**
- Services are decoupled
- Easy to add new subscribers
- TUI stays in sync automatically

---

## 6. Component Details

### 6.1 Phase Interface

```go
package phase

type Phase interface {
    // Identity
    Name() string
    Description() string

    // Behavior
    SystemPrompt() string
    AllowedTools() []string

    // Lifecycle
    OnEnter(ctx context.Context) error
    OnExit(ctx context.Context) error

    // Optional: Custom handling
    PreProcess(input string) (string, error)
    PostProcess(output string) (string, error)
}

// BasePhase provides default implementations
type BasePhase struct {
    name        string
    description string
    prompt      string
    tools       []string
}

func (p *BasePhase) OnEnter(ctx context.Context) error { return nil }
func (p *BasePhase) OnExit(ctx context.Context) error  { return nil }
func (p *BasePhase) PreProcess(input string) (string, error) { return input, nil }
func (p *BasePhase) PostProcess(output string) (string, error) { return output, nil }
```

### 6.2 Socrates Phase

```go
package phase

type SocratesPhase struct {
    BasePhase
    questionCount int
    maxQuestions  int
}

func NewSocratesPhase() *SocratesPhase {
    return &SocratesPhase{
        BasePhase: BasePhase{
            name:        "socrates",
            description: "Requirements gathering through questions",
            prompt:      socratesSystemPrompt,
            tools:       []string{"view", "glob", "grep", "ls"},
        },
        maxQuestions: 10,
    }
}

const socratesSystemPrompt = `You are Socrates, a requirements gathering expert.

Your goal is to deeply understand what the user wants to build through
thoughtful questions. You should:

1. Ask clarifying questions about the feature/bug/enhancement
2. Explore edge cases and constraints
3. Identify dependencies and potential blockers
4. Summarize requirements when you have enough information

You have READ-ONLY access to the codebase. Use tools to understand existing
code but do not make changes.

When you have gathered sufficient requirements, tell the user they can
proceed to /plan to create an implementation plan.`
```

### 6.3 Slash Command

```go
package slash

// SocratesCommand switches to Socrates phase
type SocratesCommand struct {
    agent *agent.PhaseAgent
}

func (c *SocratesCommand) Name() string { return "socrates" }
func (c *SocratesCommand) Description() string {
    return "Start requirements gathering with Socrates"
}

func (c *SocratesCommand) Execute(ctx context.Context, args string) error {
    if err := c.agent.SwitchPhase("socrates"); err != nil {
        return err
    }

    // If args provided, immediately send as first message
    if args != "" {
        return c.agent.HandleMessage(args)
    }

    return nil
}
```

### 6.4 Debate Room

```go
package debate

type Room struct {
    ID         string
    Topic      string
    Agents     []*AgentInstance
    Moderator  *AgentInstance
    Strategy   Strategy
    Transcript []Message
    Config     RoomConfig

    // Dependencies
    factory    *Factory
    broker     *pubsub.Broker[Event]
}

type RoomConfig struct {
    MaxRounds    int
    AgentCount   int
    DefaultTier  tier.Tier
    Personas     []string
}

func (r *Room) Start(ctx context.Context, topic string) (*Outcome, error) {
    r.Topic = topic

    // Spawn agents
    for i := 0; i < r.Config.AgentCount; i++ {
        persona := r.Config.Personas[i % len(r.Config.Personas)]
        agent := r.factory.NewInstance(r.Config.DefaultTier, persona)
        r.Agents = append(r.Agents, agent)
    }

    // Spawn moderator (uses bigger model)
    r.Moderator = r.factory.NewInstance(tier.Big, "Moderator")

    // Run strategy
    outcome, err := r.Strategy.Run(ctx, r)
    if err != nil {
        return nil, err
    }

    // Publish completion event
    r.broker.Publish(pubsub.Created, DebateCompletedEvent{
        RoomID:  r.ID,
        Outcome: outcome,
    })

    return outcome, nil
}
```

### 6.5 Parallel Strategy

```go
package strategy

type ParallelStrategy struct{}

func (s *ParallelStrategy) Run(ctx context.Context, room *debate.Room) (*debate.Outcome, error) {
    // Round 1: Proposals (parallel)
    proposals := s.collectProposals(ctx, room)
    room.AddTranscriptRound("proposals", proposals)

    // Round 2: Critiques (parallel)
    critiques := s.collectCritiques(ctx, room, proposals)
    room.AddTranscriptRound("critiques", critiques)

    // Round 3: Rebuttals (parallel)
    rebuttals := s.collectRebuttals(ctx, room, critiques)
    room.AddTranscriptRound("rebuttals", rebuttals)

    // Synthesis by moderator
    synthesis := s.synthesize(ctx, room)

    return &debate.Outcome{
        Recommendation: synthesis,
        Transcript:     room.Transcript,
        Consensus:      s.calculateConsensus(rebuttals),
    }, nil
}

func (s *ParallelStrategy) collectProposals(ctx context.Context, room *debate.Room) []debate.Message {
    var wg sync.WaitGroup
    results := make([]debate.Message, len(room.Agents))

    for i, agent := range room.Agents {
        wg.Add(1)
        go func(i int, agent *debate.AgentInstance) {
            defer wg.Done()
            response, _ := agent.Respond(ctx, fmt.Sprintf(
                "Topic: %s\n\nProvide your proposal for how to approach this.",
                room.Topic,
            ))
            results[i] = debate.Message{
                AgentID: agent.ID,
                Round:   1,
                Phase:   "proposal",
                Content: response,
            }
        }(i, agent)
    }

    wg.Wait()
    return results
}
```

---

## 7. Data Flow

### 7.1 Normal Chat Flow

```
User Input ("/socrates implement auth")
    │
    ▼
┌─────────────────────────────────────────────────────────────┐
│ TUI Editor                                                  │
│   • Capture input                                           │
│   • Send to App                                             │
└────────────────────────────┬────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────┐
│ Slash Router                                                │
│   • Parse: name="socrates", args="implement auth"           │
│   • Lookup command                                          │
│   • Execute SocratesCommand                                 │
└────────────────────────────┬────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────┐
│ Phase Agent                                                 │
│   • SwitchPhase("socrates")                                 │
│   • Update system prompt                                    │
│   • Filter tools to read-only                               │
│   • Send "implement auth" to coordinator                    │
└────────────────────────────┬────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────┐
│ Agent Loop                                                  │
│   • Build messages with Socrates system prompt              │
│   • Call provider with filtered tools                       │
│   • Stream response                                         │
│   • Execute any tool calls                                  │
└────────────────────────────┬────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────┐
│ Message Service                                             │
│   • Save assistant message                                  │
│   • Publish MessageCreated event                            │
└────────────────────────────┬────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────┐
│ TUI                                                         │
│   • Receive event via pub/sub                               │
│   • Update messages view                                    │
│   • Re-render                                               │
└─────────────────────────────────────────────────────────────┘
```

### 7.2 Debate Flow

```
User Input ("/debate How should we implement auth?")
    │
    ▼
┌─────────────────────────────────────────────────────────────┐
│ Slash Router                                                │
│   • Parse: name="debate", args="How should..."              │
│   • Execute DebateCommand                                   │
└────────────────────────────┬────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────┐
│ Debate Command                                              │
│   • Create new Room                                         │
│   • Switch TUI to Debate Page                               │
│   • Start debate                                            │
└────────────────────────────┬────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────┐
│ Debate Room                                                 │
│   • Spawn AgentInstances (parallel)                         │
│   • Run Strategy                                            │
└────────────────────────────┬────────────────────────────────┘
                             │
         ┌───────────────────┼───────────────────┐
         ▼                   ▼                   ▼
┌─────────────┐      ┌─────────────┐      ┌─────────────┐
│  Agent 1    │      │  Agent 2    │      │  Agent 3    │
│  (Sonnet)   │      │  (Opus)     │      │  (Sonnet)   │
│             │      │             │      │             │
│  Proposal   │      │  Proposal   │      │  Proposal   │
└──────┬──────┘      └──────┬──────┘      └──────┬──────┘
       │                    │                    │
       └────────────────────┼────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│ Strategy                                                    │
│   • Collect proposals                                       │
│   • Broadcast for critiques                                 │
│   • Collect critiques                                       │
│   • Send to moderator for synthesis                         │
└────────────────────────────┬────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────┐
│ Moderator (Opus)                                            │
│   • Analyze all proposals and critiques                     │
│   • Generate synthesis/recommendation                       │
└────────────────────────────┬────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────┐
│ Debate Page TUI                                             │
│   • Display agent panels with streaming                     │
│   • Show transcript                                         │
│   • Display final synthesis                                 │
└─────────────────────────────────────────────────────────────┘
```

---

## 8. Extension Points

### 8.1 Adding a New Phase

```go
// 1. Create phase file in internal/phase/
type CustomPhase struct {
    BasePhase
}

func NewCustomPhase() *CustomPhase {
    return &CustomPhase{
        BasePhase: BasePhase{
            name:   "custom",
            prompt: customPrompt,
            tools:  []string{"view", "edit", "bash"},
        },
    }
}

// 2. Register in phase registry
registry.Register(NewCustomPhase())

// 3. Add slash command in internal/slash/builtin/
type CustomCommand struct { ... }
func (c *CustomCommand) Execute(ctx, args) error {
    return c.agent.SwitchPhase("custom")
}
```

### 8.2 Adding a New Slash Command

```go
// 1. Create command file in internal/slash/builtin/
type MyCommand struct {
    app *app.App
}

func (c *MyCommand) Name() string { return "mycommand" }
func (c *MyCommand) Description() string { return "Does something" }
func (c *MyCommand) Execute(ctx context.Context, args string) error {
    // Implementation
    return nil
}

// 2. Register in router setup
router.Register(&MyCommand{app: app})
```

### 8.3 Adding a New Debate Strategy

```go
// 1. Create strategy file in internal/debate/strategy/
type MyStrategy struct{}

func (s *MyStrategy) Run(ctx context.Context, room *debate.Room) (*debate.Outcome, error) {
    // Custom debate flow
    return &debate.Outcome{...}, nil
}

// 2. Use via config or command
// /debate --strategy=mystrategy "topic"
```

### 8.4 Adding a New Tool

```go
// 1. Create tool file implementing Tool interface
type MyTool struct{}

func (t *MyTool) Name() string { return "my_tool" }
func (t *MyTool) Description() string { return "Does something" }
func (t *MyTool) Schema() map[string]any { return schema }
func (t *MyTool) Execute(ctx context.Context, params map[string]any) (Result, error) {
    // Implementation
    return Result{Content: result}, nil
}

// 2. Register in tool registry
registry.Register(&MyTool{})

// 3. Add to phase's allowed tools if needed
```

### 8.5 Adding a New Provider

```go
// 1. Implement Provider interface
type OllamaProvider struct {
    baseURL string
    model   string
}

func (p *OllamaProvider) Name() string { return "ollama" }
func (p *OllamaProvider) Model() string { return p.model }

func (p *OllamaProvider) CreateMessage(ctx context.Context, req Request) (*Response, error) {
    // Call Ollama API
}

func (p *OllamaProvider) StreamMessage(ctx context.Context, req Request) (<-chan Event, error) {
    // Stream from Ollama
}

// 2. Register in provider registry
registry.Register("ollama", NewOllamaProvider(cfg))
```

---

## Summary

### Architecture Principles

This architecture is built on these key principles:

1. **Clean-Room Implementation** - Inspired by Crush patterns, but our own code
2. **Interface-First Design** - Define contracts before implementations
3. **Layered Architecture** - Clear separation: Presentation → Application → Domain → Infrastructure
4. **Event-Driven Services** - Pub/Sub for decoupled communication
5. **Extensible by Design** - Clear extension points for phases, commands, strategies, tools, providers

### What Makes Socrates Different

```
Standard Chat CLI              Socrates CLI
────────────────────          ────────────────────
Single behavior         →     Phase-based behavior (CDD workflow)
One conversation        →     Debate room (multi-agent)
Specific models         →     Model tiers (Small/Mid/Big)
Commands as hacks       →     First-class slash commands
```

### Estimated Code Size

```
┌─────────────────────────────────────────────────────────────┐
│                    ESTIMATED LINES OF CODE                  │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Infrastructure Layer                                       │
│  ├── pubsub/          ~200 lines                            │
│  ├── database/        ~500 lines (+ generated SQLC)         │
│  ├── provider/        ~800 lines                            │
│  └── tools/           ~1,500 lines                          │
│                                                             │
│  Domain Layer                                               │
│  ├── session/         ~300 lines                            │
│  ├── message/         ~400 lines                            │
│  ├── permission/      ~400 lines                            │
│  └── history/         ~200 lines                            │
│                                                             │
│  Application Layer                                          │
│  ├── app/             ~400 lines                            │
│  ├── agent/           ~600 lines                            │
│  ├── phase/           ~800 lines                            │
│  ├── slash/           ~500 lines                            │
│  ├── debate/          ~1,200 lines                          │
│  ├── config/          ~400 lines                            │
│  └── template/        ~200 lines                            │
│                                                             │
│  Presentation Layer                                         │
│  ├── tui/             ~2,500 lines                          │
│  └── cmd/             ~200 lines                            │
│                                                             │
│  ─────────────────────────────────────────────              │
│  TOTAL ESTIMATE:      ~10,000-12,000 lines                  │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Build Order

```
Phase 1: Foundation        Phase 2: CDD           Phase 3: Debate
─────────────────          ─────────────          ─────────────
pubsub/                    phase/                 debate/
database/                  slash/                 tui/page/debate/
provider/ (Anthropic)      agent/ (PhaseAgent)
tools/ (core)
session/, message/
permission/
tui/ (chat page)

Result: Working CLI        Result: /socrates      Result: /debate
                           /plan, /exec
```

### Key Insight

**We're building a complete CLI from scratch, using Crush as a reference for proven patterns.** This gives us:

- **Full ownership** - No dependency coupling
- **Flexibility** - Free to evolve differently
- **Simplicity** - Only build what we need
- **Understanding** - We know every line of code
