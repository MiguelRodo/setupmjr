package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var runExternalCommand = func(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func handleAgent(args []string) error {
	fs := flag.NewFlagSet("agent", flag.ContinueOnError)
	subagentAgy := fs.Bool("subagent-agy", false, "Configure Codex to use agy as an explicitly authorised Gemini subagent")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("agent does not accept positional arguments")
	}
	if !*subagentAgy {
		return fmt.Errorf("agent requires --subagent-agy")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("find home directory: %w", err)
	}
	if err := setupSubagentAgy(home); err != nil {
		return err
	}

	fmt.Println("Configured Codex agy subagent guidance and execution rule.")
	if _, err := exec.LookPath("agy"); err != nil {
		fmt.Println("Note: agy is not currently on PATH; install and authenticate Google Antigravity CLI before using the subagent setup.")
	}
	return nil
}

func handleProject(args []string) error {
	fs := flag.NewFlagSet("project", flag.ContinueOnError)
	ghSkill := fs.Bool("gh-skill", false, "Install the github-projects skill for universal agents at user scope")
	pj := fs.Bool("pj", false, "Install or update the pj launcher from MiguelRodo/pj")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("project does not accept positional arguments")
	}
	if !*ghSkill && !*pj {
		return fmt.Errorf("project requires --gh-skill and/or --pj")
	}

	if *ghSkill {
		if err := runExternalCommand(
			"gh", "skill", "install",
			"MiguelRodo/github-projects-skill", "github-projects",
			"--agent", "universal",
			"--scope", "user",
			"--force",
		); err != nil {
			return fmt.Errorf("install github-projects skill: %w", err)
		}
		fmt.Println("Installed github-projects at universal user scope.")
	}

	if *pj {
		if err := installPj(); err != nil {
			return fmt.Errorf("install pj launcher: %w", err)
		}
		fmt.Println("Installed or updated pj launcher from canonical source MiguelRodo/pj.")
	}
	return nil
}

func installPj() error {
	tmpDir, err := os.MkdirTemp("", "setupmjr-pj-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := runExternalCommand("git", "clone", "--depth", "1", "https://github.com/MiguelRodo/pj.git", tmpDir); err != nil {
		return fmt.Errorf("clone pj repo: %w", err)
	}

	installScript := filepath.Join(tmpDir, "install.sh")
	if err := runExternalCommand("bash", installScript); err != nil {
		return fmt.Errorf("run pj install.sh: %w", err)
	}
	return nil
}

func setupSubagentAgy(home string) error {
	codexInstructions := filepath.Join(home, ".codex", "AGENTS.md")
	if err := updateManagedBlock(
		codexInstructions,
		"<!-- setupmjr-subagent-agy:start -->",
		"<!-- setupmjr-subagent-agy:end -->",
		agyInstructionsBlock,
	); err != nil {
		return fmt.Errorf("update Codex agent instructions: %w", err)
	}

	codexRules := filepath.Join(home, ".codex", "rules", "default.rules")
	if err := updateManagedBlock(
		codexRules,
		"# setupmjr-subagent-agy:start",
		"# setupmjr-subagent-agy:end",
		agyRulesBlock,
	); err != nil {
		return fmt.Errorf("update Codex agy execution rule: %w", err)
	}
	return nil
}

func updateManagedBlock(path, startMarker, endMarker, block string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create parent directory: %w", err)
	}

	existingBytes, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read %s: %w", path, err)
	}
	existing := string(existingBytes)

	startCount := strings.Count(existing, startMarker)
	endCount := strings.Count(existing, endMarker)
	if startCount != endCount || startCount > 1 {
		return fmt.Errorf("malformed managed block markers in %s", path)
	}

	managed := startMarker + "\n" + strings.TrimSpace(block) + "\n" + endMarker
	var updated string
	if startCount == 1 {
		start := strings.Index(existing, startMarker)
		endStart := strings.Index(existing[start+len(startMarker):], endMarker)
		if endStart < 0 {
			return fmt.Errorf("malformed managed block markers in %s", path)
		}
		end := start + len(startMarker) + endStart + len(endMarker)
		updated = existing[:start] + managed + existing[end:]
	} else {
		updated = existing
		if strings.TrimSpace(updated) != "" {
			if !strings.HasSuffix(updated, "\n") {
				updated += "\n"
			}
			updated += "\n"
		}
		updated += managed + "\n"
	}

	mode := os.FileMode(0644)
	if info, statErr := os.Stat(path); statErr == nil {
		mode = info.Mode().Perm()
	} else if !os.IsNotExist(statErr) {
		return fmt.Errorf("stat %s: %w", path, statErr)
	}
	if err := os.WriteFile(path, []byte(updated), mode); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

const agyInstructionsBlock = `## Optional agy subagent delegation for Codex

Codex may use agy as an external Gemini worker only when the operator explicitly authorises agy for the current task or conversation. Conversation-level authorisation covers repeated useful calls within that conversation; the existence of this setup is not itself authorisation.

When authorised, delegate bounded mechanical or investigative work early enough to avoid loading unnecessary context into the primary model. Suitable work includes repository discovery, locating files or symbols, routine edits, test runs, and iterative mechanical fixes. Keep intent interpretation, architecture and design, consequential decisions, and final review in Codex.

Prefer a bounded call such as:

    agy -p "<self-contained delegated task>" --model gemini-3.8-flash-high --add-dir "/absolute/repository/root"

Register every repository the worker needs with --add-dir. Ask the worker to return a compact result covering status, files changed, checks or tests, and unresolved items rather than a full working transcript.

If a permission is blocked, use the normal narrow approval path. Do not add wildcard permissions or --dangerously-skip-permissions merely to enable delegation.`

const agyRulesBlock = `prefix_rule(
    pattern = ["agy"],
    decision = "allow",
    justification = "Allow the opt-in agy subagent command; Codex instructions still require explicit operator authorisation for the task or conversation.",
)`
