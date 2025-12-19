# CMD Layer Documentation

> The `cmd/` package is the entry point layer for CDD CLI. It handles command-line parsing, initialization, and bootstrapping of the application.

---

## Overview

The CMD layer is built with [Cobra](https://github.com/spf13/cobra) and serves as the presentation layer for CLI commands. It is responsible for:

- Parsing command-line arguments and flags
- Initializing the application infrastructure
- Coordinating the startup sequence
- Launching the Terminal User Interface (TUI)

---

## Files

| File | Purpose |
|------|---------|
| `root.go` | Root command definition, main entry point, and TUI launcher |
| `version.go` | Version subcommand for displaying build information |

---

## Flow Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              CMD LAYER FLOW                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  User runs: cdd [--debug]                                                   │
│       │                                                                     │
│       ▼                                                                     │
│  ┌─────────────┐                                                            │
│  │  Execute()  │  Entry point called from main.go                           │
│  └──────┬──────┘                                                            │
│         │                                                                   │
│         ▼                                                                   │
│  ┌─────────────────┐                                                        │
│  │ newRootCmd()    │  Creates the Cobra root command                        │
│  │                 │  • Registers --debug flag                              │
│  │                 │  • Adds version subcommand                             │
│  └────────┬────────┘                                                        │
│           │                                                                 │
│           ▼                                                                 │
│  ┌─────────────────┐                                                        │
│  │    runTUI()     │  Main execution function                               │
│  └────────┬────────┘                                                        │
│           │                                                                 │
│           │  (See detailed steps below)                                     │
│           ▼                                                                 │
│  ┌─────────────────┐                                                        │
│  │   TUI Running   │                                                        │
│  └─────────────────┘                                                        │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Detailed Step-by-Step Flow

### Step 1: Enable Debug Logging (Optional)

**What happens:**
- If `--debug` flag is set, debug logging is enabled
- Log file is created at `~/.local/share/cdd/debug.log`
- Deferred cleanup ensures logs are properly closed on exit

**Layer Reference:**
→ `internal/debug` module

```go
if debugMode {
    logPath := filepath.Join(xdg.DataHome, "cdd", "debug.log")
    debug.Enable(logPath)
    defer debug.Disable()
}
```

---

### Step 2: Check First Run Status

**What happens:**
- Determines if this is the user's first time running CDD
- Used to decide whether to show the configuration wizard

**Layer Reference:**
→ `internal/config` module (`IsFirstRun()` function)

```go
isFirstRun := config.IsFirstRun()
```

---

### Step 3: Load Configuration

**What happens:**
- Attempts to load configuration from disk (`~/.config/cdd/cdd.json`)
- If loading fails, creates an empty configuration for the wizard
- Configuration includes: providers, selected models, and options

**Layer Reference:**
→ `internal/config` module (`Load()` and `NewConfig()` functions)

```go
cfg, err := config.Load()
if err != nil {
    cfg = config.NewConfig()
}
```

---

### Step 4: Load Providers

**What happens:**
- Retrieves the list of known LLM providers from configuration
- If not already loaded, attempts to load providers
- Providers include: Anthropic, OpenAI, and compatible APIs

**Layer Reference:**
→ `internal/config` module (`KnownProviders()` and `LoadProviders()` functions)

```go
providers := cfg.KnownProviders()
if len(providers) == 0 {
    providers, err = config.LoadProviders(cfg)
}
```

---

### Step 5: Create Agent (If Not First Run)

**What happens:**
- If valid configuration exists and it's not the first run:
  - Calls `createAgent()` to initialize the AI agent
  - Agent creation involves building models and setting up tools
- If agent creation fails, continues without agent (wizard will configure it)

**Layer Reference:**
→ Internally calls `createAgent()` which coordinates:
  - `internal/provider` module (model building)
  - `internal/tools` module (tool registry)
  - `internal/agent` module (agent initialization)

```go
if !isFirstRun {
    ag, err = createAgent(cfg)
}
```

---

### Step 6: Create Agent Factory

**What happens:**
- Creates a factory function that can reload config and create a new agent
- This factory is passed to the TUI so it can create the agent after wizard completion
- Enables hot-reloading of agent without restarting the application

**Layer Reference:**
→ Factory internally uses:
  - `internal/config` module (`Load()` function)
  - `createAgent()` function

```go
agentFactory := func() (*agent.DefaultAgent, error) {
    newCfg, err := config.Load()
    return createAgent(newCfg)
}
```

---

### Step 7: Launch TUI

**What happens:**
- Passes all initialized components to the TUI
- TUI takes over the terminal and handles user interaction
- Components passed: providers, first-run status, agent, agent factory

**Layer Reference:**
→ `internal/tui` module (`Run()` function)

```go
return tui.Run(providers, isFirstRun, ag, agentFactory)
```

---

## Agent Creation Flow (`createAgent`)

When `createAgent()` is called, it performs the following steps:

### Step A: Build Models

**What happens:**
- Creates a provider builder with the configuration
- Builds the "large" model for complex tasks
- Optionally builds a "small" model for simpler tasks
- Handles OAuth token refresh if needed

**Layer Reference:**
→ `internal/provider` module (`NewBuilder()` and `BuildModels()` functions)

```go
builder := provider.NewBuilder(cfg)
largeModel, _, err := builder.BuildModels(ctx)
```

---

### Step B: Get Working Directory

**What happens:**
- Determines the current working directory
- This becomes the base path for file operations

**Layer Reference:**
→ Standard library (`os.Getwd()`)

```go
cwd, err := os.Getwd()
```

---

### Step C: Create Tools Registry

**What happens:**
- Initializes the default tool registry
- Registers all available tools: `read`, `write`, `edit`, `glob`, `grep`, `bash`
- Tools are scoped to the working directory for security

**Layer Reference:**
→ `internal/tools` module (`DefaultRegistry()` function)

```go
registry := tools.DefaultRegistry(cwd)
```

---

### Step D: Create Agent

**What happens:**
- Assembles the agent configuration with:
  - Language model from the provider
  - All registered tools
  - Default system prompt
- Creates and returns the agent instance

**Layer Reference:**
→ `internal/agent` module (`New()` function and `Config` struct)

```go
agentCfg := agent.Config{
    Model:        largeModel.Model,
    Tools:        registry.All(),
    SystemPrompt: defaultSystemPrompt,
}
return agent.New(agentCfg), nil
```

---

## Version Command Flow

The `version` subcommand is simple and self-contained:

```
cdd version
    │
    ▼
┌─────────────────────────────────────┐
│ Prints:                             │
│   cdd {Version}                     │
│     commit: {Commit}                │
│     built:  {BuildDate}             │
└─────────────────────────────────────┘
```

**Note:** Version information is injected at build time via ldflags.

---

## Layer Dependencies

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           CMD LAYER DEPENDENCIES                            │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  cmd/                                                                       │
│    │                                                                        │
│    ├──▶ internal/debug      (debug logging)                                 │
│    │                                                                        │
│    ├──▶ internal/config     (configuration loading/management)              │
│    │                                                                        │
│    ├──▶ internal/provider   (LLM provider building)                         │
│    │                                                                        │
│    ├──▶ internal/tools      (tool registry creation)                        │
│    │                                                                        │
│    ├──▶ internal/agent      (agent initialization)                          │
│    │                                                                        │
│    └──▶ internal/tui        (terminal UI launch)                            │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## System Prompt

The CMD layer defines the default system prompt for the agent:

```
You are CDD (Context-Driven Development), an AI coding assistant.

You help developers write, understand, and improve code through structured workflows.

When working with code:
1. Read files before modifying them
2. Use appropriate tools for the task
3. Explain your reasoning clearly
4. Ask clarifying questions when requirements are unclear

Available tools allow you to read files, search code, write files, edit code, 
and execute shell commands.
```

**Note:** For OAuth authentication (Claude Code), an additional header is prepended by the agent layer: `"You are Claude Code, Anthropic's official CLI for Claude."`

---

## Summary

| Step | Action | Target Layer/Module |
|------|--------|---------------------|
| 1 | Enable debug logging | `internal/debug` |
| 2 | Check first run | `internal/config` |
| 3 | Load configuration | `internal/config` |
| 4 | Load providers | `internal/config` |
| 5 | Create agent | `internal/provider`, `internal/tools`, `internal/agent` |
| 6 | Create agent factory | (closure using config and createAgent) |
| 7 | Launch TUI | `internal/tui` |
