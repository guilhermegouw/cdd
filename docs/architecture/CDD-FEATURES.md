# CDD CLI - Feature Roadmap

> Features organized by category to support incremental development and clear prioritization.

---

## Table of Contents

1. [Overview](#overview)
2. [Crush Reusable Components](#crush-reusable-components)
3. [Core Features](#core-features)
4. [Innovative Features](#innovative-features)
5. [Feature Dependencies](#feature-dependencies)
6. [Suggested Roadmap](#suggested-roadmap)

---

## Overview

This document separates CDD CLI features into three categories:

| Category | Description | Priority |
|----------|-------------|----------|
| **From Crush** | Production-ready components to port from Crush CLI | Port first |
| **Core** | Essential features any coding agent CLI needs to function | Must have for MVP |
| **Innovative** | Differentiating features that make CDD CLI unique | Post-MVP enhancements |

---

## Crush as Reference

We use [Crush CLI](/home/guilhermegouw/code/crush) as a **reference implementation** for proven patterns, but we build our own code from scratch.

### Why Clean-Room Instead of Port?

| Approach | Pros | Cons |
|----------|------|------|
| **Port from Crush** | Faster initial development | Dependency coupling, less flexibility |
| **Clean-Room (Chosen)** | Full ownership, flexibility to evolve | More initial work |

### Patterns We Adopt from Crush

| Pattern | Description | Our Implementation |
|---------|-------------|-------------------|
| **Pub/Sub Broker** | Generic `Broker[T]` for events | `internal/pubsub/` |
| **Service + Broker** | Each service owns data + publishes events | `internal/session/`, `message/`, etc. |
| **TEA Architecture** | Model → Update → View for TUI | `internal/tui/` |
| **Tool Interface** | Common contract for agent tools | `internal/tools/` |
| **Permission Flow** | Blocking channel for approvals | `internal/permission/` |

### What We DON'T Need from Crush

| Component | Why Skip |
|-----------|----------|
| LSP Integration | Not needed for MVP, add later if wanted |
| MCP Support | Not needed for MVP, add later if wanted |
| Catwalk library | We'll implement simpler provider abstraction |
| Fantasy library | We'll implement our own agent loop |

### Summary: Build Fresh, Reference Often

```
Approach:           Clean-room implementation
Reference:          Crush patterns and architecture
Dependency:         None (fully independent)
Flexibility:        Maximum (free to evolve)

What we BUILD ourselves:
🔨 Pub/Sub system (~200 lines)
🔨 Database layer (~500 lines)
🔨 Provider abstraction (~800 lines)
🔨 All tools (~1,500 lines)
🔨 Services (~1,300 lines)
🔨 TUI (~2,500 lines)
🔨 Phase system (~800 lines)
🔨 Slash commands (~500 lines)
🔨 Debate room (~1,200 lines)

TOTAL: ~10,000-12,000 lines (our own code)
```

---

## Core Features

These are the foundational features required for a functional coding agent CLI.

> **Note:** We build these ourselves, using Crush patterns as reference.

### 1. Chat System

| Feature | Description | Status | Effort |
|---------|-------------|--------|--------|
| Interactive Chat | REPL-style conversation with AI assistant | 🔨 Build | Medium |
| Streaming Responses | Real-time token streaming for better UX | 🔨 Build | Medium |
| Conversation History | In-memory message history within session | 🔨 Build | Low |
| System Prompts | Configurable system prompts for agent behavior | 🔨 Build | Low |

**Location:** `internal/agent/`, `internal/message/`

---

### 2. LLM Provider

| Feature | Description | Status | Effort |
|---------|-------------|--------|--------|
| Provider Interface | Abstract contract for LLM communication | 🔨 Build | Low |
| Anthropic Provider | Claude models support | 🔨 Build | Medium |
| OpenAI Provider | GPT models support | 🔨 Build | Medium |
| API Key Configuration | Secure storage and usage of API keys | 🔨 Build | Low |
| Message Formatting | Convert internal messages to provider format | 🔨 Build | Medium |
| Error Handling | Graceful handling of API errors and rate limits | 🔨 Build | Low |
| Model Tier Abstraction | Small/Mid/Big model selection | 🔨 Build | Low |

**Location:** `internal/provider/`

---

### 3. File Tools

| Feature | Description | Risk Level | Status | Effort |
|---------|-------------|------------|--------|--------|
| `read` | Read file contents | SAFE | 🔨 Build | Low |
| `edit` | Replace text in files | MEDIUM | 🔨 Build | Medium |
| `write` | Write/create files | MEDIUM | 🔨 Build | Low |
| `ls` | List directory contents | SAFE | 🔨 Build | Low |
| `glob` | Find files by pattern | SAFE | 🔨 Build | Low |
| `grep` | Search file contents | SAFE | 🔨 Build | Medium |

**Location:** `internal/tools/`

---

### 4. Shell Execution

| Feature | Description | Risk Level | Status | Effort |
|---------|-------------|------------|--------|--------|
| `bash` | Execute shell commands | HIGH | 🔨 Build | Medium |
| Output Capture | Capture stdout/stderr | - | 🔨 Build | Low |
| Timeout Handling | Prevent hanging commands | - | 🔨 Build | Low |
| Working Directory | Execute in correct directory | - | 🔨 Build | Low |
| Background Jobs | Non-blocking long-running commands | - | 🔨 Build | Medium |

**Location:** `internal/tools/bash.go`, `internal/tools/job.go`

---

### 5. Permission System

| Feature | Description | Status | Effort |
|---------|-------------|--------|--------|
| Tool Risk Classification | Classify tools as SAFE/MEDIUM/HIGH | 🔨 Build | Low |
| User Approval Flow | Ask user before risky operations | 🔨 Build | Medium |
| Approval UI | Modal dialog for permission requests | 🔨 Build | Medium |
| Approval Modes | Paranoid/Balanced/YOLO modes | 🔨 Build | Low |

**Location:** `internal/permission/`, `internal/tui/dialog/permission.go`

---

### 6. Context Loading

| Feature | Description | Status | Effort |
|---------|-------------|--------|--------|
| Project Detection | Detect project root (git, package.json, etc.) | 🔨 Build | Low |
| Context File Loading | Load CDD.md, .cursorrules | 🔨 Build | Low |
| Context Injection | Include context in system prompt | 🔨 Build | Low |

**Location:** `internal/config/`

---

### 7. TUI Foundation

| Feature | Description | Status | Effort |
|---------|-------------|--------|--------|
| Chat Page | Main conversation interface | 🔨 Build | High |
| Message Display | Render messages with markdown | 🔨 Build | Medium |
| Input Editor | Multi-line input with basic editing | 🔨 Build | Medium |
| Basic Keybindings | Essential navigation (submit, quit, scroll) | 🔨 Build | Low |
| Session Sidebar | List and switch sessions | 🔨 Build | Medium |
| Model Picker Dialog | Select model/provider | 🔨 Build | Medium |
| Session Picker Dialog | Manage sessions | 🔨 Build | Medium |
| Status Bar | Show session, model, tokens | 🔨 Build | Low |

**Location:** `internal/tui/`

---

### 8. CLI Structure

| Feature | Description | Status | Effort |
|---------|-------------|--------|--------|
| `run` Command | Start interactive chat session | 🔨 Build | Low |
| `login` Command | OAuth authentication | 🔨 Build | Medium |
| `--help` Flag | Display usage information | 🔨 Build | Low |
| `--version` Flag | Display version | 🔨 Build | Low |
| Config File Support | Load settings from config file | 🔨 Build | Medium |

**Location:** `cmd/`

---

### 9. Persistence

| Feature | Description | Status | Effort |
|---------|-------------|--------|--------|
| SQLite Database | Persistent storage | 🔨 Build | Medium |
| Session Persistence | Save/resume sessions | 🔨 Build | Medium |
| Message History | Store conversation history | 🔨 Build | Medium |
| File History | Track file modifications | 🔨 Build | Low |
| Migrations | Database schema management (Goose) | 🔨 Build | Low |

**Location:** `internal/database/`, `internal/session/`, `internal/message/`, `internal/history/`

---

### Core Features Summary

```
Total Core Features:        ~45 items
All New Development:        100%

Estimated Lines of Code for Core:
├── Provider:      ~800 lines
├── Tools:         ~1,500 lines
├── Services:      ~1,300 lines
├── Permission:    ~400 lines
├── Database:      ~500 lines
├── TUI:           ~2,500 lines
├── Config:        ~400 lines
└── Total Core:    ~7,400 lines
```

---

## Innovative Features

These features differentiate CDD CLI from other coding agents.

> **Note:** Many originally "innovative" features are already in Crush! Only truly new features need development.

### 1. CDD Workflow (Phases) - 🔨 NEW

| Feature | Description | Status |
|---------|-------------|--------|
| Phase System | Switchable agent behavior modes | 🔨 Build |
| CDD Phase | Requirements gathering with questions | 🔨 Build |
| Planner Phase | Implementation planning | 🔨 Build |
| Executor Phase | Code execution with full tools | 🔨 Build |
| Phase Transitions | Smooth handoff between phases | 🔨 Build |
| `/cdd` Command | Enter CDD phase | 🔨 Build |
| `/plan` Command | Enter Planner phase | 🔨 Build |
| `/exec` Command | Enter Executor phase | 🔨 Build |
| Tool Filtering by Phase | Restrict tools per phase | 🔨 Build |

**Why Innovative:** Structured workflow for complex tasks, not just open-ended chat.

**Dependencies:** Core chat (from Crush), slash command router (build)

**Effort:** Medium - This is the main differentiator from Crush

---

### 2. Debate Room - 🔨 NEW

| Feature | Description | Status |
|---------|-------------|--------|
| Agent Instances | Spawn multiple independent agents | 🔨 Build |
| Debate Orchestrator | Manage multi-agent discussions | 🔨 Build |
| Proposal Round | Agents propose solutions in parallel | 🔨 Build |
| Critique Round | Agents critique each other's proposals | 🔨 Build |
| Synthesis | Moderator synthesizes recommendations | 🔨 Build |
| Debate Strategies | Parallel, round-robin, adversarial | 🔨 Build |
| `/debate` Command | Start a debate session | 🔨 Build |
| Debate Page TUI | Dedicated UI for debates | 🔨 Build |

**Why Innovative:** Multi-agent exploration of solutions before implementation.

**Dependencies:** Multi-provider (from Crush), agent factory (build), TUI page (build)

**Effort:** High - Most complex new feature

---

### 3. Multi-Provider Support

| Feature | Description | Status | Effort |
|---------|-------------|--------|--------|
| Provider Interface | Abstract provider contract | 🔨 Build | Low |
| Provider Registry | Manage multiple providers | 🔨 Build | Low |
| Anthropic Provider | Claude models support | 🔨 Build | Medium |
| OpenAI Provider | GPT models support | 🔨 Build | Medium |
| Runtime Switching | Change providers during session | 🔨 Build | Low |

**Location:** `internal/provider/`

**Note:** Start with Anthropic + OpenAI. Add more providers later as needed.

---

### 4. Model Tier Abstraction

| Feature | Description | Status |
|---------|-------------|--------|
| Tier Definition | Small/Mid/Big abstraction | 🔨 Build |
| Tier Mapping | Map tiers to specific models per provider | 🔨 Build |
| Tier Selection | User chooses tier, not specific model | 🔨 Build |
| Cost Awareness | Users can optimize for cost vs capability | 🔨 Build |

**Why Innovative:** Simplifies model selection for CDD workflow and Debate Room.

**Dependencies:** Multi-provider (from Crush)

**Effort:** Low - Configuration layer on top of Crush's provider system

---

### 5. Advanced Approval Modes

| Feature | Description | Status | Effort |
|---------|-------------|--------|--------|
| YOLO Mode | Skip all permission requests | 🔨 Build | Low |
| Allowed Tools List | Auto-approve specific tools | 🔨 Build | Low |
| Persistent Grants | Remember user approvals | 🔨 Build | Medium |
| Modal Approval UI | Clear permission dialogs | 🔨 Build | Medium |

**Location:** `internal/permission/`

---

### 6. Background Processes

| Feature | Description | Status | Effort |
|---------|-------------|--------|--------|
| Background Jobs in bash | Start non-blocking commands | 🔨 Build | Medium |
| `job_output` | Get buffered output | 🔨 Build | Low |
| `job_kill` | Stop running process | 🔨 Build | Low |
| Output Buffering | Capture background output | 🔨 Build | Medium |

**Location:** `internal/tools/job.go`

---

### 7. Session Persistence

| Feature | Description | Status | Effort |
|---------|-------------|--------|--------|
| SQLite Database | Persistent storage | 🔨 Build | Medium |
| Session Service | CRUD + pub/sub | 🔨 Build | Medium |
| Message Service | Store conversation history | 🔨 Build | Medium |
| Session Picker | UI to select/resume sessions | 🔨 Build | Medium |

**Location:** `internal/database/`, `internal/session/`, `internal/message/`

---

### 8. Pub/Sub Event System

| Feature | Description | Status | Effort |
|---------|-------------|--------|--------|
| Generic Broker | Type-safe `Broker[T]` | 🔨 Build | Low |
| Service Events | Events for session, message, permission | 🔨 Build | Low |
| TUI Integration | Forward events via program.Send() | 🔨 Build | Medium |
| Decoupled Architecture | Services communicate via events | 🔨 Build | - |

**Location:** `internal/pubsub/`

---

### 9. Git Integration - 🔨 ENHANCE

| Feature | Description | Status |
|---------|-------------|--------|
| `git_status` | Show working tree status | 🔨 Build (dedicated tool) |
| `git_diff` | Show file diffs | 🔨 Build (dedicated tool) |
| `git_log` | Show commit history | 🔨 Build (dedicated tool) |
| `/commit` Command | AI-assisted commits | 🔨 Build |

**Note:** Crush can do git via `bash`, but dedicated tools provide better output formatting.

**Dependencies:** Shell execution (from Crush)

**Effort:** Low - Wrapper tools around git commands

---

### 10. Project Initialization - 🔨 NEW

| Feature | Description | Status |
|---------|-------------|--------|
| `/init` Command | Initialize project for CDD | 🔨 Build |
| `/new` Command | Create from templates | 🔨 Build |
| Template System | Feature spec, bug plan, enhancement templates | 🔨 Build |
| CDD.md Generation | Create initial CDD.md file | 🔨 Build |

**Why Innovative:** Structured project setup for CDD workflow.

**Dependencies:** File tools (from Crush)

**Effort:** Low - File generation utilities

---

### 11. Slash Command Router - 🔨 NEW

| Feature | Description | Status |
|---------|-------------|--------|
| Command Parser | Parse `/command args` syntax | 🔨 Build |
| Command Registry | Register available commands | 🔨 Build |
| Help System | `/help` shows available commands | 🔨 Build |
| Command Routing | Route to appropriate handler | 🔨 Build |

**Why Needed:** Required for CDD workflow commands.

**Dependencies:** TUI input (from Crush)

**Effort:** Low - Simple routing system

---

## Feature Dependencies

```
FROM CRUSH (Port First) ──────────────────────────────────────────────────────┐
│                                                                              │
│  Foundation          Services           Tools              TUI              │
│  ┌──────────┐       ┌──────────┐       ┌──────────┐       ┌──────────┐     │
│  │ Pub/Sub  │       │ Session  │       │ view     │       │ Chat Page│     │
│  │ Database │       │ Message  │       │ edit     │       │ Dialogs  │     │
│  │ Config   │       │ Permission│      │ write    │       │ Editor   │     │
│  │ OAuth    │       │ History  │       │ glob/grep│       │ Messages │     │
│  └──────────┘       └──────────┘       │ bash/job │       │ Status   │     │
│                                        └──────────┘       └──────────┘     │
│                                                                              │
└──────────────────────────────────────────────────────────────────────────────┘
                                        │
                                        ▼
                              Crush Ported = Working CLI
                                        │
                                        ▼
NEW FEATURES (Build on top of Crush) ──────────────────────────────────────────┐
│                                                                              │
│  Phase 1: Slash Commands & CDD ──────────────────────────────────────────   │
│  ┌──────────────────┐                                                       │
│  │ Slash Router     │◄── Required for all /commands                         │
│  │ • Parser         │                                                       │
│  │ • Registry       │                                                       │
│  │ • /help          │                                                       │
│  └────────┬─────────┘                                                       │
│           │                                                                  │
│           ▼                                                                  │
│  ┌──────────────────┐     ┌──────────────────┐                              │
│  │ Phase System     │     │ Model Tiers      │                              │
│  │ • CDD            │     │ • Small/Mid/Big  │                              │
│  │ • Planner        │     │ • Per-provider   │                              │
│  │ • Executor       │     │   mapping        │                              │
│  └────────┬─────────┘     └──────────────────┘                              │
│           │                                                                  │
│           ▼                                                                  │
│  ┌──────────────────┐                                                       │
│  │ CDD Commands     │                                                       │
│  │ • /cdd           │                                                       │
│  │ • /plan          │                                                       │
│  │ • /exec          │                                                       │
│  └──────────────────┘                                                       │
│                                                                              │
│  Phase 2: Project Tools ─────────────────────────────────────────────────   │
│  ┌──────────────────┐     ┌──────────────────┐                              │
│  │ Project Init     │     │ Git Tools        │                              │
│  │ • /init          │     │ • git_status     │                              │
│  │ • /new           │     │ • git_diff       │                              │
│  │ • Templates      │     │ • git_log        │                              │
│  │ • CDD.md gen     │     │ • /commit        │                              │
│  └──────────────────┘     └──────────────────┘                              │
│                                                                              │
│  Phase 3: Debate Room (Most Complex) ────────────────────────────────────   │
│  ┌──────────────────┐                                                       │
│  │ Agent Factory    │◄── Spawns multiple independent agents                 │
│  │ • AgentInstance  │                                                       │
│  └────────┬─────────┘                                                       │
│           │                                                                  │
│           ▼                                                                  │
│  ┌──────────────────┐     ┌──────────────────┐                              │
│  │ Debate Room      │     │ Debate TUI Page  │                              │
│  │ • Orchestrator   │────▶│ • Agent panels   │                              │
│  │ • Strategies     │     │ • Transcript     │                              │
│  │ • Synthesis      │     │ • Controls       │                              │
│  └──────────────────┘     └──────────────────┘                              │
│                                                                              │
└──────────────────────────────────────────────────────────────────────────────┘
```

---

## Suggested Roadmap

> **Clean-Room Approach:** We build everything ourselves, using Crush as a reference for proven patterns.

### Phase 1: Foundation (Working CLI)

**Goal:** Build a basic working CLI from scratch

| Priority | Component | Description | Effort |
|----------|-----------|-------------|--------|
| 1 | Project setup | go.mod, folder structure, Makefile | Low |
| 2 | `internal/pubsub/` | Generic Broker[T] for events | Low |
| 3 | `internal/database/` | SQLite + SQLC + migrations | Medium |
| 4 | `internal/provider/` | Anthropic provider (start with one) | Medium |
| 5 | `internal/tools/` | Core tools: read, write, edit, bash | Medium |
| 6 | `internal/session/` | Session service + broker | Medium |
| 7 | `internal/message/` | Message service + broker | Medium |
| 8 | `internal/permission/` | Basic permission system | Medium |
| 9 | `internal/agent/` | Agent loop (ReAct pattern) | High |
| 10 | `internal/tui/` | Basic chat page | High |
| 11 | `cmd/` | CLI commands (run, version) | Low |

**Deliverable:** Working CLI with chat, tools, and persistence.

**Estimated Code:** ~5,000-6,000 lines

---

### Phase 2: CDD Workflow (Main Differentiator)

**Goal:** Implement the CDD → Planner → Executor workflow

| Priority | Component | Description | Effort |
|----------|-----------|-------------|--------|
| 1 | `internal/slash/` | Slash command router | Low |
| 2 | `internal/phase/` | Phase interface and registry | Medium |
| 3 | CDD Phase | Requirements gathering prompts + tools | Medium |
| 4 | Planner Phase | Planning prompts + tools | Medium |
| 5 | Executor Phase | Execution prompts + full tools | Low |
| 6 | `/cdd`, `/plan`, `/exec` | Slash commands | Low |
| 7 | Tool filtering | Filter tools by current phase | Low |

**Deliverable:** Full CDD workflow. `/cdd` → `/plan` → `/exec`

**Estimated Code:** ~1,500 lines

---

### Phase 3: Polish & Multi-Provider

**Goal:** Add OpenAI, model tiers, and project tools

| Priority | Component | Description | Effort |
|----------|-----------|-------------|--------|
| 1 | OpenAI Provider | GPT models support | Medium |
| 2 | Model Tiers | Small/Mid/Big abstraction | Low |
| 3 | `/init` command | Create CDD.md | Low |
| 4 | `/new` command | Templates for specs | Low |
| 5 | Git tools | git_status, git_diff, git_log | Low |
| 6 | `/commit` command | AI-assisted commits | Low |
| 7 | Session sidebar | Session management UI | Medium |
| 8 | Model picker dialog | Provider/model selection UI | Medium |

**Deliverable:** Multi-provider, cost control, project initialization.

**Estimated Code:** ~2,000 lines

---

### Phase 4: Debate Room

**Goal:** Multi-agent solution exploration

| Priority | Component | Description | Effort |
|----------|-----------|-------------|--------|
| 1 | `internal/debate/instance.go` | AgentInstance (spawnable agent) | Medium |
| 2 | `internal/debate/factory.go` | Agent factory | Medium |
| 3 | `internal/debate/room.go` | DebateRoom orchestrator | High |
| 4 | `internal/debate/strategy/` | Parallel, round-robin strategies | Medium |
| 5 | Synthesis generation | Moderator synthesis | Medium |
| 6 | `/debate` command | Start debate | Low |
| 7 | `internal/tui/page/debate/` | Debate page TUI | High |

**Deliverable:** Full multi-agent debate capability.

**Estimated Code:** ~2,500 lines

---

### Phase 5: Production Polish

**Goal:** Production readiness

| Priority | Component | Description | Effort |
|----------|-----------|-------------|--------|
| 1 | Error handling | Comprehensive error messages | Medium |
| 2 | Logging | Structured logging | Low |
| 3 | Config validation | Validate config on load | Low |
| 4 | Tests | Unit and integration tests | High |
| 5 | Documentation | README, usage docs | Medium |

**Deliverable:** Production-ready CLI.

---

## Summary

### What We're Building

| Phase | Focus | Lines of Code |
|-------|-------|---------------|
| Phase 1 | Foundation (Working CLI) | ~5,500 lines |
| Phase 2 | CDD Workflow | ~1,500 lines |
| Phase 3 | Polish & Multi-Provider | ~2,000 lines |
| Phase 4 | Debate Room | ~2,500 lines |
| Phase 5 | Production Polish | ~500 lines |
| **Total** | **Complete CLI** | **~12,000 lines** |

### Build Order Principle

```
Phase 1: Foundation        Phase 2: CDD           Phase 3: Polish
─────────────────          ─────────────          ─────────────
pubsub/                    phase/                 OpenAI provider
database/                  slash/                 Model tiers
provider/ (Anthropic)      PhaseAgent             /init, /new
tools/ (core)                                     Git tools
services/                                         UI improvements
agent/
tui/ (basic)

Result: Working CLI        Result: CDD Flow       Result: Full Features
```

```
Phase 4: Debate            Phase 5: Polish
─────────────────          ─────────────
debate/instance            Error handling
debate/room                Logging
debate/strategy            Tests
tui/page/debate            Documentation

Result: Multi-Agent        Result: Production
```

### Key Advantages of Clean-Room

- **Full Ownership** - Every line is ours
- **Flexibility** - Free to evolve in any direction
- **Simplicity** - Only build what we need
- **Understanding** - Deep knowledge of the codebase
- **No Dependencies** - No coupling to external projects
