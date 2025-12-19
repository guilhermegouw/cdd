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

// EditToolName is the name of the edit tool.
const EditToolName = "edit"

// EditParams are the parameters for the edit tool.
type EditParams struct {
	FilePath   string `json:"file_path" description:"The absolute path to the file to modify"`
	OldString  string `json:"old_string" description:"The text to replace"`
	NewString  string `json:"new_string" description:"The text to replace it with (must be different from old_string)"`
	ReplaceAll bool   `json:"replace_all,omitempty" description:"Replace all occurrences of old_string (default false)"`
}

// EditResponseMetadata provides metadata about the edit operation.
type EditResponseMetadata struct {
	FilePath     string `json:"file_path"`
	Replacements int    `json:"replacements"`
	Additions    int    `json:"additions"`
	Removals     int    `json:"removals"`
}

const editDescription = `Performs exact string replacements in files.

Usage:
- You must use the Read tool at least once before editing a file
- The edit will FAIL if old_string is not unique in the file (unless replace_all=true)
- Use replace_all for replacing and renaming strings across the file
- When old_string is empty, creates a new file with new_string as content
- When new_string is empty, deletes the old_string from the file`

// NewEditTool creates a new edit tool.
func NewEditTool(workingDir string) fantasy.AgentTool {
	return fantasy.NewAgentTool(
		EditToolName,
		editDescription,
		func(ctx context.Context, params EditParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if params.FilePath == "" {
				return fantasy.NewTextErrorResponse("file_path is required"), nil
			}

			// Resolve path
			filePath := ResolvePath(workingDir, params.FilePath)

			// Handle different edit modes
			if params.OldString == "" {
				// Create new file mode
				return createNewFile(filePath, params.NewString)
			}

			if params.NewString == "" {
				// Delete content mode
				return deleteContent(filePath, params.OldString, params.ReplaceAll)
			}

			// Replace content mode
			return replaceContent(filePath, params.OldString, params.NewString, params.ReplaceAll)
		})
}

func createNewFile(filePath, content string) (fantasy.ToolResponse, error) {
	if content == "" {
		return fantasy.NewTextErrorResponse("new_string is required when creating a new file"), nil
	}

	fileInfo, err := os.Stat(filePath)
	if err == nil {
		if fileInfo.IsDir() {
			return fantasy.NewTextErrorResponse(fmt.Sprintf("path is a directory, not a file: %s", filePath)), nil
		}
		return fantasy.NewTextErrorResponse(fmt.Sprintf("file already exists: %s", filePath)), nil
	} else if !os.IsNotExist(err) {
		return fantasy.ToolResponse{}, fmt.Errorf("failed to access file: %w", err)
	}

	// Create parent directories
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil { //nolint:gosec // G301: Standard dir permissions for user files
		return fantasy.ToolResponse{}, fmt.Errorf("failed to create parent directories: %w", err)
	}

	// Write file
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil { //nolint:gosec // G306: Standard file permissions for user files
		return fantasy.ToolResponse{}, fmt.Errorf("failed to write file: %w", err)
	}

	RecordFileWrite(filePath)
	RecordFileRead(filePath)

	return fantasy.WithResponseMetadata(
		fantasy.NewTextResponse(fmt.Sprintf("File created: %s", filePath)),
		EditResponseMetadata{
			FilePath:     filePath,
			Replacements: 0,
			Additions:    len(strings.Split(content, "\n")),
			Removals:     0,
		},
	), nil
}

func deleteContent(filePath, oldString string, replaceAll bool) (fantasy.ToolResponse, error) {
	// Check file exists
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fantasy.NewTextErrorResponse(fmt.Sprintf("file not found: %s", filePath)), nil
		}
		return fantasy.ToolResponse{}, fmt.Errorf("failed to access file: %w", err)
	}

	if fileInfo.IsDir() {
		return fantasy.NewTextErrorResponse(fmt.Sprintf("path is a directory, not a file: %s", filePath)), nil
	}

	// Check if file was read first
	if GetLastReadTime(filePath).IsZero() {
		return fantasy.NewTextErrorResponse("you must read the file before editing it. Use the Read tool first"), nil
	}

	// Check modification time
	modTime := fileInfo.ModTime()
	lastRead := GetLastReadTime(filePath)
	if modTime.After(lastRead) {
		return fantasy.NewTextErrorResponse(fmt.Sprintf(
			"file %s has been modified since it was last read (mod time: %s, last read: %s)",
			filePath, modTime.Format(time.RFC3339), lastRead.Format(time.RFC3339))), nil
	}

	// Read content
	content, err := os.ReadFile(filePath) //nolint:gosec // G304: File path is validated above
	if err != nil {
		return fantasy.ToolResponse{}, fmt.Errorf("failed to read file: %w", err)
	}

	oldContent := normalizeLineEndings(string(content))
	var newContent string
	var deletionCount int

	if replaceAll {
		newContent = strings.ReplaceAll(oldContent, oldString, "")
		deletionCount = strings.Count(oldContent, oldString)
		if deletionCount == 0 {
			return fantasy.NewTextErrorResponse("old_string not found in file. Make sure it matches exactly, including whitespace and line breaks"), nil
		}
	} else {
		index := strings.Index(oldContent, oldString)
		if index == -1 {
			return fantasy.NewTextErrorResponse("old_string not found in file. Make sure it matches exactly, including whitespace and line breaks"), nil
		}

		lastIndex := strings.LastIndex(oldContent, oldString)
		if index != lastIndex {
			return fantasy.NewTextErrorResponse("old_string appears multiple times in the file. Please provide more context to ensure a unique match, or set replace_all to true"), nil
		}

		newContent = oldContent[:index] + oldContent[index+len(oldString):]
		deletionCount = 1
	}

	// Write file
	if err := os.WriteFile(filePath, []byte(newContent), 0o644); err != nil { //nolint:gosec // G306: Standard file permissions for user files
		return fantasy.ToolResponse{}, fmt.Errorf("failed to write file: %w", err)
	}

	RecordFileWrite(filePath)
	RecordFileRead(filePath)

	additions, removals := generateSimpleDiff(oldContent, newContent)

	return fantasy.WithResponseMetadata(
		fantasy.NewTextResponse(fmt.Sprintf("Content deleted from file: %s (%d occurrence(s))", filePath, deletionCount)),
		EditResponseMetadata{
			FilePath:     filePath,
			Replacements: deletionCount,
			Additions:    additions,
			Removals:     removals,
		},
	), nil
}

func replaceContent(filePath, oldString, newString string, replaceAll bool) (fantasy.ToolResponse, error) {
	// Check file exists
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fantasy.NewTextErrorResponse(fmt.Sprintf("file not found: %s", filePath)), nil
		}
		return fantasy.ToolResponse{}, fmt.Errorf("failed to access file: %w", err)
	}

	if fileInfo.IsDir() {
		return fantasy.NewTextErrorResponse(fmt.Sprintf("path is a directory, not a file: %s", filePath)), nil
	}

	// Check if file was read first
	if GetLastReadTime(filePath).IsZero() {
		return fantasy.NewTextErrorResponse("you must read the file before editing it. Use the Read tool first"), nil
	}

	// Check modification time
	modTime := fileInfo.ModTime()
	lastRead := GetLastReadTime(filePath)
	if modTime.After(lastRead) {
		return fantasy.NewTextErrorResponse(fmt.Sprintf(
			"file %s has been modified since it was last read (mod time: %s, last read: %s)",
			filePath, modTime.Format(time.RFC3339), lastRead.Format(time.RFC3339))), nil
	}

	// Read content
	content, err := os.ReadFile(filePath) //nolint:gosec // G304: File path is validated above
	if err != nil {
		return fantasy.ToolResponse{}, fmt.Errorf("failed to read file: %w", err)
	}

	oldContent := normalizeLineEndings(string(content))
	var newContent string
	var replacementCount int

	if replaceAll {
		newContent = strings.ReplaceAll(oldContent, oldString, newString)
		replacementCount = strings.Count(oldContent, oldString)
		if replacementCount == 0 {
			return fantasy.NewTextErrorResponse("old_string not found in file. Make sure it matches exactly, including whitespace and line breaks"), nil
		}
	} else {
		index := strings.Index(oldContent, oldString)
		if index == -1 {
			return fantasy.NewTextErrorResponse("old_string not found in file. Make sure it matches exactly, including whitespace and line breaks"), nil
		}

		lastIndex := strings.LastIndex(oldContent, oldString)
		if index != lastIndex {
			return fantasy.NewTextErrorResponse("old_string appears multiple times in the file. Please provide more context to ensure a unique match, or set replace_all to true"), nil
		}

		newContent = oldContent[:index] + newString + oldContent[index+len(oldString):]
		replacementCount = 1
	}

	if oldContent == newContent {
		return fantasy.NewTextErrorResponse("new content is the same as old content. No changes made."), nil
	}

	// Write file
	if err := os.WriteFile(filePath, []byte(newContent), 0o644); err != nil { //nolint:gosec // G306: Standard file permissions for user files
		return fantasy.ToolResponse{}, fmt.Errorf("failed to write file: %w", err)
	}

	RecordFileWrite(filePath)
	RecordFileRead(filePath)

	additions, removals := generateSimpleDiff(oldContent, newContent)

	return fantasy.WithResponseMetadata(
		fantasy.NewTextResponse(fmt.Sprintf("Content replaced in file: %s (%d replacement(s))", filePath, replacementCount)),
		EditResponseMetadata{
			FilePath:     filePath,
			Replacements: replacementCount,
			Additions:    additions,
			Removals:     removals,
		},
	), nil
}

func normalizeLineEndings(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}
