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

	for _, rc := range []string{".bashrc", ".zshrc"} {
		path := filepath.Join(home, rc)
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		text := string(content)
		if strings.Contains(text, "ghp-unit-secret") {
			t.Fatalf("%s contains the literal token", rc)
		}
		if strings.Count(text, githubBlockStart) != 1 {
			t.Fatalf("%s managed block count = %d, want 1", rc, strings.Count(text, githubBlockStart))
		}
	}

	if err := SetupGitAuthText("--global"); err != nil {
		t.Fatalf("second SetupGitAuthText: %v", err)
	}
	for _, rc := range []string{".bashrc", ".zshrc"} {
		content, err := os.ReadFile(filepath.Join(home, rc))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Count(string(content), githubBlockStart) != 1 {
			t.Fatalf("%s duplicated the managed block", rc)
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
