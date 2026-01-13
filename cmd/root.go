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
	"github.com/guilhermegouw/cdd/internal/db"
	"github.com/guilhermegouw/cdd/internal/debug"
	"github.com/guilhermegouw/cdd/internal/message"
	"github.com/guilhermegouw/cdd/internal/provider"
	"github.com/guilhermegouw/cdd/internal/pubsub"
	"github.com/guilhermegouw/cdd/internal/session"
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
	cmd.AddCommand(newStatusCmd())
	cmd.AddCommand(newProvidersCmd())

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

	// Create the pub/sub hub for event distribution.
	hub := pubsub.NewHub()
	defer hub.Shutdown()

	// Create agent if not first run.
	var ag *agent.DefaultAgent
	var modelName string
	var sessionSvc *session.Service
	if !isFirstRun {
		ag, modelName, sessionSvc, err = createAgent(cfg, hub)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to create agent: %v\n", err)
		}
	}

	// Define agent factory for TUI to reload agent on config changes.
	agentFactory := func() (*agent.DefaultAgent, *session.Service, error) {
		newCfg, loadErr := config.Load()
		if loadErr != nil {
			return nil, nil, fmt.Errorf("loading config: %w", loadErr)
		}
		newAgent, _, newSessionSvc, createErr := createAgent(newCfg, hub)
		return newAgent, newSessionSvc, createErr
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

	return tui.Run(cfg, providers, isFirstRun, ag, agentFactory, modelFactory, hub, modelName, sessionSvc)
}

func createAgent(cfg *config.Config, hub *pubsub.Hub) (*agent.DefaultAgent, string, *session.Service, error) {
	ctx := context.Background()

	// Initialize database for persistent sessions first (independent of model building).
	var sessions agent.Sessions
	var sessionSvc *session.Service
	dbPath := filepath.Join(cfg.DataDir(), "cdd.db")
	database, dbErr := db.Open(dbPath)
	if dbErr != nil {
		// Fall back to in-memory sessions if database unavailable.
		debug.Log("Database unavailable, using in-memory sessions: %v", dbErr)
		sessions = nil // Will use default in-memory store
	} else {
		// Create persistent session store.
		sessionStore := session.NewSQLiteStore(database.Conn())
		sessionSvc = session.NewService(sessionStore, hub.Session)

		messageStore := message.NewSQLiteStore(database.Conn())
		messageSvc := message.NewService(messageStore, hub.Session)

		sessions = agent.NewPersistentSessionStore(sessionSvc, messageSvc)
		debug.Log("Using persistent sessions: %s", dbPath)
	}

	// Build models from configuration.
	builder := provider.NewBuilder(cfg)
	largeModel, _, err := builder.BuildModels(ctx)
	if err != nil {
		return nil, "", nil, fmt.Errorf("building models: %w", err)
	}

	// Get working directory.
	cwd, err := os.Getwd()
	if err != nil {
		return nil, "", nil, fmt.Errorf("getting working directory: %w", err)
	}

	// Create todo store and tools registry.
	todoStore := tools.NewTodoStore()
	registry := tools.NewDefaultRegistry(tools.RegistryConfig{
		WorkingDir: cwd,
		Hub:        hub,
		TodoStore:  todoStore,
	})

	// Create agent configuration.
	agentCfg := agent.Config{
		Model:        largeModel.Model,
		Tools:        registry.All(),
		SystemPrompt: agent.DefaultSystemPrompt,
		Hub:          hub,
		Sessions:     sessions,
	}

	// Get model name for display
	modelName := largeModel.CatwalkCfg.Name
	if modelName == "" {
		modelName = largeModel.CatwalkCfg.ID
	}

	return agent.New(agentCfg), modelName, sessionSvc, nil
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
