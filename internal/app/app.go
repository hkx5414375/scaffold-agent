// Package app implements the command-line application boundary.
package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strings"

	"github.com/hkx5414375/scaffold-agent/internal/engine"
	"github.com/hkx5414375/scaffold-agent/internal/mcp"
	"github.com/hkx5414375/scaffold-agent/internal/result"
	"github.com/hkx5414375/scaffold-agent/internal/version"
)

const usage = `Scaffold Agent

Usage:
  scaffold-agent <command> [options]

Commands:
  query     Return compact Engine or managed-project facts
  validate  Validate one project Blueprint
  plan      Build and store an immutable project plan
  preview   Preview a stored plan and obtain its apply token
  apply     Apply a previewed plan transactionally
  verify    Verify every Engine-managed file
  result    Read one page from a stored result
  rollback  Restore one fully applied transaction
  recover   Restore one interrupted transaction
  mcp       Run the newline-delimited JSON-RPC STDIO server
  doctor    Check the local development environment
  version   Print build version information
  help      Show this help
`

// Run executes the command and returns a process exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	return RunIO(args, strings.NewReader(""), stdout, stderr)
}

// RunIO executes the application with an explicit input stream for MCP STDIO.
func RunIO(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = io.WriteString(stdout, usage)
		return 0
	}

	switch args[0] {
	case "help", "--help", "-h":
		_, _ = io.WriteString(stdout, usage)
		return 0
	case "version", "--version", "-v":
		return runVersion(args[1:], stdout, stderr)
	case "doctor":
		return runDoctor(args[1:], stdout, stderr)
	case "query", "validate", "plan", "preview", "apply", "verify", "result", "rollback", "recover":
		return runEngineCommand(args[0], args[1:], stdout, stderr)
	case "mcp":
		if len(args) != 1 {
			_, _ = fmt.Fprintln(stderr, "mcp does not accept command arguments")
			return 2
		}
		current := version.Current()
		server := mcp.New(engine.New(current.Version), current.Version)
		if err := server.Serve(context.Background(), stdin, stdout); err != nil {
			_, _ = fmt.Fprintf(stderr, "run MCP server: %v\n", err)
			return 1
		}
		return 0
	default:
		_, _ = fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		return 2
	}
}

func writeEnvelope(stdout, stderr io.Writer, envelope result.Envelope) int {
	if code := writeJSON(stdout, stderr, envelope); code != 0 {
		return code
	}
	if envelope.Status == result.StatusError {
		return 1
	}
	return 0
}

func runVersion(args []string, stdout, stderr io.Writer) int {
	jsonOutput, err := parseJSONFlag(args)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}

	info := version.Current()
	if jsonOutput {
		return writeJSON(stdout, stderr, info)
	}
	_, _ = fmt.Fprintf(stdout, "scaffold-agent %s (%s, %s)\n", info.Version, info.Commit, info.BuildDate)
	return 0
}

type doctorResult struct {
	Status string        `json:"status"`
	Checks []doctorCheck `json:"checks"`
}

type doctorCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Details string `json:"details"`
}

func runDoctor(args []string, stdout, stderr io.Writer) int {
	jsonOutput, err := parseJSONFlag(args)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}

	checks := []doctorCheck{
		{Name: "go", Status: "ok", Details: runtime.Version()},
		{Name: "platform", Status: "ok", Details: runtime.GOOS + "/" + runtime.GOARCH},
		commandCheck("git"),
	}
	status := "ok"
	for _, check := range checks {
		if check.Status != "ok" {
			status = "warning"
			break
		}
	}
	result := doctorResult{Status: status, Checks: checks}

	if jsonOutput {
		return writeJSON(stdout, stderr, result)
	}
	for _, check := range checks {
		_, _ = fmt.Fprintf(stdout, "%-10s %-8s %s\n", check.Name, check.Status, check.Details)
	}
	return 0
}

func commandCheck(name string) doctorCheck {
	path, err := exec.LookPath(name)
	if err != nil {
		return doctorCheck{Name: name, Status: "missing", Details: "not found in PATH"}
	}
	return doctorCheck{Name: name, Status: "ok", Details: path}
}

func parseJSONFlag(args []string) (bool, error) {
	if len(args) == 0 {
		return false, nil
	}
	if len(args) == 1 && args[0] == "--json" {
		return true, nil
	}
	return false, errors.New("only --json is supported for this command")
}

func writeJSON(stdout, stderr io.Writer, value any) int {
	encoder := json.NewEncoder(stdout)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		_, _ = fmt.Fprintf(stderr, "encode JSON output: %v\n", err)
		return 1
	}
	return 0
}

// Usage returns a normalized usage string for tests and integrations.
func Usage() string {
	return strings.TrimSpace(usage) + "\n"
}
