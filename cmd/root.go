// Package cmd provides the CLI commands for CDD.
package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/guilhermegouw/cdd/internal/agent"
	"github.com/guilhermegouw/cdd/internal/config"
	"github.com/guilhermegouw/cdd/internal/provider"
	"github.com/guilhermegouw/cdd/internal/tools"
	"github.com/guilhermegouw/cdd/internal/tui"
)

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

	cmd.AddCommand(newVersionCmd())

	return cmd
}

// runTUI launches the terminal user interface.
func runTUI(_ *cobra.Command, _ []string) error {
	// Check if this is first run.
	isFirstRun := config.IsFirstRun()

	// Load providers from catwalk (for the wizard).
	cfg := config.NewConfig()

	// Try to load providers even if config doesn't exist.
	providers, err := config.LoadProviders(cfg)
	if err != nil {
		// If we can't load providers, show an error.
		fmt.Fprintf(os.Stderr, "Warning: Failed to load providers: %v\n", err)
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

	return tui.Run(providers, isFirstRun, ag)
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
