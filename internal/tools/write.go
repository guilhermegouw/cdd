package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"charm.land/fantasy"
)

// WriteToolName is the name of the write tool.
const WriteToolName = "write"

// WriteParams are the parameters for the write tool.
type WriteParams struct {
	FilePath string `json:"file_path" description:"The absolute path to the file to write"`
	Content  string `json:"content" description:"The content to write to the file"`
}

// WriteResponseMetadata provides metadata about the write operation.
type WriteResponseMetadata struct {
	FilePath     string `json:"file_path"`
	BytesWritten int    `json:"bytes_written"`
	Created      bool   `json:"created"`
}

const writeDescription = `Writes content to a file on the local filesystem.

Usage:
- The file_path parameter must be an absolute path, not a relative path
- This tool will overwrite the existing file if there is one at the provided path
- If this is an existing file, you MUST use the Read tool first to read the file's contents
- Parent directories will be created automatically if they don't exist`

// NewWriteTool creates a new write tool.
func NewWriteTool(workingDir string) fantasy.AgentTool {
	return fantasy.NewAgentTool(
		WriteToolName,
		writeDescription,
		func(ctx context.Context, params WriteParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if params.FilePath == "" {
				return fantasy.NewTextErrorResponse("file_path is required"), nil
			}

			if params.Content == "" {
				return fantasy.NewTextErrorResponse("content is required"), nil
			}

			// Resolve path
			filePath := ResolvePath(workingDir, params.FilePath)

			// Check if file already exists
			fileInfo, err := os.Stat(filePath)
			created := os.IsNotExist(err)

			if err == nil {
				// File exists
				if fileInfo.IsDir() {
					return fantasy.NewTextErrorResponse(fmt.Sprintf("Path is a directory, not a file: %s", filePath)), nil
				}

				// Check if file has been modified since last read
				modTime := fileInfo.ModTime()
				lastRead := GetLastReadTime(filePath)
				if !lastRead.IsZero() && modTime.After(lastRead) {
					return fantasy.NewTextErrorResponse(fmt.Sprintf(
						"File %s has been modified since it was last read.\n"+
							"Last modification: %s\n"+
							"Last read: %s\n\n"+
							"Please read the file again before modifying it.",
						filePath, modTime.Format(time.RFC3339), lastRead.Format(time.RFC3339))), nil
				}

				// Check for no-op writes
				oldContent, readErr := os.ReadFile(filePath) //nolint:gosec // G304: File path is validated above
				if readErr == nil && string(oldContent) == params.Content {
					return fantasy.NewTextErrorResponse(fmt.Sprintf(
						"File %s already contains the exact content. No changes made.", filePath)), nil
				}
			} else if !os.IsNotExist(err) {
				return fantasy.ToolResponse{}, fmt.Errorf("error checking file: %w", err)
			}

			// Create parent directories if needed
			dir := filepath.Dir(filePath)
			if err := os.MkdirAll(dir, 0o755); err != nil { //nolint:gosec // G301: Standard dir permissions for user files
				return fantasy.ToolResponse{}, fmt.Errorf("error creating directory: %w", err)
			}

			// Write the file
			if err := os.WriteFile(filePath, []byte(params.Content), 0o644); err != nil { //nolint:gosec // G306: Standard file permissions for user files
				return fantasy.ToolResponse{}, fmt.Errorf("error writing file: %w", err)
			}

			// Record the write and read (since we now know the content)
			RecordFileWrite(filePath)
			RecordFileRead(filePath)

			action := "written"
			if created {
				action = "created"
			}

			return fantasy.WithResponseMetadata(
				fantasy.NewTextResponse(fmt.Sprintf("File successfully %s: %s", action, filePath)),
				WriteResponseMetadata{
					FilePath:     filePath,
					BytesWritten: len(params.Content),
					Created:      created,
				},
			), nil
		})
}

// generateSimpleDiff generates a simple diff summary.
func generateSimpleDiff(oldContent, newContent string) (additions, removals int) {
	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	oldSet := make(map[string]int)
	for _, line := range oldLines {
		oldSet[line]++
	}

	for _, line := range newLines {
		if oldSet[line] > 0 {
			oldSet[line]--
		} else {
			additions++
		}
	}

	for _, count := range oldSet {
		removals += count
	}

	return additions, removals
}
