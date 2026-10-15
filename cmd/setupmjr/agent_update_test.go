package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestAgentUpdateCommands(t *testing.T) {
	for _, tt := range []struct {
		agent, path, goos, command string
		args                       []string
		fails                      bool
	}{
		{"claude", "/home/me/.local/share/claude/versions/1", "linux", "", nil, false},
		{"codex", "/home/me/.local/bin/codex", "linux", "", nil, false},
		{"copilot", "/usr/local/bin/copilot", "darwin", "", nil, false},
		{"claude", "/opt/homebrew/Caskroom/claude-code@latest/1/claude", "darwin", "brew", []string{"upgrade", "--cask", "claude-code@latest"}, false},
		{"codex", "/opt/homebrew/Caskroom/codex/1/codex", "darwin", "brew", []string{"upgrade", "--cask", "codex"}, false},
		{"copilot", "/home/linuxbrew/.linuxbrew/Cellar/copilot-cli/1/bin/copilot", "linux", "brew", []string{"upgrade", "--formula", "copilot-cli"}, false},
		{"copilot", "/opt/homebrew/Caskroom/copilot-cli@prerelease/1/copilot", "darwin", "brew", []string{"upgrade", "--cask", "copilot-cli@prerelease"}, false},
		{"claude", `C:\Users\me\AppData\Local\Microsoft\WinGet\Packages\Anthropic.ClaudeCode_Microsoft.Winget.Source_8wekyb3d8bbwe\claude.exe`, "windows", "winget", []string{"upgrade", "--id", "Anthropic.ClaudeCode", "--exact", "--source", "winget"}, false},
		{"copilot", `C:\WinGet\Packages\GitHub.Copilot.Prerelease_source\copilot.exe`, "windows", "winget", []string{"upgrade", "--id", "GitHub.Copilot.Prerelease", "--exact", "--source", "winget"}, false},
		{"codex", `C:\Users\me\.local\bin\codex.exe`, "windows", "", nil, false},
		{"codex", "/usr/local/lib/node_modules/@openai/codex/bin/codex.js", "linux", "", nil, true},
		{"copilot", `C:\npm\copilot.cmd`, "windows", "", nil, true},
		{"claude", "/usr/bin/claude", "linux", "", nil, true},
		{"codex", "/nix/store/codex/bin/codex", "linux", "", nil, true},
		{"codex", "/opt/homebrew/Caskroom/other/1/codex", "darwin", "", nil, true},
	} {
		t.Run(tt.agent+tt.path, func(t *testing.T) {
			command, args, err := codingAgentUpdateCommand(tt.agent, tt.path, tt.goos)
			if (err != nil) != tt.fails || command != tt.command || !reflect.DeepEqual(args, tt.args) {
				t.Fatalf("got %s %v, %v; want %s %v, fails=%v", command, args, err, tt.command, tt.args, tt.fails)
			}
		})
	}
}

func TestAgentUpdateAllContinuesAndReports(t *testing.T) {
	oldLook, oldRun, oldHelp := codingAgentLookPath, codingAgentRun, codingAgentHelp
	t.Cleanup(func() { codingAgentLookPath, codingAgentRun, codingAgentHelp = oldLook, oldRun, oldHelp })
	dir := t.TempDir()
	for _, name := range []string{"claude", "codex"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0700); err != nil {
			t.Fatal(err)
		}
	}
	codingAgentLookPath = func(name string) (string, error) {
		if name == "copilot" {
			return "", exec.ErrNotFound
		}
		return filepath.Join(dir, name), nil
	}
	codingAgentHelp = func(string) ([]byte, error) { return []byte("Commands:\n  update  Update the CLI\n"), nil }
	var calls []string
	codingAgentRun = func(name string, args ...string) error {
		calls = append(calls, filepath.Base(name))
		if !reflect.DeepEqual(args, []string{"update"}) {
			t.Fatalf("unexpected args %v", args)
		}
		if filepath.Base(name) == "claude" {
			return errors.New("network failed")
		}
		return nil
	}
	var out bytes.Buffer
	err := updateCodingAgents([]string{"--all"}, "linux", &out)
	if err == nil || !strings.Contains(err.Error(), "claude: network failed") {
		t.Fatalf("got %v", err)
	}
	if !reflect.DeepEqual(calls, []string{"claude", "codex"}) {
		t.Fatalf("calls = %v", calls)
	}
	for _, want := range []string{"claude: FAILED", "codex: update command completed", "copilot: skipped"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("missing %q in %s", want, &out)
		}
	}
	out.Reset()
	if err := updateCodingAgents([]string{"copilot"}, "linux", &out); err == nil || !strings.Contains(err.Error(), "setupmjr install copilot") {
		t.Fatalf("missing single agent should fail with installation guidance: %v", err)
	}
}

func TestAgentUpdateValidation(t *testing.T) {
	oldLook := codingAgentLookPath
	t.Cleanup(func() { codingAgentLookPath = oldLook })
	codingAgentLookPath = func(string) (string, error) { t.Fatal("looked up command for invalid arguments"); return "", nil }
	for _, args := range [][]string{nil, {"--all", "codex"}, {"codex", "--all"}, {"claude", "codex"}, {"--config"}, {"other"}} {
		if err := updateCodingAgents(args, "linux", &bytes.Buffer{}); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
	if err := updateCodingAgents([]string{"--all"}, "freebsd", &bytes.Buffer{}); err == nil {
		t.Fatal("accepted unsupported platform")
	}
	if err := handleAgent([]string{"update", "--help"}); err != nil {
		t.Fatal(err)
	}
}

func TestAgentNativeUpdateBoundary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX fake executable; Windows routing covered by command tests")
	}
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("CODEX_HOME", filepath.Join(dir, ".codex"))
	t.Setenv("COPILOT_AUTO_UPDATE", "false")
	config := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(config, []byte("provider and credential sentinel\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cli := filepath.Join(dir, "codex")
	script := `#!/bin/sh
case "$1" in
 --help) echo '  update  Update the CLI' ;;
 update)
  test "$COPILOT_AUTO_UPDATE" = false || exit 21
  test "$CODEX_HOME" = "$HOME/.codex" || exit 22
  printf updated > "$HOME/result"
  ;;
 *) exit 23 ;;
esac
`
	if err := os.WriteFile(cli, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	// Resolve the PATH launcher to detect the installation method, but execute
	// that launcher so wrappers and vendor environment handling are retained.
	link := filepath.Join(dir, "launcher")
	if err := os.Symlink(cli, link); err != nil {
		t.Fatal(err)
	}
	if err := updateCodingAgent("codex", link, "linux", &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(filepath.Join(dir, "result")); err != nil || string(data) != "updated" {
		t.Fatalf("native update did not execute: %q %v", data, err)
	}
	if data, err := os.ReadFile(config); err != nil || string(data) != "provider and credential sentinel\n" {
		t.Fatalf("configuration changed: %q %v", data, err)
	}
	if err := os.Remove(filepath.Join(dir, "result")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cli, []byte("#!/bin/sh\nif [ \"$1\" = --help ]; then echo 'Commands: login exec'; else touch \"$HOME/result\"; fi\n"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := updateCodingAgent("codex", link, "linux", &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "original installer") {
		t.Fatalf("old CLI was not rejected: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "result")); !os.IsNotExist(err) {
		t.Fatalf("old CLI launched a prompt: %v", err)
	}
}

func TestAgentUpdateUsesResolvedPackageAndRequiresManager(t *testing.T) {
	oldLook, oldRun, oldHelp := codingAgentLookPath, codingAgentRun, codingAgentHelp
	t.Cleanup(func() { codingAgentLookPath, codingAgentRun, codingAgentHelp = oldLook, oldRun, oldHelp })
	dir := t.TempDir()
	binary := filepath.Join(dir, "Caskroom", "claude-code@latest", "1", "claude")
	if err := os.MkdirAll(filepath.Dir(binary), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(binary, nil, 0700); err != nil {
		t.Fatal(err)
	}
	launcher := binary
	if runtime.GOOS != "windows" {
		launcher = filepath.Join(dir, "claude")
		if err := os.Symlink(binary, launcher); err != nil {
			t.Fatal(err)
		}
	}
	codingAgentHelp = func(string) ([]byte, error) {
		t.Fatal("probed native updater for Homebrew installation")
		return nil, nil
	}
	calls := 0
	codingAgentRun = func(name string, args ...string) error {
		calls++
		if name != "brew" || !reflect.DeepEqual(args, []string{"upgrade", "--cask", "claude-code@latest"}) {
			t.Fatalf("unexpected command: %s %v", name, args)
		}
		return errors.New("brew failed")
	}
	codingAgentLookPath = func(name string) (string, error) { return "", exec.ErrNotFound }
	if err := updateCodingAgent("claude", launcher, "darwin", &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "brew is required") {
		t.Fatalf("missing manager: %v", err)
	}
	if calls != 0 {
		t.Fatal("ran without package manager")
	}
	codingAgentLookPath = func(name string) (string, error) { return name, nil }
	if err := updateCodingAgent("claude", launcher, "darwin", &bytes.Buffer{}); err == nil || err.Error() != "brew failed" {
		t.Fatalf("lost manager failure: %v", err)
	}
	if calls != 1 {
		t.Fatalf("ran updater %d times", calls)
	}
}

func TestAdvertisesUpdate(t *testing.T) {
	for help, want := range map[string]bool{
		"Commands:\n  update|upgrade   Check for updates and install if available\n": true, // Claude Code
		"Commands:\n  update            Update Codex to the latest version\n":        true, // Codex
		"Commands:\n  update       Download the latest version\n":                    true, // Copilot
		"Commands:\n  login   Sign in\n  updates-feed  Show news\n":                  false,
	} {
		if got := advertisesUpdate(help); got != want {
			t.Errorf("advertisesUpdate(%q) = %v, want %v", help, got, want)
		}
	}
}
