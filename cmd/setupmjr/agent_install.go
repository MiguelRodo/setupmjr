package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/MiguelRodo/setupmjr/internal/netutil"
)

// Coding-agent installers are owned by their vendors. setupmjr only stages and
// invokes the documented installers; it does not manage logins or API tokens.
const (
	claudeUnixInstaller = "https://claude.ai/install.sh"
	codexUnixInstaller  = "https://chatgpt.com/codex/install.sh"
	codexWinInstaller   = "https://chatgpt.com/codex/install.ps1"
	copilotInstaller    = "https://gh.io/copilot-install"
)

type codingAgentInstallPlan struct {
	command string
	args    []string
	url     string // Empty for package-manager installations.
	ext     string
}

var (
	codingAgentDownload = netutil.DownloadFile
	codingAgentLookPath = exec.LookPath
	codingAgentRun      = func(name string, args ...string) error {
		cmd := exec.Command(name, args...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
)

func codingAgentInstallPlanFor(agent, goos string) (codingAgentInstallPlan, error) {
	switch goos {
	case "windows":
		switch agent {
		case "claude":
			return codingAgentInstallPlan{command: "winget", args: []string{"install", "--id", "Anthropic.ClaudeCode", "--exact", "--source", "winget"}}, nil
		case "codex":
			return codingAgentInstallPlan{command: "powershell.exe", args: []string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-File"}, url: codexWinInstaller, ext: ".ps1"}, nil
		case "copilot":
			return codingAgentInstallPlan{command: "winget", args: []string{"install", "--id", "GitHub.Copilot", "--exact", "--source", "winget"}}, nil
		}
	case "linux", "darwin":
		switch agent {
		case "claude":
			return codingAgentInstallPlan{command: "bash", args: []string{"--noprofile", "--norc"}, url: claudeUnixInstaller, ext: ".sh"}, nil
		case "codex":
			return codingAgentInstallPlan{command: "sh", url: codexUnixInstaller, ext: ".sh"}, nil
		case "copilot":
			return codingAgentInstallPlan{command: "bash", args: []string{"--noprofile", "--norc"}, url: copilotInstaller, ext: ".sh"}, nil
		}
	default:
		return codingAgentInstallPlan{}, fmt.Errorf("coding-agent installation is unsupported on %s", goos)
	}
	return codingAgentInstallPlan{}, fmt.Errorf("unknown coding agent %q (choose claude, codex or copilot)", agent)
}

func installCodingAgent(agent string) error {
	return installCodingAgentForOS(agent, runtime.GOOS)
}

func installCodingAgentForOS(agent, goos string) error {
	plan, err := codingAgentInstallPlanFor(agent, goos)
	if err != nil {
		return err
	}
	if _, err := codingAgentLookPath(plan.command); err != nil {
		return fmt.Errorf("install %s: %s is required: %w", agent, plan.command, err)
	}

	// WinGet is responsible for fetching and validating its signed packages.
	if plan.url == "" {
		if err := codingAgentRun(plan.command, plan.args...); err != nil {
			return fmt.Errorf("install %s using %s: %w", agent, plan.command, err)
		}
		return nil
	}

	// Never pipe a partially downloaded remote script into a shell. The
	// bounded downloader must finish successfully before any code executes.
	tempDir, err := os.MkdirTemp("", "setupmjr-agent-install-*")
	if err != nil {
		return fmt.Errorf("prepare %s installer: %w", agent, err)
	}
	defer os.RemoveAll(tempDir)
	script := filepath.Join(tempDir, "install"+plan.ext)
	if err := codingAgentDownload(plan.url, script); err != nil {
		return fmt.Errorf("download %s installer: %w", agent, err)
	}
	args := append(append([]string(nil), plan.args...), script)
	if err := codingAgentRun(plan.command, args...); err != nil {
		return fmt.Errorf("run %s installer: %w", agent, err)
	}
	return nil
}
