package tools

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"charm.land/fantasy"
)

// Tool constants for grep operations.
const (
	GrepToolName        = "grep"
	maxGrepContentWidth = 500
	grepLimit           = 100
)

// GrepParams are the parameters for the grep tool.
type GrepParams struct {
	Pattern     string `json:"pattern" description:"The regex pattern to search for in file contents"`
	Path        string `json:"path,omitempty" description:"The directory to search in. Defaults to the current working directory."`
	Include     string `json:"include,omitempty" description:"File pattern to include in the search (e.g., '*.go', '*.{ts,tsx}')"`
	LiteralText bool   `json:"literal_text,omitempty" description:"If true, the pattern will be treated as literal text. Default is false."`
}

// GrepResponseMetadata provides metadata about the grep operation.
type GrepResponseMetadata struct {
	NumberOfMatches int  `json:"number_of_matches"`
	Truncated       bool `json:"truncated"`
}

const grepDescription = `A powerful search tool for searching file contents.

Usage:
- Supports full regex syntax (e.g., "log.*Error", "function\s+\w+")
- Filter files with the include parameter (e.g., "*.js", "*.{ts,tsx}")
- Use literal_text=true to search for exact text without regex interpretation
- Results are sorted by file modification time (most recent first)
- Results are limited to 100 matches by default`

// grepMatch represents a single grep match.
type grepMatch struct {
	path     string
	lineText string
	modTime  int64
	lineNum  int
	charNum  int
}

// regexCache provides thread-safe caching of compiled regex patterns.
var (
	regexCacheMap   = make(map[string]*regexp.Regexp)
	regexCacheMutex sync.RWMutex
)

func getCachedRegex(pattern string) (*regexp.Regexp, error) {
	regexCacheMutex.RLock()
	if regex, exists := regexCacheMap[pattern]; exists {
		regexCacheMutex.RUnlock()
		return regex, nil
	}
	regexCacheMutex.RUnlock()

	regexCacheMutex.Lock()
	defer regexCacheMutex.Unlock()

	// Double-check
	if regex, exists := regexCacheMap[pattern]; exists {
		return regex, nil
	}

	regex, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	regexCacheMap[pattern] = regex
	return regex, nil
}

// NewGrepTool creates a new grep tool.
func NewGrepTool(workingDir string) fantasy.AgentTool {
	return fantasy.NewAgentTool(
		GrepToolName,
		grepDescription,
		func(ctx context.Context, params GrepParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if params.Pattern == "" {
				return fantasy.NewTextErrorResponse("pattern is required"), nil
			}

			// Escape pattern if literal
			searchPattern := params.Pattern
			if params.LiteralText {
				searchPattern = regexp.QuoteMeta(params.Pattern)
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

			matches, truncated, err := searchFiles(ctx, searchPattern, searchPath, params.Include, grepLimit)
			if err != nil {
				return fantasy.NewTextErrorResponse(fmt.Sprintf("Error searching files: %v", err)), nil
			}

			var output strings.Builder
			if len(matches) == 0 {
				output.WriteString("No matches found")
			} else {
				fmt.Fprintf(&output, "Found %d matches\n", len(matches))

				currentFile := ""
				for _, match := range matches {
					if currentFile != match.path {
						if currentFile != "" {
							output.WriteString("\n")
						}
						currentFile = match.path
						fmt.Fprintf(&output, "%s:\n", filepath.ToSlash(match.path))
					}

					lineText := match.lineText
					if len(lineText) > maxGrepContentWidth {
						lineText = lineText[:maxGrepContentWidth] + "..."
					}

					if match.charNum > 0 {
						fmt.Fprintf(&output, "  Line %d, Char %d: %s\n", match.lineNum, match.charNum, lineText)
					} else {
						fmt.Fprintf(&output, "  Line %d: %s\n", match.lineNum, lineText)
					}
				}

				if truncated {
					output.WriteString("\n(Results are truncated. Consider using a more specific path or pattern.)")
				}
			}

			return fantasy.WithResponseMetadata(
				fantasy.NewTextResponse(output.String()),
				GrepResponseMetadata{
					NumberOfMatches: len(matches),
					Truncated:       truncated,
				},
			), nil
		})
}

//nolint:gocyclo // Complex file walking and pattern matching logic
func searchFiles(ctx context.Context, pattern, rootPath, include string, limit int) ([]grepMatch, bool, error) {
	regex, err := getCachedRegex(pattern)
	if err != nil {
		return nil, false, fmt.Errorf("invalid regex pattern: %w", err)
	}

	var includePattern *regexp.Regexp
	if include != "" {
		regexPattern := globToRegex(include)
		includePattern, err = getCachedRegex(regexPattern)
		if err != nil {
			return nil, false, fmt.Errorf("invalid include pattern: %w", err)
		}
	}

	var matches []grepMatch

	err = filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err != nil {
			return nil //nolint:nilerr // Skip inaccessible files, continue walking
		}

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

		// Check include pattern
		if includePattern != nil && !includePattern.MatchString(path) {
			return nil
		}

		// Only search text files
		if !isTextFile(path) {
			return nil
		}

		match, lineNum, charNum, lineText, err := fileContainsPattern(path, regex)
		if err != nil || !match {
			return nil //nolint:nilerr // Skip files with read errors, continue walking
		}

		matches = append(matches, grepMatch{
			path:     path,
			modTime:  info.ModTime().UnixNano(),
			lineNum:  lineNum,
			charNum:  charNum,
			lineText: lineText,
		})

		if len(matches) >= limit*2 {
			return filepath.SkipAll
		}

		return nil
	})

	if err != nil && !errors.Is(err, filepath.SkipAll) {
		return nil, false, err
	}

	// Sort by modification time (most recent first)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].modTime > matches[j].modTime
	})

	truncated := len(matches) > limit
	if truncated {
		matches = matches[:limit]
	}

	return matches, truncated, nil
}

func fileContainsPattern(filePath string, pattern *regexp.Regexp) (found bool, lineNum, charNum int, lineText string, err error) {
	file, err := os.Open(filePath) //nolint:gosec // G304: File path comes from directory walk
	if err != nil {
		return false, 0, 0, "", err
	}
	defer file.Close() //nolint:errcheck // Error on close for read-only file is ignorable

	scanner := bufio.NewScanner(file)
	// Increase buffer size for large lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	lineNum = 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if loc := pattern.FindStringIndex(line); loc != nil {
			charNum = loc[0] + 1 // 1-based
			return true, lineNum, charNum, line, nil
		}
	}

	return false, 0, 0, "", scanner.Err()
}

func isTextFile(filePath string) bool {
	file, err := os.Open(filePath) //nolint:gosec // G304: File path comes from directory walk
	if err != nil {
		return false
	}
	defer file.Close() //nolint:errcheck // Error on close for read-only file is ignorable

	// Read first 512 bytes for MIME type detection
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && !errors.Is(err, io.EOF) {
		return false
	}

	contentType := http.DetectContentType(buffer[:n])

	return strings.HasPrefix(contentType, "text/") ||
		contentType == "application/json" ||
		contentType == "application/xml" ||
		contentType == "application/javascript" ||
		contentType == "application/x-sh"
}

func globToRegex(glob string) string {
	regexPattern := regexp.QuoteMeta(glob)
	regexPattern = strings.ReplaceAll(regexPattern, `\*`, ".*")
	regexPattern = strings.ReplaceAll(regexPattern, `\?`, ".")

	// Handle brace expansion {a,b,c}
	braceRegex := regexp.MustCompile(`\\\{([^}]+)\\\}`)
	regexPattern = braceRegex.ReplaceAllStringFunc(regexPattern, func(match string) string {
		// Remove escaped braces and split
		inner := match[2 : len(match)-2]
		parts := strings.Split(inner, ",")
		// Unescape each part
		for i, p := range parts {
			parts[i] = strings.ReplaceAll(p, `\\`, `\`)
		}
		return "(" + strings.Join(parts, "|") + ")"
	})

	return regexPattern
}
