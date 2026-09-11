package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func setupSubagentDeepSeek(home string) error {
	subagentHome := filepath.Join(home, ".codex-deepseek-subagent")
	provider, err := currentCodexProviderAt(subagentHome)
	if err != nil {
		return err
	}
	if provider != "deepseek" {
		if err := runDeepSeekSetup(home, subagentHome, "1"); err != nil {
			return fmt.Errorf("configure isolated DeepSeek Codex subagent: %w", err)
		}
	}

	codexInstructions := filepath.Join(home, ".codex", "AGENTS.md")
	if err := removeManagedBlock(codexInstructions, "<!-- setupmjr-subagent-agy:start -->", "<!-- setupmjr-subagent-agy:end -->"); err != nil {
		return err
	}
	if err := updateManagedBlock(
		codexInstructions,
		"<!-- setupmjr-subagent-deepseek:start -->",
		"<!-- setupmjr-subagent-deepseek:end -->",
		deepSeekInstructionsBlock(subagentHome),
	); err != nil {
		return fmt.Errorf("update Codex DeepSeek subagent instructions: %w", err)
	}

	codexRules := filepath.Join(home, ".codex", "rules", "default.rules")
	if err := removeManagedBlock(codexRules, "# setupmjr-subagent-agy:start", "# setupmjr-subagent-agy:end"); err != nil {
		return err
	}
	if err := updateManagedBlock(
		codexRules,
		"# setupmjr-subagent-deepseek:start",
		"# setupmjr-subagent-deepseek:end",
		deepSeekRulesBlock(subagentHome),
	); err != nil {
		return fmt.Errorf("update Codex DeepSeek subagent execution rule: %w", err)
	}
	return nil
}

func currentCodexProviderAt(codexHome string) (string, error) {
	content, err := os.ReadFile(filepath.Join(codexHome, "config.toml"))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	if hasTomlStringValue(string(content), "model_provider", "deepseek") {
		return "deepseek", nil
	}
	return "openai", nil
}

func removeManagedBlock(path, startMarker, endMarker string) error {
	existingBytes, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read %s: %w", path, err)
	}
	existing := string(existingBytes)
	startCount := strings.Count(existing, startMarker)
	endCount := strings.Count(existing, endMarker)
	if startCount != endCount || startCount > 1 {
		return fmt.Errorf("malformed managed block markers in %s", path)
	}
	if startCount == 0 {
		return nil
	}
	start := strings.Index(existing, startMarker)
	endStart := strings.Index(existing[start+len(startMarker):], endMarker)
	if endStart < 0 {
		return fmt.Errorf("malformed managed block markers in %s", path)
	}
	end := start + len(startMarker) + endStart + len(endMarker)
	before := strings.TrimRight(existing[:start], "\n")
	after := strings.TrimLeft(existing[end:], "\n")
	updated := before
	if before != "" && after != "" {
		updated += "\n\n"
	}
	updated += after
	if updated != "" && !strings.HasSuffix(updated, "\n") {
		updated += "\n"
	}
	mode := os.FileMode(0644)
	if info, statErr := os.Stat(path); statErr == nil {
		mode = info.Mode().Perm()
	}
	return os.WriteFile(path, []byte(updated), mode)
}

func clearSubagentDeepSeek(home string) error {
	if err := removeManagedBlock(filepath.Join(home, ".codex", "AGENTS.md"), "<!-- setupmjr-subagent-deepseek:start -->", "<!-- setupmjr-subagent-deepseek:end -->"); err != nil {
		return err
	}
	return removeManagedBlock(filepath.Join(home, ".codex", "rules", "default.rules"), "# setupmjr-subagent-deepseek:start", "# setupmjr-subagent-deepseek:end")
}

func deepSeekInstructionsBlock(codexHome string) string {
	return fmt.Sprintf(`## Optional DeepSeek subagent delegation for Codex

Codex may use the isolated DeepSeek Codex worker only when the operator explicitly authorises DeepSeek for the current task or conversation. Conversation-level authorisation covers repeated useful calls within that conversation; the existence of this setup is not itself authorisation.

When authorised, delegate bounded mechanical or investigative work early enough to avoid loading unnecessary context into the primary model. Suitable work includes repository discovery, locating files or symbols, routine edits, test runs, and iterative mechanical fixes. Keep intent interpretation, architecture and design, consequential decisions, and final review in the primary Codex session.

Run the worker with its isolated Codex home and workspace-write sandbox, for example:

    env CODEX_HOME=%q codex exec --sandbox workspace-write -C "/absolute/repository/root" "<self-contained delegated task>"

Ask the worker to return a compact result covering status, files changed, checks or tests, and unresolved items rather than a full working transcript. Do not use danger-full-access merely to enable delegation.`, codexHome)
}

func deepSeekRulesBlock(codexHome string) string {
	return fmt.Sprintf(`prefix_rule(
    pattern = ["env", %q, "codex", "exec"],
    decision = "allow",
    justification = "Allow the opt-in isolated DeepSeek Codex subagent command; Codex instructions still require explicit operator authorisation for the task or conversation.",
)`, "CODEX_HOME="+codexHome)
}
