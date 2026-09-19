package gitcmd

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSetupGitAuthTextStoresOneProtectedTokenWithoutLeakingIt(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", home)
	}
	t.Setenv("GH_TOKEN", "ghp-unit-secret")
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")

	if err := SetupGitAuthText("--global"); err != nil {
		t.Fatalf("SetupGitAuthText: %v", err)
	}

	tokenPath := githubTokenPath(home)
	token, err := os.ReadFile(tokenPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(token)); got != "ghp-unit-secret" {
		t.Fatalf("stored token = %q", got)
	}
	if runtime.GOOS != "windows" {
		if info, err := os.Stat(filepath.Dir(tokenPath)); err != nil {
			t.Fatal(err)
		} else if got := info.Mode().Perm(); got != 0700 {
			t.Fatalf("auth directory mode = %o, want 700", got)
		}
		if info, err := os.Stat(tokenPath); err != nil {
			t.Fatal(err)
		} else if got := info.Mode().Perm(); got != 0600 {
			t.Fatalf("token mode = %o, want 600", got)
		}
	}

	helper, err := RunCommand("git", "config", "--global", "--get", "credential.https://github.com.helper")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(helper, "ghp-unit-secret") {
		t.Fatal("Git config contains the literal token")
	}
	if !strings.Contains(helper, "github.token") {
		t.Fatalf("Git helper does not reference the protected token file: %s", helper)
	}

	for _, shellName := range []string{"bash", "zsh"} {
		rcPath := filepath.Join(home, "."+shellName+"rc")
		rc, err := os.ReadFile(rcPath)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(rc), "ghp-unit-secret") {
			t.Fatalf("%s contains the literal token", rcPath)
		}
		if strings.Contains(string(rc), githubBlockStart) {
			t.Fatalf("%s contains GitHub auth logic directly", rcPath)
		}
		if !strings.Contains(string(rc), "."+shellName+"rc.d") {
			t.Fatalf("%s does not source its rc.d directory", rcPath)
		}

		loginPath := filepath.Join(home, "."+shellName+"rc.d", "login.sh")
		login, err := os.ReadFile(loginPath)
		if err != nil {
			t.Fatal(err)
		}
		text := string(login)
		if strings.Contains(text, "ghp-unit-secret") {
			t.Fatalf("%s contains the literal token", loginPath)
		}
		if strings.Count(text, githubBlockStart) != 1 {
			t.Fatalf("%s managed block count = %d, want 1", loginPath, strings.Count(text, githubBlockStart))
		}
		if !strings.Contains(text, "github.token") {
			t.Fatalf("%s does not read the protected token file", loginPath)
		}
	}

	if err := SetupGitAuthText("--global"); err != nil {
		t.Fatalf("second SetupGitAuthText: %v", err)
	}
	for _, shellName := range []string{"bash", "zsh"} {
		login, err := os.ReadFile(filepath.Join(home, "."+shellName+"rc.d", "login.sh"))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Count(string(login), githubBlockStart) != 1 {
			t.Fatalf("%s login.sh duplicated the managed block", shellName)
		}
	}
}

func TestWireGitHubTokenEnvMigratesDirectRCBlockToLogin(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", home)
	}

	oldBlock := githubBlockStart + "\nold-direct-wiring=yes\n" + githubBlockEnd + "\n"
	for _, rc := range []string{".bashrc", ".zshrc"} {
		if err := os.WriteFile(filepath.Join(home, rc), []byte(oldBlock), 0644); err != nil {
			t.Fatal(err)
		}
	}

	if err := wireGitHubTokenEnv(home); err != nil {
		t.Fatal(err)
	}

	for _, shellName := range []string{"bash", "zsh"} {
		rc, err := os.ReadFile(filepath.Join(home, "."+shellName+"rc"))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(rc), githubBlockStart) || strings.Contains(string(rc), "old-direct-wiring") {
			t.Fatalf(".%src retained direct GitHub auth wiring", shellName)
		}
		login, err := os.ReadFile(filepath.Join(home, "."+shellName+"rc.d", "login.sh"))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Count(string(login), githubBlockStart) != 1 {
			t.Fatalf(".%src.d/login.sh managed block count = %d, want 1", shellName, strings.Count(string(login), githubBlockStart))
		}
	}
}

func TestScrubLegacyGitHubCredentialsPreservesUnrelatedLoginContent(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".bashrc.d", "login.sh")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	before := "KEEP_ME=yes\nGITHUB_USERNAME=\"miguel\"\nGH_TOKEN=\"ghp-old-secret\"\nHF_TOKEN=\"hf-keep\"\n"
	if err := os.WriteFile(path, []byte(before), 0755); err != nil {
		t.Fatal(err)
	}

	if err := scrubLegacyGitHubCredentials(home); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	for _, forbidden := range []string{"GITHUB_USERNAME=", "GH_TOKEN=", "ghp-old-secret"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("legacy GitHub credential remained: %q in %s", forbidden, text)
		}
	}
	for _, want := range []string{"KEEP_ME=yes", `HF_TOKEN="hf-keep"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("unrelated login content was removed: %q", want)
		}
	}
}

func TestWithoutGitHubTokenEnvRemovesOnlyGitHubTokens(t *testing.T) {
	got := withoutGitHubTokenEnv([]string{
		"PATH=/bin",
		"GH_TOKEN=one",
		"GITHUB_TOKEN=two",
		"HF_TOKEN=three",
	})
	joined := strings.Join(got, "\n")
	if strings.Contains(joined, "GH_TOKEN=") || strings.Contains(joined, "GITHUB_TOKEN=") {
		t.Fatalf("GitHub token environment survived filtering: %v", got)
	}
	for _, want := range []string{"PATH=/bin", "HF_TOKEN=three"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("unrelated environment variable missing: %s", want)
		}
	}
}
