# CDD - Context-Driven Development

[![Go Version](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat&logo=go)](https://go.dev/)
[![CI](https://github.com/guilhermegouw/cdd/actions/workflows/ci.yml/badge.svg)](https://github.com/guilhermegouw/cdd/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/guilhermegouw/cdd/graph/badge.svg)](https://codecov.io/gh/guilhermegouw/cdd)
[![Go Report Card](https://goreportcard.com/badge/github.com/guilhermegouw/cdd)](https://goreportcard.com/report/github.com/guilhermegouw/cdd)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

An AI-powered coding assistant CLI that implements structured development workflows.

## Overview

CDD (Context-Driven Development) helps you write, understand, and improve your code through a structured workflow:

- **Socrates**: Clarify requirements through dialogue
- **Planner**: Design implementation strategy
- **Executor**: Write and modify code
- **Sync-Docs**: Keep documentation aligned with reality

## Features

- Multi-provider support (Anthropic, OpenAI, OpenAI-compatible APIs)
- Two-tier model system (large for complex reasoning, small for quick tasks)
- OAuth and API key authentication
- Interactive TUI setup wizard

## Installation

```bash
go install github.com/guilhermegouw/cdd@latest
```

## Usage

```bash
cdd
```

## Development

```bash
# Build
task build

# Run tests
task test

# Run linter
task lint

# Run all checks
task check
```

## License

MIT
