package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

var agentDoctorLookPath = exec.LookPath
var agentDoctorRun = runAgentDoctorProbe

// Only extract a numeric version: raw CLI output can contain credentials or
// personal details, even on failure. Never print subprocess errors or auth output.
var agentDoctorVersion = regexp.MustCompile(`[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?\b`)

type doctorOutput struct{ buffer bytes.Buffer }

func (b *doctorOutput) Write(p []byte) (int, error) {
	n := len(p)
	if remaining := 64*1024 - b.buffer.Len(); remaining > 0 {
		if len(p) > remaining {
			p = p[:remaining]
		}
		_, _ = b.buffer.Write(p)
	}
	return n, nil
}

func runAgentDoctorProbe(path string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, args...)
	// No stdin, terminal, shell, prompts, agent sessions or inference requests.
	var output doctorOutput
	cmd.Stdout = &output
	cmd.Stderr = io.Discard
	cmd.WaitDelay = time.Second
	err := cmd.Run()
	return output.buffer.Bytes(), err
}

func handleAgentDoctor(args []string) error {
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Println("Usage: setupmjr agent doctor\nRead-only installation, configuration and local authentication checks. Exit 1 if any check fails.")
		return nil
	}
	if len(args) != 0 {
		return fmt.Errorf("usage: setupmjr agent doctor (no options)")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("find home directory: %w", err)
	}
	return runAgentDoctor(home, os.Stdout)
}

func runAgentDoctor(home string, out io.Writer) error {
	failures := 0
	report := func(status, message string) {
		fmt.Fprintf(out, "%s: %s\n", status, message)
		if status == "FAIL" {
			failures++
		}
	}
	platform := runtime.GOOS
	if platform == "linux" {
		kernel, _ := os.ReadFile("/proc/sys/kernel/osrelease")
		if os.Getenv("WSL_DISTRO_NAME") != "" || strings.Contains(strings.ToLower(string(kernel)), "microsoft") {
			platform = "linux (WSL; checking this Linux environment, not Windows installations)"
		}
	}
	fmt.Fprintf(out, "Agent doctor: %s/%s\n", platform, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		report("INFO", "Native Windows PATH/PATHEXT lookup; Bash/Zsh provider snippets are not loaded by PowerShell. Claude Code requires Git for Windows (Git Bash).")
	}
	report("INFO", "Local checks only; credential presence does not prove validity, network access or subscription entitlement.")

	codexProvider, err := currentCodexProvider(home)
	if err != nil {
		report("FAIL", "Cannot read managed Codex configuration; check access to ~/.codex/config.toml.")
	}
	copilotProvider := currentCopilotProvider(home)
	if _, err := os.ReadFile(copilotEnvPath(home)); err != nil && !os.IsNotExist(err) {
		report("FAIL", "Cannot read managed Copilot configuration; check access to ~/.config/setupmjr/agent/copilot.env.")
	}
	subagent, err := currentSubagent(home)
	if err != nil {
		report("FAIL", "Cannot read managed subagent instructions; check access to ~/.codex/AGENTS.md.")
	}
	fmt.Fprintf(out, "setupmjr configuration: Codex=%s; Copilot=%s; subagent=%s\n", codexProvider, copilotProvider, subagent)
	codexHome := filepath.Join(home, ".codex")
	if override := os.Getenv("CODEX_HOME"); override != "" {
		codexHome = override
		report("WARN", "CODEX_HOME overrides ~/.codex; managed provider/subagent settings above may not apply.")
	}
	activeCodexProvider, err := currentCodexProviderAt(codexHome)
	if err != nil {
		report("FAIL", "Cannot read active Codex configuration; check CODEX_HOME and config.toml permissions.")
	}

	for _, agent := range []string{"claude", "codex", "copilot"} {
		path, err := agentDoctorLookPath(agent)
		if err != nil {
			report("FAIL", fmt.Sprintf("%s not executable on PATH; run setupmjr install %s, then reopen this terminal and check PATH.", agent, agent))
			continue
		}
		fmt.Fprintf(out, "FOUND: %s at %q\n", agent, path)
		if runtime.GOOS == "windows" && (strings.EqualFold(filepath.Ext(path), ".cmd") || strings.EqualFold(filepath.Ext(path), ".bat")) {
			report("FAIL", agent+" resolves to a shell shim which doctor cannot probe directly; install the native executable with setupmjr install "+agent+" or put the native executable first on PATH.")
			continue
		}
		version, err := agentDoctorRun(path, "--version")
		if err != nil || agentDoctorVersion.Find(version) == nil {
			report("FAIL", fmt.Sprintf("%s --version failed, timed out or returned no recognizable version; repair the installation with setupmjr install %s and check its runtime dependencies.", agent, agent))
			continue
		}
		if agent == "copilot" && !strings.Contains(strings.ToLower(string(version)), "github copilot") {
			report("FAIL", "copilot on PATH does not identify as GitHub Copilot; check for an AWS Copilot/name collision and install GitHub Copilot CLI.")
			continue
		}
		report("OK", fmt.Sprintf("%s version %s", agent, agentDoctorVersion.Find(version)))
		switch agent {
		case "claude":
			_, err := agentDoctorRun(path, "auth", "status")
			if err != nil {
				report("FAIL", "Claude auth status failed or timed out; run claude auth status locally, then claude auth login if needed (or check your API/provider configuration).")
			} else {
				report("OK", "Claude reports local authentication configured.")
			}
		case "codex":
			if activeCodexProvider == "deepseek" {
				if readDeepSeekAPIKey(filepath.Join(codexHome, "config.toml")) == "" && strings.TrimSpace(os.Getenv("DEEPSEEK_API_KEY")) == "" {
					report("FAIL", "Active DeepSeek Codex configuration has no credential; run setupmjr agent --auth deepseek and setupmjr agent -c deepseek.")
				} else {
					report("OK", "DeepSeek Codex credential present (not validated).")
				}
			} else {
				_, err := agentDoctorRun(path, "login", "status")
				if err != nil {
					report("FAIL", "Codex login status failed or timed out; run codex login status locally, then codex login if needed. For a custom provider, check its credential configuration.")
				} else {
					report("OK", "Codex reports local authentication configured.")
				}
			}
		case "copilot":
			doctorCopilotAuth(copilotProvider, report)
		}
	}
	if copilotProvider == "deepseek" {
		key, err := os.ReadFile(deepSeekCredentialPath(home))
		if err != nil || strings.TrimSpace(string(key)) == "" {
			report("FAIL", "Managed Copilot DeepSeek credential missing, empty or unreadable; run setupmjr agent --auth deepseek.")
		}
		if os.Getenv("COPILOT_PROVIDER_BASE_URL") != copilotDeepSeekBaseURL || os.Getenv("COPILOT_PROVIDER_TYPE") != copilotDeepSeekProviderType || strings.TrimSpace(os.Getenv("COPILOT_PROVIDER_API_KEY")) == "" {
			report("FAIL", "Managed Copilot DeepSeek environment is not active; restart Bash/Zsh or source ~/.config/setupmjr/agent/copilot.env. On native Windows configure the provider in your shell; Bash exports do not configure PowerShell.")
		}
	}
	switch subagent {
	case "agy":
		if path, err := agentDoctorLookPath("agy"); err != nil {
			report("FAIL", "Configured agy subagent is not on PATH; install and authenticate Google Antigravity CLI, or select another subagent.")
		} else {
			fmt.Fprintf(out, "FOUND: agy at %q (authentication not checked)\n", path)
		}
	case "deepseek":
		dir := filepath.Join(home, ".codex-deepseek-subagent")
		provider, err := currentCodexProviderAt(dir)
		if err != nil || provider != "deepseek" || readDeepSeekAPIKey(filepath.Join(dir, "config.toml")) == "" {
			report("FAIL", "Isolated DeepSeek subagent configuration or credential missing/unreadable; run setupmjr agent --subagent deepseek.")
		}
	case "multiple (check configuration)":
		report("FAIL", "Multiple managed subagents configured; select one with setupmjr agent --subagent agy or --subagent deepseek.")
	}
	fmt.Fprintf(out, "Checks complete: %d failed. WARN means unverified, not failed.\n", failures)
	if failures > 0 {
		return fmt.Errorf("agent doctor: %d failed checks", failures)
	}
	return nil
}

func doctorCopilotAuth(provider string, report func(string, string)) {
	if strings.TrimSpace(os.Getenv("COPILOT_PROVIDER_API_KEY")) != "" {
		report("WARN", "Copilot provider API key present; custom provider access is not validated.")
		return
	}
	if provider != "github" || os.Getenv("COPILOT_PROVIDER_TYPE") != "" {
		report("WARN", "Copilot uses a custom provider; check provider credentials and configuration locally.")
		return
	}
	for _, name := range []string{"COPILOT_GITHUB_TOKEN", "GH_TOKEN", "GITHUB_TOKEN"} {
		if value, set := os.LookupEnv(name); set {
			if strings.TrimSpace(value) == "" {
				report("FAIL", name+" is empty; unset it or supply a supported Copilot token.")
			} else if strings.HasPrefix(value, "ghp_") {
				report("FAIL", name+" contains an unsupported classic PAT; use a fine-grained PAT with Copilot Requests permission or run copilot login.")
			} else {
				report("WARN", name+" present; token validity and Copilot entitlement are not checked.")
			}
			return
		}
	}
	report("WARN", "Copilot authentication unverified: credentials may be in the OS credential store. If access fails, run copilot login; doctor does not start sign-in.")
}
