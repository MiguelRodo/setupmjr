package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRunDeepSeekOfficialInstallerStagesDownloadAndLimitsEnvironment(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires bash")
	}

	home := t.TempDir()
	codexHome := filepath.Join(home, ".codex")
	t.Setenv("DEEPSEEK_API_KEY", "sk-test")
	t.Setenv("OPENAI_API_KEY", "must-not-leak")
	t.Setenv("BASH_ENV", filepath.Join(home, "must-not-source"))

	originalDownload := downloadRemoteFile
	defer func() { downloadRemoteFile = originalDownload }()
	downloadRemoteFile = func(url, dest string) error {
		if url != deepSeekCodexSetupURL {
			t.Fatalf("download URL = %q", url)
		}
		script := `read -r action
printf '%s' "$action" > "$HOME/action.txt"
printf '%s' "$CODEX_HOME" > "$HOME/codex-home.txt"
printf '%s' "$DEEPSEEK_API_KEY" > "$HOME/deepseek-key.txt"
printf '%s' "${OPENAI_API_KEY:-}" > "$HOME/openai-key.txt"
printf '%s' "${BASH_ENV:-}" > "$HOME/bash-env.txt"
`
		return os.WriteFile(dest, []byte(script), 0600)
	}

	if err := runDeepSeekOfficialInstaller(home, codexHome, "1"); err != nil {
		t.Fatalf("runDeepSeekOfficialInstaller() error = %v", err)
	}

	checks := map[string]string{
		"action.txt":       "1",
		"codex-home.txt":   codexHome,
		"deepseek-key.txt": "sk-test",
		"openai-key.txt":   "",
		"bash-env.txt":     "",
	}
	for name, want := range checks {
		got, err := os.ReadFile(filepath.Join(home, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if strings.TrimSpace(string(got)) != want {
			t.Fatalf("%s = %q, want %q", name, strings.TrimSpace(string(got)), want)
		}
	}
}

func TestRunDeepSeekOfficialInstallerDoesNotRunAfterDownloadFailure(t *testing.T) {
	home := t.TempDir()
	codexHome := filepath.Join(home, ".codex")

	originalDownload := downloadRemoteFile
	defer func() { downloadRemoteFile = originalDownload }()
	downloadRemoteFile = func(string, string) error {
		return os.ErrDeadlineExceeded
	}

	err := runDeepSeekOfficialInstaller(home, codexHome, "9")
	if err == nil {
		t.Fatal("failed download unexpectedly ran installer")
	}
	if !strings.Contains(err.Error(), "download official DeepSeek Codex setup script") {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, statErr := os.Stat(codexHome); !os.IsNotExist(statErr) {
		t.Fatalf("failed download created Codex home: %v", statErr)
	}
}
