package tools

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"charm.land/fantasy"
)

// Tool constants for read operations.
const (
	ReadToolName     = "read"
	MaxReadSize      = 5 * 1024 * 1024 // 5MB
	DefaultReadLimit = 2000
	MaxLineLength    = 2000
)

// ReadParams are the parameters for the read tool.
type ReadParams struct {
	FilePath string `json:"file_path" description:"The absolute path to the file to read"`
	Offset   int    `json:"offset,omitempty" description:"The line number to start reading from (1-based). Defaults to 1."`
	Limit    int    `json:"limit,omitempty" description:"The number of lines to read. Defaults to 2000."`
}

// ReadResponseMetadata provides metadata about the read operation.
type ReadResponseMetadata struct {
	FilePath   string `json:"file_path"`
	LineCount  int    `json:"line_count"`
	TotalLines int    `json:"total_lines"`
	Truncated  bool   `json:"truncated"`
}

const readDescription = `Reads a file from the local filesystem.

Usage:
- The file_path parameter must be an absolute path, not a relative path
- By default, it reads up to 2000 lines starting from the beginning of the file
- You can optionally specify a line offset and limit for long files
- Any lines longer than 2000 characters will be truncated
- Results are returned with line numbers starting at 1
- This tool can only read files, not directories`

// NewReadTool creates a new read tool.
func NewReadTool(workingDir string) fantasy.AgentTool {
	return fantasy.NewAgentTool(
		ReadToolName,
		readDescription,
		func(ctx context.Context, params ReadParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if params.FilePath == "" {
				return fantasy.NewTextErrorResponse("file_path is required"), nil
			}

			// Resolve path
			filePath := ResolvePath(workingDir, params.FilePath)

			// Check if file exists
			fileInfo, err := os.Stat(filePath)
			if err != nil {
				if os.IsNotExist(err) {
					return fantasy.NewTextErrorResponse(fmt.Sprintf("File not found: %s", filePath)), nil
				}
				return fantasy.ToolResponse{}, fmt.Errorf("error accessing file: %w", err)
			}

			// Check if it's a directory
			if fileInfo.IsDir() {
				return fantasy.NewTextErrorResponse(fmt.Sprintf("Path is a directory, not a file: %s", filePath)), nil
			}

			// Check file size
			if fileInfo.Size() > MaxReadSize {
				return fantasy.NewTextErrorResponse(fmt.Sprintf("File is too large (%d bytes). Maximum size is %d bytes",
					fileInfo.Size(), MaxReadSize)), nil
			}

			// Set defaults
			if params.Limit <= 0 {
				params.Limit = DefaultReadLimit
			}
			if params.Offset < 0 {
				params.Offset = 0
			}

			// Read the file content
			content, lineCount, totalLines, err := readTextFile(filePath, params.Offset, params.Limit)
			if err != nil {
				return fantasy.ToolResponse{}, fmt.Errorf("error reading file: %w", err)
			}

			// Validate UTF-8
			if !utf8.ValidString(content) {
				return fantasy.NewTextErrorResponse("File content is not valid UTF-8"), nil
			}

			// Record the read
			RecordFileRead(filePath)

			// Format the output with line numbers
			startLine := params.Offset + 1
			if params.Offset == 0 {
				startLine = 1
			}
			output := addLineNumbers(content, startLine)

			// Add truncation note if needed
			truncated := lineCount < totalLines-params.Offset
			if truncated {
				output += fmt.Sprintf("\n\n(File has %d total lines. Use 'offset' parameter to read beyond line %d)",
					totalLines, params.Offset+lineCount)
			}

			return fantasy.WithResponseMetadata(
				fantasy.NewTextResponse(output),
				ReadResponseMetadata{
					FilePath:   filePath,
					LineCount:  lineCount,
					TotalLines: totalLines,
					Truncated:  truncated,
				},
			), nil
		})
}

func addLineNumbers(content string, startLine int) string {
	if content == "" {
		return ""
	}

	lines := strings.Split(content, "\n")
	result := make([]string, 0, len(lines))

	for i, line := range lines {
		line = strings.TrimSuffix(line, "\r")
		lineNum := i + startLine
		numStr := fmt.Sprintf("%d", lineNum)

		if len(numStr) >= 6 {
			result = append(result, fmt.Sprintf("%s\t%s", numStr, line))
		} else {
			paddedNum := fmt.Sprintf("%6s", numStr)
			result = append(result, fmt.Sprintf("%s\t%s", paddedNum, line))
		}
	}

	return strings.Join(result, "\n")
}

func readTextFile(filePath string, offset, limit int) (content string, lineCount, totalLines int, err error) {
	file, err := os.Open(filePath) //nolint:gosec // G304: File path is validated above
	if err != nil {
		return "", 0, 0, err
	}
	defer file.Close() //nolint:errcheck // Error on close for read-only file is ignorable

	scanner := newLineScanner(file)
	totalLines = 0

	// Skip to offset
	for totalLines < offset && scanner.Scan() {
		totalLines++
	}
	if scanErr := scanner.Err(); scanErr != nil {
		return "", 0, 0, scanErr
	}

	// Reset if we're starting from beginning
	if offset == 0 {
		_, err = file.Seek(0, io.SeekStart)
		if err != nil {
			return "", 0, 0, err
		}
		scanner = newLineScanner(file)
	}

	// Read lines up to limit
	lines := make([]string, 0, limit)
	for scanner.Scan() && len(lines) < limit {
		totalLines++
		lineText := scanner.Text()
		if len(lineText) > MaxLineLength {
			lineText = lineText[:MaxLineLength] + "..."
		}
		lines = append(lines, lineText)
	}

	// Count remaining lines
	for scanner.Scan() {
		totalLines++
	}

	if scanErr := scanner.Err(); scanErr != nil {
		return "", 0, 0, scanErr
	}

	return strings.Join(lines, "\n"), len(lines), totalLines, nil
}

// lineScanner wraps bufio.Scanner with a larger buffer.
type lineScanner struct {
	scanner *bufio.Scanner
}

func newLineScanner(r io.Reader) *lineScanner {
	scanner := bufio.NewScanner(r)
	// Increase buffer size to handle large lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	return &lineScanner{scanner: scanner}
}

func (s *lineScanner) Scan() bool {
	return s.scanner.Scan()
}

func (s *lineScanner) Text() string {
	return s.scanner.Text()
}

func (s *lineScanner) Err() error {
	return s.scanner.Err()
}
