package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const agentUpdateUsage = `Usage: setupmjr agent update <claude|codex|copilot>
       setupmjr agent update --all

Update the selected CLI, or every supported CLI found on PATH.
Missing CLIs are skipped with --all. Failures do not stop other updates.
Vendor output distinguishes an installed update from an already-current version.`

var codingAgentHelp = func(path string) ([]byte, error) {
	return exec.Command(path, "--help").Output()
}

func handleAgentUpdate(args []string) error {
	return updateCodingAgents(args, runtime.GOOS, os.Stdout)
}

func updateCodingAgents(args []string, goos string, out io.Writer) error {
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Fprintln(out, agentUpdateUsage)
		return nil
	}
	if len(args) != 1 {
		return fmt.Errorf("select one agent or --all\n%s", agentUpdateUsage)
	}
	all := args[0] == "--all"
	agents := []string{args[0]}
	if all {
		agents = []string{"claude", "codex", "copilot"}
	} else if args[0] != "claude" && args[0] != "codex" && args[0] != "copilot" {
		return fmt.Errorf("unknown agent %q\n%s", args[0], agentUpdateUsage)
	}
	if goos != "linux" && goos != "darwin" && goos != "windows" {
		return fmt.Errorf("coding-agent updates are unsupported on %s", goos)
	}
	var failures []error
	for _, agent := range agents {
		path, err := codingAgentLookPath(agent)
		if errors.Is(err, exec.ErrNotFound) || errors.Is(err, os.ErrNotExist) {
			if all {
				fmt.Fprintf(out, "%s: skipped (not found on PATH)\n", agent)
				continue
			}
			err = fmt.Errorf("not found on PATH; install with setupmjr install %s", agent)
		}
		if err == nil {
			err = updateCodingAgent(agent, path, goos, out)
		}
		if err != nil {
			fmt.Fprintf(out, "%s: FAILED: %v\n", agent, err)
			failures = append(failures, fmt.Errorf("%s: %w", agent, err))
		} else {
			fmt.Fprintf(out, "%s: update command completed (see vendor output above)\n", agent)
		}
	}
	return errors.Join(failures...)
}

func updateCodingAgent(agent, path, goos string, out io.Writer) error {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return fmt.Errorf("resolve installed executable: %w", err)
	}
	command, args, err := codingAgentUpdateCommand(agent, resolved, goos)
	if err != nil {
		return err
	}
	if command == "" {
		// Older CLIs can interpret an unknown subcommand as a chat prompt. Check
		// the command list first so an update never accidentally starts a session.
		help, err := codingAgentHelp(path)
		if err != nil {
			return fmt.Errorf("check native update support: %w", err)
		}
		if !advertisesUpdate(string(help)) {
			return fmt.Errorf("this CLI does not advertise a native update command; update using its original installer or package manager")
		}
		command, args = path, []string{"update"}
	} else if _, err := codingAgentLookPath(command); err != nil {
		return fmt.Errorf("%s is required for this installation: %w", command, err)
	}
	fmt.Fprintf(out, "%s: running %s %s\n", agent, command, strings.Join(args, " "))
	err = codingAgentRun(command, args...)
	// WinGet uses a nonzero status when the selected package is already current.
	var exitErr *exec.ExitError
	if command == "winget" && errors.As(err, &exitErr) && uint32(exitErr.ExitCode()) == 0x8A15002B {
		fmt.Fprintf(out, "%s: already up to date (WinGet)\n", agent)
		return nil
	}
	return err
}

// advertisesUpdate reports whether CLI help lists an update subcommand,
// including aliased forms such as Claude Code's "update|upgrade".
func advertisesUpdate(help string) bool {
	for _, line := range strings.Split(help, "\n") {
		fields := strings.Fields(line)
		if len(fields) > 0 && strings.HasPrefix(fields[0]+"|", "update|") {
			return true
		}
	}
	return false
}

// Use the resolved executable, not merely the presence of a package manager:
// a machine may have both native and package-managed installations on PATH.
func codingAgentUpdateCommand(agent, resolved, goos string) (string, []string, error) {
	path := strings.ReplaceAll(resolved, "\\", "/")
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if (part == "Caskroom" || part == "Cellar") && i+1 < len(parts) {
			pkg := parts[i+1]
			base := strings.SplitN(pkg, "@", 2)[0]
			expected := map[string]string{"claude": "claude-code", "codex": "codex", "copilot": "copilot-cli"}[agent]
			if base != expected {
				return "", nil, fmt.Errorf("unrecognised Homebrew package %q; update with its package manager", pkg)
			}
			kind := "--cask"
			if part == "Cellar" {
				kind = "--formula"
			}
			return "brew", []string{"upgrade", kind, pkg}, nil
		}
	}
	if goos == "windows" && strings.Contains(strings.ToLower(path), "/winget/packages/") {
		for _, pkg := range []string{"Anthropic.ClaudeCode", "GitHub.Copilot.Prerelease", "GitHub.Copilot"} {
			if strings.Contains(strings.ToLower(path), "/"+strings.ToLower(pkg)+"_") {
				expected := "copilot"
				if pkg == "Anthropic.ClaudeCode" {
					expected = "claude"
				}
				if agent == expected {
					return "winget", []string{"upgrade", "--id", pkg, "--exact", "--source", "winget"}, nil
				}
			}
		}
		return "", nil, fmt.Errorf("unrecognised WinGet installation; update with WinGet")
	}
	// Do not migrate package-managed installs to a native installation or choose
	// a new npm release channel. The operator's package manager owns these.
	if strings.Contains(path, "/node_modules/") || strings.HasSuffix(strings.ToLower(path), ".cmd") || strings.HasSuffix(strings.ToLower(path), ".ps1") {
		return "", nil, fmt.Errorf("package-managed installation; update with its original package manager (preserve the installed release channel)")
	}
	if goos != "windows" && (strings.HasPrefix(path, "/usr/bin/") || strings.HasPrefix(path, "/bin/") || strings.HasPrefix(path, "/nix/")) {
		return "", nil, fmt.Errorf("system-managed installation; update with its original package manager")
	}
	return "", nil, nil
}
