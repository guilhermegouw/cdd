package tools

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"charm.land/fantasy"
)

const (
	BashToolName    = "bash"
	MaxOutputLength = 30000
	DefaultTimeout  = 2 * time.Minute
)

// BashParams are the parameters for the bash tool.
type BashParams struct {
	Command     string `json:"command" description:"The command to execute"`
	Description string `json:"description,omitempty" description:"A brief description of what the command does"`
	WorkingDir  string `json:"working_dir,omitempty" description:"The working directory to execute the command in"`
	Timeout     int    `json:"timeout,omitempty" description:"Timeout in milliseconds (max 600000, default 120000)"`
}

// BashResponseMetadata provides metadata about the bash execution.
type BashResponseMetadata struct {
	StartTime        int64  `json:"start_time"`
	EndTime          int64  `json:"end_time"`
	ExitCode         int    `json:"exit_code"`
	Output           string `json:"output"`
	Description      string `json:"description"`
	WorkingDirectory string `json:"working_directory"`
}

// bannedCommands lists commands that should not be executed.
var bannedCommands = []string{
	// Network tools
	"curl", "wget", "ssh", "scp", "nc", "telnet",
	// Browsers
	"chrome", "firefox", "safari",
	// System administration
	"sudo", "su", "doas",
	// Package managers
	"apt", "apt-get", "yum", "dnf", "pacman", "brew",
	// System modification
	"systemctl", "service", "mount", "umount", "fdisk", "mkfs",
	// Network configuration
	"iptables", "ufw", "firewall-cmd", "ifconfig", "ip",
}

const bashDescription = `Executes a shell command with optional timeout.

Usage:
- The command is executed in a shell (bash on Unix, cmd on Windows)
- Commands are subject to safety restrictions
- Output is truncated if it exceeds 30000 characters
- Default timeout is 2 minutes, maximum is 10 minutes
- Use working_dir to specify a different working directory

Banned commands include: network tools, browsers, sudo/su, package managers, and system modification tools.`

// NewBashTool creates a new bash tool.
func NewBashTool(workingDir string) fantasy.AgentTool {
	return fantasy.NewAgentTool(
		BashToolName,
		bashDescription,
		func(ctx context.Context, params BashParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if params.Command == "" {
				return fantasy.NewTextErrorResponse("command is required"), nil
			}

			// Check for banned commands
			cmdLower := strings.ToLower(params.Command)
			for _, banned := range bannedCommands {
				if isBannedCommand(cmdLower, banned) {
					return fantasy.NewTextErrorResponse(fmt.Sprintf("Command '%s' is not allowed for security reasons", banned)), nil
				}
			}

			// Determine working directory
			execWorkingDir := params.WorkingDir
			if execWorkingDir == "" {
				execWorkingDir = workingDir
			} else {
				execWorkingDir = ResolvePath(workingDir, execWorkingDir)
			}

			// Determine timeout
			timeout := DefaultTimeout
			if params.Timeout > 0 {
				timeout = time.Duration(params.Timeout) * time.Millisecond
				if timeout > 10*time.Minute {
					timeout = 10 * time.Minute
				}
			}

			// Create context with timeout
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			startTime := time.Now()

			// Execute command
			var cmd *exec.Cmd
			if runtime.GOOS == "windows" {
				cmd = exec.CommandContext(ctx, "cmd", "/c", params.Command) //nolint:gosec // G204: Command execution is the tool's purpose
			} else {
				cmd = exec.CommandContext(ctx, "bash", "-c", params.Command) //nolint:gosec // G204: Command execution is the tool's purpose
			}
			cmd.Dir = execWorkingDir

			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err := cmd.Run()
			endTime := time.Now()

			exitCode := 0
			if err != nil {
				var exitErr *exec.ExitError
				if errors.As(err, &exitErr) {
					exitCode = exitErr.ExitCode()
				} else if errors.Is(ctx.Err(), context.DeadlineExceeded) {
					return fantasy.NewTextErrorResponse(fmt.Sprintf(
						"Command timed out after %v", timeout)), nil
				} else if errors.Is(ctx.Err(), context.Canceled) {
					return fantasy.NewTextErrorResponse("Command was cancelled"), nil
				}
			}

			// Format output
			output := formatBashOutput(stdout.String(), stderr.String(), exitCode)

			return fantasy.WithResponseMetadata(
				fantasy.NewTextResponse(output),
				BashResponseMetadata{
					StartTime:        startTime.UnixMilli(),
					EndTime:          endTime.UnixMilli(),
					ExitCode:         exitCode,
					Output:           output,
					Description:      params.Description,
					WorkingDirectory: execWorkingDir,
				},
			), nil
		})
}

func isBannedCommand(cmdLine, banned string) bool {
	// Check if command starts with banned command
	if strings.HasPrefix(cmdLine, banned) {
		// Make sure it's a complete word
		if len(cmdLine) == len(banned) ||
			cmdLine[len(banned)] == ' ' ||
			cmdLine[len(banned)] == '\t' {
			return true
		}
	}

	// Check for pipe or semicolon followed by banned command
	for _, sep := range []string{"|", ";", "&&", "||"} {
		parts := strings.Split(cmdLine, sep)
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, banned) {
				if len(part) == len(banned) ||
					part[len(banned)] == ' ' ||
					part[len(banned)] == '\t' {
					return true
				}
			}
		}
	}

	return false
}

func formatBashOutput(stdout, stderr string, exitCode int) string {
	stdout = strings.TrimSpace(stdout)
	stderr = strings.TrimSpace(stderr)

	var output strings.Builder

	if stdout != "" {
		output.WriteString(truncateOutput(stdout))
	}

	if stderr != "" {
		if output.Len() > 0 {
			output.WriteString("\n\n")
		}
		output.WriteString("STDERR:\n")
		output.WriteString(truncateOutput(stderr))
	}

	if exitCode != 0 {
		if output.Len() > 0 {
			output.WriteString("\n\n")
		}
		output.WriteString(fmt.Sprintf("Exit code: %d", exitCode))
	}

	if output.Len() == 0 {
		return "(no output)"
	}

	return output.String()
}

func truncateOutput(content string) string {
	if len(content) <= MaxOutputLength {
		return content
	}

	halfLength := MaxOutputLength / 2
	start := content[:halfLength]
	end := content[len(content)-halfLength:]

	truncatedLines := strings.Count(content[halfLength:len(content)-halfLength], "\n")
	return fmt.Sprintf("%s\n\n... [%d lines truncated] ...\n\n%s", start, truncatedLines, end)
}
