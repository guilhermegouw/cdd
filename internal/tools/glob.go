package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"charm.land/fantasy"
)

const GlobToolName = "glob"

// GlobParams are the parameters for the glob tool.
type GlobParams struct {
	Pattern string `json:"pattern" description:"The glob pattern to match files against (e.g., '**/*.go', 'src/**/*.ts')"`
	Path    string `json:"path,omitempty" description:"The directory to search in. Defaults to the current working directory."`
}

// GlobResponseMetadata provides metadata about the glob operation.
type GlobResponseMetadata struct {
	NumberOfFiles int  `json:"number_of_files"`
	Truncated     bool `json:"truncated"`
}

const globDescription = `Fast file pattern matching tool that works with any codebase size.

Usage:
- Supports glob patterns like "**/*.js" or "src/**/*.ts"
- Returns matching file paths sorted by modification time
- Use this tool when you need to find files by name patterns
- Results are limited to 100 files by default`

const globLimit = 100

// NewGlobTool creates a new glob tool.
func NewGlobTool(workingDir string) fantasy.AgentTool {
	return fantasy.NewAgentTool(
		GlobToolName,
		globDescription,
		func(ctx context.Context, params GlobParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if params.Pattern == "" {
				return fantasy.NewTextErrorResponse("pattern is required"), nil
			}

			searchPath := params.Path
			if searchPath == "" {
				searchPath = workingDir
			} else {
				searchPath = ResolvePath(workingDir, searchPath)
			}

			// Check if search path exists
			if _, err := os.Stat(searchPath); err != nil {
				if os.IsNotExist(err) {
					return fantasy.NewTextErrorResponse(fmt.Sprintf("Directory not found: %s", searchPath)), nil
				}
				return fantasy.ToolResponse{}, fmt.Errorf("error accessing directory: %w", err)
			}

			files, truncated, err := globFiles(ctx, params.Pattern, searchPath, globLimit)
			if err != nil {
				return fantasy.NewTextErrorResponse(fmt.Sprintf("Error finding files: %v", err)), nil
			}

			var output string
			if len(files) == 0 {
				output = "No files found"
			} else {
				// Normalize paths for output
				for i, f := range files {
					files[i] = filepath.ToSlash(f)
				}
				output = strings.Join(files, "\n")
				if truncated {
					output += "\n\n(Results are truncated. Consider using a more specific path or pattern.)"
				}
			}

			return fantasy.WithResponseMetadata(
				fantasy.NewTextResponse(output),
				GlobResponseMetadata{
					NumberOfFiles: len(files),
					Truncated:     truncated,
				},
			), nil
		})
}

type fileInfo struct {
	path    string
	modTime int64
}

func globFiles(ctx context.Context, pattern, searchPath string, limit int) ([]string, bool, error) {
	var matches []fileInfo

	// Handle ** patterns
	hasDoublestar := strings.Contains(pattern, "**")

	err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err != nil {
			return nil // Skip errors
		}

		// Skip directories themselves, but continue walking into them
		if info.IsDir() {
			// Skip hidden directories
			if info.Name() != "." && strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			// Skip common non-source directories
			switch info.Name() {
			case "node_modules", "vendor", "__pycache__", ".git":
				return filepath.SkipDir
			}
			return nil
		}

		// Skip hidden files
		if strings.HasPrefix(info.Name(), ".") {
			return nil
		}

		// Get relative path for matching
		relPath, err := filepath.Rel(searchPath, path)
		if err != nil {
			return nil
		}

		// Match the pattern
		matched, err := matchGlob(pattern, relPath, hasDoublestar)
		if err != nil || !matched {
			return nil
		}

		matches = append(matches, fileInfo{
			path:    path,
			modTime: info.ModTime().UnixNano(),
		})

		// Early exit if we have enough matches
		if len(matches) >= limit*2 {
			return filepath.SkipAll
		}

		return nil
	})

	if err != nil && err != filepath.SkipAll {
		return nil, false, err
	}

	// Sort by modification time (most recent first)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].modTime > matches[j].modTime
	})

	// Truncate to limit
	truncated := len(matches) > limit
	if truncated {
		matches = matches[:limit]
	}

	// Extract paths
	result := make([]string, len(matches))
	for i, m := range matches {
		result[i] = m.path
	}

	return result, truncated, nil
}

func matchGlob(pattern, path string, hasDoublestar bool) (bool, error) {
	// Normalize separators
	pattern = filepath.ToSlash(pattern)
	path = filepath.ToSlash(path)

	if hasDoublestar {
		return matchDoublestar(pattern, path)
	}

	return filepath.Match(pattern, path)
}

func matchDoublestar(pattern, path string) (bool, error) {
	// Handle simple **/*.ext pattern
	if strings.HasPrefix(pattern, "**/") {
		subPattern := pattern[3:]
		// Try matching against just the filename
		if matched, _ := filepath.Match(subPattern, filepath.Base(path)); matched {
			return true, nil
		}
		// Try matching against each suffix of the path
		parts := strings.Split(path, "/")
		for i := range parts {
			subPath := strings.Join(parts[i:], "/")
			if matched, _ := filepath.Match(subPattern, subPath); matched {
				return true, nil
			}
		}
		return false, nil
	}

	// Handle prefix/**/*.ext pattern
	if idx := strings.Index(pattern, "/**/"); idx != -1 {
		prefix := pattern[:idx]
		suffix := pattern[idx+4:]

		// Check prefix matches
		if !strings.HasPrefix(path, prefix+"/") && path != prefix {
			return false, nil
		}

		// Get the remaining path after prefix
		remaining := strings.TrimPrefix(path, prefix+"/")
		if remaining == path {
			remaining = strings.TrimPrefix(path, prefix)
		}

		// Try matching suffix
		if matched, _ := filepath.Match(suffix, remaining); matched {
			return true, nil
		}
		if matched, _ := filepath.Match(suffix, filepath.Base(remaining)); matched {
			return true, nil
		}

		// Try matching against each suffix of remaining
		parts := strings.Split(remaining, "/")
		for i := range parts {
			subPath := strings.Join(parts[i:], "/")
			if matched, _ := filepath.Match(suffix, subPath); matched {
				return true, nil
			}
		}
		return false, nil
	}

	// Fallback to simple match
	return filepath.Match(pattern, path)
}
