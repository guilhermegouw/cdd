// Package cmd provides the CLI commands for CDD.
package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"charm.land/fantasy"
	"github.com/adrg/xdg"
	"github.com/spf13/cobra"

	"github.com/guilhermegouw/cdd/internal/agent"
	"github.com/guilhermegouw/cdd/internal/config"
	"github.com/guilhermegouw/cdd/internal/debug"
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

	cmd.Flags().Bool("debug", false, "Enable debug logging to ~/.cdd/debug.log")
	cmd.AddCommand(newVersionCmd())

	return cmd
}

func runTUI(cmd *cobra.Command, _ []string) error {
	// Enable debug logging if requested.
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

	// Load configuration.
	isFirstRun := config.IsFirstRun()
	cfg, err := config.Load()
	if err != nil {
		cfg = config.NewConfig()
	}

	// Load providers.
	providers := cfg.KnownProviders()
	if len(providers) == 0 {
		providers, err = config.LoadProviders(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to load providers: %v\n", err)
		}
	}

	// Create agent if not first run.
	var ag *agent.DefaultAgent
	if !isFirstRun {
		ag, err = createAgent(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to create agent: %v\n", err)
		}
	}

	// Define agent factory for TUI to reload agent on config changes.
	agentFactory := func() (*agent.DefaultAgent, error) {
		newCfg, err := config.Load()
		if err != nil {
			return nil, fmt.Errorf("loading config: %w", err)
		}
		return createAgent(newCfg)
	}

	// Define model factory for rebuilding model with fresh tokens.
	// This allows swapping the model without creating a new agent, preserving session history.
	modelFactory := func() (fantasy.LanguageModel, error) {
		newCfg, err := config.Load()
		if err != nil {
			return nil, fmt.Errorf("loading config: %w", err)
		}
		return createModel(newCfg)
	}

	return tui.Run(providers, isFirstRun, ag, agentFactory, modelFactory)
}

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
		SystemPrompt: agent.DefaultSystemPrompt,
	}

	return agent.New(agentCfg), nil
}

// createModel builds just the model from config with fresh tokens.
// Used for swapping models after token refresh without creating a new agent.
func createModel(cfg *config.Config) (fantasy.LanguageModel, error) {
	ctx := context.Background()

	// Build models from configuration (this reloads tokens from disk).
	builder := provider.NewBuilder(cfg)
	largeModel, _, err := builder.BuildModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("building models: %w", err)
	}

	return largeModel.Model, nil
}

// Execute runs the root command.
func Execute() error {
	return newRootCmd().Execute()
}
