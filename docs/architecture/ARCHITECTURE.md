# CDD - Architecture Document

A CDD-themed AI coding assistant CLI, inspired by the cdd-agent-cli project but built with deliberate architecture from the ground up.

## Vision

CDD is a code assistant where AI agents are themed after CDD characters, each with distinct responsibilities in helping the user (Neo) navigate and modify codebases.

## Agent System - The CDD Metaphor

| Role | Character | Responsibility |
|------|-----------|----------------|
| **User** | Neo | The One - makes decisions, provides vision and intent |
| **General Agent** | Operator (Tank/Link) | Guides Neo through the codebase, provides real-time support, interprets the code |
| **Requirements** | Oracle | Asks the right questions, helps clarify what you *really* want through Socratic dialogue |
| **Planning** | Architect | Designs the implementation, understands structure, creates the blueprint |
| **Execution** | Seraph | Executes with precision, implements the plan, protects code integrity |

### Potential Future Agents

| Character | Potential Role |
|-----------|----------------|
| **Morpheus** | Onboarding/teaching agent ("I can only show you the door...") |
| **Trinity** | Debugging/rescue agent for when things go wrong |
| **Keymaker** | Authentication, secrets, and API key management |
| **Niobe** | Navigation agent for exploring large codebases |
| **Merovingian** | Causality agent - explains the "why" behind code |

## Tech Stack

| Concern | Choice | Rationale |
|---------|--------|-----------|
| **Language** | Python 3.12+ | Rich AI/LLM ecosystem, rapid iteration |
| **Package manager** | `uv` | Fast, modern, excellent for CLI distribution |
| **CLI framework** | `typer` | Type-hint based API, built on Click |
| **TUI** | `textual` | Rich terminal UI components |
| **Terminal output** | `rich` | Beautiful formatting, syntax highlighting |
| **Validation** | `pydantic` | Runtime validation, settings management |
| **LLM clients** | `anthropic`, `openai` | Direct SDK usage, no wrapper overhead |
| **Async** | `asyncio` | Native async for parallel agent execution |
| **Testing** | `pytest` + `pytest-asyncio` | Standard, well-supported |
| **Linting** | `ruff` | Fast, replaces flake8/black/isort |
| **Type checking** | `mypy --strict` | Catch issues at development time |

## Project Structure

```
cdd/
├── src/cdd/
│   ├── __init__.py
│   ├── main.py                     # Typer app entry point
│   │
│   ├── core/
│   │   ├── __init__.py
│   │   ├── loop.py                 # Base async agent loop (ReAct pattern)
│   │   ├── config.py               # Pydantic settings & configuration
│   │   ├── context.py              # Context loading (project, global)
│   │   │
│   │   ├── providers/
│   │   │   ├── __init__.py
│   │   │   ├── base.py             # Provider protocol/interface
│   │   │   ├── anthropic.py        # Anthropic implementation
│   │   │   ├── openai.py           # OpenAI implementation
│   │   │   └── factory.py          # Provider factory
│   │   │
│   │   └── tools/
│   │       ├── __init__.py
│   │       ├── registry.py         # Tool registry with decorators
│   │       ├── approval.py         # Risk-based approval system
│   │       ├── files.py            # File operations
│   │       ├── search.py           # Glob, grep operations
│   │       ├── shell.py            # Bash execution
│   │       └── git.py              # Git operations
│   │
│   ├── agents/
│   │   ├── __init__.py
│   │   ├── base.py                 # Base agent class
│   │   ├── operator.py             # General assistant (Tank/Link)
│   │   ├── oracle.py               # Requirements gathering (Socrates)
│   │   ├── architect.py            # Planning agent
│   │   └── seraph.py               # Execution agent
│   │
│   ├── session/
│   │   ├── __init__.py
│   │   ├── conversation.py         # Conversation history management
│   │   ├── state.py                # Session state
│   │   └── compaction.py           # Context window management
│   │
│   ├── tui/
│   │   ├── __init__.py
│   │   ├── app.py                  # Main Textual application
│   │   ├── chat.py                 # Chat interface component
│   │   └── widgets/                # Custom widgets
│   │
│   └── utils/
│       ├── __init__.py
│       ├── markdown.py             # Markdown rendering
│       └── yaml.py                 # YAML parsing
│
├── tests/
│   ├── __init__.py
│   ├── conftest.py                 # Pytest fixtures
│   ├── core/
│   ├── agents/
│   └── integration/
│
├── pyproject.toml
├── README.md
└── ARCHITECTURE.md
```

## Core Design Patterns

### 1. Provider Strategy Pattern

Clean abstraction over LLM providers enabling provider-agnostic operation:

```python
class ProviderProtocol(Protocol):
    async def create_message(
        self,
        messages: list[Message],
        tools: list[Tool],
        system: str,
    ) -> Response: ...

    async def stream_message(
        self,
        messages: list[Message],
        tools: list[Tool],
        system: str,
    ) -> AsyncGenerator[Event, None]: ...
```

### 2. Tool Registry Pattern

Decorator-based registration with auto-schema generation:

```python
@registry.register(risk_level=RiskLevel.SAFE)
async def read_file(path: str) -> str:
    """Read contents of a file."""
    ...
```

### 3. Risk-Based Approval

```python
class RiskLevel(Enum):
    SAFE = "safe"       # Read-only operations
    MEDIUM = "medium"   # File modifications
    HIGH = "high"       # Shell execution, destructive ops
```

### 4. Model Tiers

Abstract over specific model names:

```python
class ModelTier(Enum):
    SMALL = "small"   # Fast, cheap (haiku, gpt-4o-mini)
    MID = "mid"       # Balanced (sonnet, gpt-4o)
    BIG = "big"       # Maximum capability (opus, gpt-4)
```

### 5. Async ReAct Loop

```python
async def run(self, user_message: str) -> str:
    self.messages.append(user_msg(user_message))

    for _ in range(self.max_iterations):
        response = await self.provider.create_message(
            messages=self.messages,
            tools=self.tools,
            system=self.system_prompt,
        )

        if response.stop_reason == "end_turn":
            return extract_text(response)

        if response.stop_reason == "tool_use":
            self.messages.append(assistant_msg(response.content))
            tool_results = await self.execute_tools(response.content)
            self.messages.append(user_msg(tool_results))
```

## Development Principles

1. **Strict typing from day one** - `mypy --strict` in CI
2. **Clear module boundaries** - Each module has a single responsibility
3. **Pydantic everywhere** - All data structures validated
4. **Test coverage early** - Write tests as you build
5. **Async native** - Design for concurrency from the start

## Lineage

CDD takes inspiration from [cdd-agent-cli](https://github.com/...), preserving its validated patterns:

- Provider abstraction strategy
- Tool registry with risk levels
- ReAct agentic loop
- Hierarchical context loading
- Conversation compaction

While improving on:

- Stricter type safety
- Async-first design
- Cleaner module boundaries
- More deliberate architecture

---

*"Unfortunately, no one can be told what the CDD is. You have to see it for yourself."* - Morpheus
