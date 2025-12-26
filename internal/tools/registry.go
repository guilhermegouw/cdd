// Package tools provides tool definitions and registry for the agent.
package tools

import (
	"charm.land/fantasy"

	"github.com/guilhermegouw/cdd/internal/pubsub"
)

// RegistryConfig holds configuration for creating a tool registry.
type RegistryConfig struct {
	WorkingDir string
	Hub        *pubsub.Hub
	TodoStore  *TodoStore
}

// ToolMetadata holds metadata about a tool.
type ToolMetadata struct {
	Name        string
	Category    string
	Description string
	Safe        bool // Safe tools don't modify files or execute commands
}

// Registry manages a collection of agent tools.
type Registry struct {
	tools    map[string]fantasy.AgentTool
	metadata map[string]ToolMetadata
}

// NewRegistry creates a new empty tool registry.
func NewRegistry() *Registry {
	return &Registry{
		tools:    make(map[string]fantasy.AgentTool),
		metadata: make(map[string]ToolMetadata),
	}
}

// Register adds a tool to the registry with its metadata.
func (r *Registry) Register(tool fantasy.AgentTool, meta ToolMetadata) {
	r.tools[meta.Name] = tool
	r.metadata[meta.Name] = meta
}

// Get retrieves a tool by name.
func (r *Registry) Get(name string) (fantasy.AgentTool, bool) {
	tool, ok := r.tools[name]
	return tool, ok
}

// All returns all registered tools.
func (r *Registry) All() []fantasy.AgentTool {
	tools := make([]fantasy.AgentTool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// Filter returns tools matching the given names.
func (r *Registry) Filter(names []string) []fantasy.AgentTool {
	tools := make([]fantasy.AgentTool, 0, len(names))
	for _, name := range names {
		if tool, ok := r.tools[name]; ok {
			tools = append(tools, tool)
		}
	}
	return tools
}

// SafeTools returns all tools marked as safe.
func (r *Registry) SafeTools() []fantasy.AgentTool {
	tools := make([]fantasy.AgentTool, 0)
	for name, tool := range r.tools {
		if meta, ok := r.metadata[name]; ok && meta.Safe {
			tools = append(tools, tool)
		}
	}
	return tools
}

// Metadata returns the metadata for a tool by name.
func (r *Registry) Metadata(name string) (ToolMetadata, bool) {
	meta, ok := r.metadata[name]
	return meta, ok
}

// Names returns the names of all registered tools.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// DefaultRegistry creates a registry with the default set of tools.
//
// Deprecated: Use NewDefaultRegistry instead.
func DefaultRegistry(workingDir string) *Registry {
	return NewDefaultRegistry(RegistryConfig{WorkingDir: workingDir})
}

// NewDefaultRegistry creates a registry with the default set of tools.
func NewDefaultRegistry(cfg RegistryConfig) *Registry {
	r := NewRegistry()

	r.Register(NewReadTool(cfg.WorkingDir), ToolMetadata{
		Name:        ReadToolName,
		Category:    "file",
		Description: "Read file contents with line numbers",
		Safe:        true,
	})

	r.Register(NewGlobTool(cfg.WorkingDir), ToolMetadata{
		Name:        GlobToolName,
		Category:    "file",
		Description: "Find files by pattern",
		Safe:        true,
	})

	r.Register(NewGrepTool(cfg.WorkingDir), ToolMetadata{
		Name:        GrepToolName,
		Category:    "file",
		Description: "Search file contents",
		Safe:        true,
	})

	r.Register(NewWriteTool(cfg.WorkingDir), ToolMetadata{
		Name:        WriteToolName,
		Category:    "file",
		Description: "Write or create files",
		Safe:        false,
	})

	r.Register(NewEditTool(cfg.WorkingDir), ToolMetadata{
		Name:        EditToolName,
		Category:    "file",
		Description: "Edit file contents",
		Safe:        false,
	})

	r.Register(NewBashTool(cfg.WorkingDir), ToolMetadata{
		Name:        BashToolName,
		Category:    "system",
		Description: "Execute shell commands",
		Safe:        false,
	})

	// Register TodoWrite if store is provided
	if cfg.TodoStore != nil {
		r.Register(NewTodoWriteTool(cfg.TodoStore, cfg.Hub), ToolMetadata{
			Name:        TodoWriteToolName,
			Category:    "task",
			Description: "Manage task list for tracking progress",
			Safe:        true,
		})
	}

	return r
}
