package main

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCodingAgentInstallPlans(t *testing.T) {
	tests := []struct {
		agent, goos string
		want        codingAgentInstallPlan
	}{
		{"claude", "linux", codingAgentInstallPlan{command: "bash", args: []string{"--noprofile", "--norc"}, url: claudeUnixInstaller, ext: ".sh"}},
		{"claude", "darwin", codingAgentInstallPlan{command: "bash", args: []string{"--noprofile", "--norc"}, url: claudeUnixInstaller, ext: ".sh"}},
		{"codex", "linux", codingAgentInstallPlan{command: "sh", url: codexUnixInstaller, ext: ".sh"}},
		{"codex", "darwin", codingAgentInstallPlan{command: "sh", url: codexUnixInstaller, ext: ".sh"}},
		{"copilot", "linux", codingAgentInstallPlan{command: "bash", args: []string{"--noprofile", "--norc"}, url: copilotInstaller, ext: ".sh"}},
		{"copilot", "darwin", codingAgentInstallPlan{command: "bash", args: []string{"--noprofile", "--norc"}, url: copilotInstaller, ext: ".sh"}},
		{"claude", "windows", codingAgentInstallPlan{command: "winget", args: []string{"install", "--id", "Anthropic.ClaudeCode", "--exact", "--source", "winget"}}},
		{"codex", "windows", codingAgentInstallPlan{command: "powershell.exe", args: []string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-File"}, url: codexWinInstaller, ext: ".ps1"}},
		{"copilot", "windows", codingAgentInstallPlan{command: "winget", args: []string{"install", "--id", "GitHub.Copilot", "--exact", "--source", "winget"}}},
	}
	for _, tt := range tests {
		t.Run(tt.agent+"-"+tt.goos, func(t *testing.T) {
			got, err := codingAgentInstallPlanFor(tt.agent, tt.goos)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("plan = %#v; want %#v", got, tt.want)
			}
		})
	}
	for _, tt := range []struct{ agent, goos string }{{"unknown", "linux"}, {"claude", "freebsd"}, {"other", "windows"}} {
		if _, err := codingAgentInstallPlanFor(tt.agent, tt.goos); err == nil {
			t.Fatalf("expected an error for %s on %s", tt.agent, tt.goos)
		}
	}
}

func TestCodingAgentInstallerStagesCompleteScript(t *testing.T) {
	oldLook, oldDownload, oldRun := codingAgentLookPath, codingAgentDownload, codingAgentRun
	t.Cleanup(func() { codingAgentLookPath, codingAgentDownload, codingAgentRun = oldLook, oldDownload, oldRun })
	codingAgentLookPath = func(name string) (string, error) { return name, nil }
	var stagedPath, executedCommand string
	codingAgentDownload = func(url, path string) error {
		if url != codexUnixInstaller {
			t.Fatalf("unexpected installer URL %q", url)
		}
		return os.WriteFile(path, []byte("echo installed\n"), 0600)
	}
	codingAgentRun = func(name string, args ...string) error {
		executedCommand = name
		stagedPath = args[len(args)-1]
		data, err := os.ReadFile(stagedPath)
		if err != nil || string(data) != "echo installed\n" {
			t.Fatalf("script not fully downloaded at invocation: %q, %v", data, err)
		}
		if !strings.HasSuffix(stagedPath, ".sh") {
			t.Fatalf("unexpected script name %q", stagedPath)
		}
		return nil
	}
	if err := installCodingAgentForOS("codex", "linux"); err != nil {
		t.Fatal(err)
	}
	if executedCommand != "sh" {
		t.Fatalf("ran %q, expected sh", executedCommand)
	}
	if _, err := os.Stat(filepath.Dir(stagedPath)); !os.IsNotExist(err) {
		t.Fatalf("temporary installer directory was not removed: %v", err)
	}
}

func TestCodingAgentInstallerRefusesFailedDownload(t *testing.T) {
	oldLook, oldDownload, oldRun := codingAgentLookPath, codingAgentDownload, codingAgentRun
	t.Cleanup(func() { codingAgentLookPath, codingAgentDownload, codingAgentRun = oldLook, oldDownload, oldRun })
	codingAgentLookPath = func(name string) (string, error) { return name, nil }
	codingAgentDownload = func(string, string) error { return errors.New("network unavailable") }
	codingAgentRun = func(string, ...string) error { t.Fatal("installer ran after download failed"); return nil }
	err := installCodingAgentForOS("claude", "linux")
	if err == nil || !strings.Contains(err.Error(), "download claude installer") {
		t.Fatalf("expected download error, got %v", err)
	}
}

func TestCodingAgentInstallerRequiresCommandBeforeDownload(t *testing.T) {
	oldLook, oldDownload := codingAgentLookPath, codingAgentDownload
	t.Cleanup(func() { codingAgentLookPath, codingAgentDownload = oldLook, oldDownload })
	codingAgentLookPath = func(name string) (string, error) { return "", os.ErrNotExist }
	codingAgentDownload = func(string, string) error { t.Fatal("downloaded without required command"); return nil }
	err := installCodingAgentForOS("copilot", "windows")
	if err == nil || !strings.Contains(err.Error(), "winget is required") {
		t.Fatalf("unexpected prerequisite error: %v", err)
	}
}

func TestCodingAgentPackageManagerInstallation(t *testing.T) {
	oldLook, oldRun := codingAgentLookPath, codingAgentRun
	t.Cleanup(func() { codingAgentLookPath, codingAgentRun = oldLook, oldRun })
	codingAgentLookPath = func(name string) (string, error) { return name, nil }
	codingAgentRun = func(name string, args ...string) error {
		if name != "winget" || !reflect.DeepEqual(args, []string{"install", "--id", "GitHub.Copilot", "--exact", "--source", "winget"}) {
			t.Fatalf("unexpected package-manager command %s %v", name, args)
		}
		return nil
	}
	if err := installCodingAgentForOS("copilot", "windows"); err != nil {
		t.Fatal(err)
	}
}

func TestInstallSubcommandValidationAndLegacyFlags(t *testing.T) {
	for _, args := range [][]string{{"unknown"}, {"claude", "--pj"}, {"copilot", "other"}, {"--unknown"}} {
		if err := handleInstall(args); err == nil {
			t.Fatalf("expected error for install %v", args)
		}
	}
	oldRepos := installRepos
	t.Cleanup(func() { installRepos = oldRepos })
	calls := 0
	installRepos = func() error { calls++; return nil }
	if err := handleInstall([]string{"--repos"}); err != nil {
		t.Fatalf("existing --repos flag failed: %v", err)
	}
	if calls != 1 {
		t.Fatalf("legacy installer invoked %d times", calls)
	}
	if err := handleInstall([]string{"--help"}); err != nil {
		t.Fatalf("help failed: %v", err)
	}
}
