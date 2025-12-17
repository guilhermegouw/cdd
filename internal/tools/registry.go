package tools

import (
	"charm.land/fantasy"
)

// ToolMetadata provides additional information about a tool.
type ToolMetadata struct {
	Name        string
	Category    string
	Description string
	Safe        bool // Safe tools don't modify files or execute commands
}

// Registry manages agent tools.
type Registry struct {
	tools    map[string]fantasy.AgentTool
	metadata map[string]ToolMetadata
}

// NewRegistry creates a new tool registry.
func NewRegistry() *Registry {
	return &Registry{
		tools:    make(map[string]fantasy.AgentTool),
		metadata: make(map[string]ToolMetadata),
	}
}

// Register adds a tool to the registry.
func (r *Registry) Register(tool fantasy.AgentTool, meta ToolMetadata) {
	r.tools[meta.Name] = tool
	r.metadata[meta.Name] = meta
}

// Get returns a tool by name.
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

// Filter returns tools by name.
func (r *Registry) Filter(names []string) []fantasy.AgentTool {
	tools := make([]fantasy.AgentTool, 0, len(names))
	for _, name := range names {
		if tool, ok := r.tools[name]; ok {
			tools = append(tools, tool)
		}
	}
	return tools
}

// SafeTools returns only tools marked as safe.
func (r *Registry) SafeTools() []fantasy.AgentTool {
	tools := make([]fantasy.AgentTool, 0)
	for name, tool := range r.tools {
		if meta, ok := r.metadata[name]; ok && meta.Safe {
			tools = append(tools, tool)
		}
	}
	return tools
}

// Metadata returns metadata for a tool.
func (r *Registry) Metadata(name string) (ToolMetadata, bool) {
	meta, ok := r.metadata[name]
	return meta, ok
}

// Names returns all tool names.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// DefaultRegistry creates a registry with all default tools.
func DefaultRegistry(workingDir string) *Registry {
	r := NewRegistry()

	// Register read-only tools
	r.Register(NewReadTool(workingDir), ToolMetadata{
		Name:        ReadToolName,
		Category:    "file",
		Description: "Read file contents with line numbers",
		Safe:        true,
	})

	r.Register(NewGlobTool(workingDir), ToolMetadata{
		Name:        GlobToolName,
		Category:    "file",
		Description: "Find files by pattern",
		Safe:        true,
	})

	r.Register(NewGrepTool(workingDir), ToolMetadata{
		Name:        GrepToolName,
		Category:    "file",
		Description: "Search file contents",
		Safe:        true,
	})

	// Register write tools
	r.Register(NewWriteTool(workingDir), ToolMetadata{
		Name:        WriteToolName,
		Category:    "file",
		Description: "Write or create files",
		Safe:        false,
	})

	r.Register(NewEditTool(workingDir), ToolMetadata{
		Name:        EditToolName,
		Category:    "file",
		Description: "Edit file contents",
		Safe:        false,
	})

	r.Register(NewBashTool(workingDir), ToolMetadata{
		Name:        BashToolName,
		Category:    "system",
		Description: "Execute shell commands",
		Safe:        false,
	})

	return r
}
