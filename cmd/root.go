// Package cmd provides the CLI commands for CDD.
package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/spf13/cobra"

	"github.com/guilhermegouw/cdd/internal/agent"
	"github.com/guilhermegouw/cdd/internal/config"
	"github.com/guilhermegouw/cdd/internal/debug"
	"github.com/guilhermegouw/cdd/internal/provider"
	"github.com/guilhermegouw/cdd/internal/tools"
	"github.com/guilhermegouw/cdd/internal/tui"
)

var debugMode bool

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cdd",
		Short: "Context-Driven Development CLI",
		Long: `CDD is an AI-powered coding assistant that helps you write,
understand, and improve your code through structured workflows.

It supports multiple phases of development:
  - Socrates: Clarify requirements through dialogue
  - Planner: Design implementation strategy
  - Executor: Write and modify code`,
		RunE: runTUI,
	}

	cmd.Flags().BoolVar(&debugMode, "debug", false, "Enable debug logging to ~/.cdd/debug.log")
	cmd.AddCommand(newVersionCmd())

	return cmd
}

// runTUI launches the terminal user interface.
func runTUI(_ *cobra.Command, _ []string) error {
	// Enable debug logging if requested.
	if debugMode {
		logPath := filepath.Join(xdg.DataHome, "cdd", "debug.log")
		if err := debug.Enable(logPath); err != nil {
			fmt.Printf("Warning: Failed to enable debug logging: %v\n", err)
		} else {
			defer debug.Disable()
			fmt.Printf("Debug: %s\n", logPath)
		}
	}

	// Check if this is first run.
	isFirstRun := config.IsFirstRun()

	// Try to load full config from disk.
	cfg, err := config.Load()
	if err != nil {
		// If config fails to load, create empty config for wizard.
		cfg = config.NewConfig()
	}

	// Get providers for the wizard.
	providers := cfg.KnownProviders()
	if len(providers) == 0 {
		// Try to load providers if not already loaded.
		providers, err = config.LoadProviders(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to load providers: %v\n", err)
		}
	}

	// Try to create the agent if we have a valid configuration.
	var ag *agent.DefaultAgent
	if !isFirstRun {
		ag, err = createAgent(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to create agent: %v\n", err)
			// Continue without agent - wizard will configure it.
		}
	}

	// Create an agent factory that reloads config and creates a new agent.
	// This is called after the wizard completes to create the agent without restarting.
	agentFactory := func() (*agent.DefaultAgent, error) {
		// Reload config from disk (wizard just saved it).
		newCfg, err := config.Load()
		if err != nil {
			return nil, fmt.Errorf("loading config: %w", err)
		}
		return createAgent(newCfg)
	}

	return tui.Run(providers, isFirstRun, ag, agentFactory)
}

// createAgent creates the agent with tools and model from configuration.
func createAgent(cfg *config.Config) (*agent.DefaultAgent, error) {
	ctx := context.Background()

	// Build models from configuration.
	builder := provider.NewBuilder(cfg)
	largeModel, _, err := builder.BuildModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("building models: %w", err)
	}

	// Get working directory.
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getting working directory: %w", err)
	}

	// Create tools registry.
	registry := tools.DefaultRegistry(cwd)

	// Create agent configuration.
	agentCfg := agent.Config{
		Model:        largeModel.Model,
		Tools:        registry.All(),
		SystemPrompt: defaultSystemPrompt,
	}

	return agent.New(agentCfg), nil
}

// defaultSystemPrompt is the main system prompt for the agent.
// Note: The OAuth header "You are Claude Code..." is added separately in the agent
// as a separate content block, as required by Anthropic's OAuth API.
const defaultSystemPrompt = `You are CDD (Context-Driven Development), an AI coding assistant.

You help developers write, understand, and improve code through structured workflows.

When working with code:
1. Read files before modifying them
2. Use appropriate tools for the task
3. Explain your reasoning clearly
4. Ask clarifying questions when requirements are unclear

Available tools allow you to read files, search code, write files, edit code, and execute shell commands.`

// Execute runs the root command.
func Execute() error {
	return newRootCmd().Execute()
}
