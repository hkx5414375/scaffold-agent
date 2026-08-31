// Package app implements the command-line application boundary.
package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strings"

	"github.com/hkx5414375/scaffold-agent/internal/version"
)

const usage = `Scaffold Agent

Usage:
  scaffold-agent <command> [options]

Commands:
  doctor    Check the local development environment
  version   Print build version information
  help      Show this help
`

// Run executes the command and returns a process exit code.
func Run(args []string, stdout, stderr io.Writer) int {
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
	default:
		_, _ = fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		return 2
	}
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
