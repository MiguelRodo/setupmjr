package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestParseAgentArgsAuthRequiresEndpoint(t *testing.T) {
	if _, err := parseAgentArgs([]string{"--auth"}); err == nil {
		t.Fatal("--auth without endpoint unexpectedly succeeded")
	}
}

func TestParseAgentArgsAuthDeepSeek(t *testing.T) {
	for _, args := range [][]string{
		{"--auth", "deepseek"},
		{"-a", "d"},
	} {
		got, err := parseAgentArgs(args)
		if err != nil {
			t.Fatalf("parseAgentArgs(%v): %v", args, err)
		}
		if got.authEndpoint != "deepseek" {
			t.Fatalf("parseAgentArgs(%v) auth endpoint = %q, want deepseek", args, got.authEndpoint)
		}
	}
}

func TestParseAgentArgsAuthRejectsUnknownEndpoint(t *testing.T) {
	if _, err := parseAgentArgs([]string{"--auth", "other"}); err == nil {
		t.Fatal("unknown auth endpoint unexpectedly succeeded")
	}
}

func TestParseAgentArgsCopilotProvider(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{args: []string{"-p"}, want: "github"},
		{args: []string{"-p", "d"}, want: "deepseek"},
		{args: []string{"--copilot-provider", "github"}, want: "github"},
		{args: []string{"--copilot-provider", "default"}, want: "github"},
	}
	for _, tt := range tests {
		got, err := parseAgentArgs(tt.args)
		if err != nil {
			t.Fatalf("parseAgentArgs(%v): %v", tt.args, err)
		}
		if got.copilotProvider != tt.want {
			t.Fatalf("parseAgentArgs(%v) Copilot provider = %q, want %q", tt.args, got.copilotProvider, tt.want)
		}
	}
}

func TestParseAgentArgsCopilotProviderRejectsUnknown(t *testing.T) {
	if _, err := parseAgentArgs([]string{"--copilot-provider", "other"}); err == nil {
		t.Fatal("unknown Copilot provider unexpectedly succeeded")
	}
}

func TestSetupDeepSeekAuthStoresCredentialOnly(t *testing.T) {
	home := t.TempDir()
	t.Setenv("DEEPSEEK_API_KEY", "sk-test-deepseek")

	if err := setupDeepSeekAuth(home); err != nil {
		t.Fatalf("setupDeepSeekAuth failed: %v", err)
	}

	keyPath := deepSeekCredentialPath(home)
	keyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(keyBytes)) != "sk-test-deepseek" {
		t.Fatalf("stored key = %q", strings.TrimSpace(string(keyBytes)))
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(keyPath)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != 0600 {
			t.Fatalf("key mode = %o, want 600", got)
		}
	}

	if _, err := os.Stat(copilotEnvPath(home)); !os.IsNotExist(err) {
		t.Fatalf("auth unexpectedly configured Copilot provider environment: %v", err)
	}
	for _, rc := range []string{".bashrc", ".zshrc"} {
		if _, err := os.Stat(filepath.Join(home, rc)); !os.IsNotExist(err) {
			t.Fatalf("auth unexpectedly modified %s: %v", rc, err)
		}
	}
}

func TestConfigureCopilotDeepSeekRequiresAuth(t *testing.T) {
	home := t.TempDir()
	if err := configureCopilotDeepSeek(home); err == nil {
		t.Fatal("DeepSeek provider setup unexpectedly succeeded without stored auth")
	}
}

func TestConfigureCopilotDeepSeekWritesProviderConfiguration(t *testing.T) {
	home := t.TempDir()
	t.Setenv("DEEPSEEK_API_KEY", "sk-test-deepseek")
	if err := setupDeepSeekAuth(home); err != nil {
		t.Fatal(err)
	}

	if err := configureCopilotDeepSeek(home); err != nil {
		t.Fatalf("configureCopilotDeepSeek failed: %v", err)
	}

	envBytes, err := os.ReadFile(copilotEnvPath(home))
	if err != nil {
		t.Fatal(err)
	}
	env := string(envBytes)
	for _, want := range []string{
		"export COPILOT_PROVIDER_TYPE='anthropic'",
		"export COPILOT_PROVIDER_BASE_URL='https://api.deepseek.com/anthropic'",
		"export COPILOT_MODEL='deepseek-v4-pro'",
		"export COPILOT_PROVIDER_MAX_PROMPT_TOKENS='840000'",
		"export COPILOT_PROVIDER_MAX_OUTPUT_TOKENS='128000'",
		"deepseek.key",
	} {
		if !strings.Contains(env, want) {
			t.Fatalf("Copilot environment missing %q:\n%s", want, env)
		}
	}
	if strings.Contains(env, "sk-test-deepseek") {
		t.Fatal("Copilot environment duplicated the raw API key")
	}
	if got := currentCopilotProvider(home); got != "deepseek" {
		t.Fatalf("currentCopilotProvider = %q, want deepseek", got)
	}

	for _, rc := range []string{".bashrc", ".zshrc"} {
		path := filepath.Join(home, rc)
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		text := string(content)
		if strings.Count(text, "# setupmjr-copilot-provider:start") != 1 {
			t.Fatalf("%s managed block count wrong:\n%s", rc, text)
		}
		if !strings.Contains(text, ".config/setupmjr/agent/copilot.env") {
			t.Fatalf("%s does not source Copilot provider environment:\n%s", rc, text)
		}
	}

	if err := configureCopilotDeepSeek(home); err != nil {
		t.Fatalf("second configureCopilotDeepSeek failed: %v", err)
	}
	for _, rc := range []string{".bashrc", ".zshrc"} {
		content, err := os.ReadFile(filepath.Join(home, rc))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Count(string(content), "# setupmjr-copilot-provider:start") != 1 {
			t.Fatalf("%s duplicated managed block", rc)
		}
	}
}

func TestAuthDoesNotChangeExistingCopilotProvider(t *testing.T) {
	home := t.TempDir()
	t.Setenv("DEEPSEEK_API_KEY", "sk-first")
	if err := setupDeepSeekAuth(home); err != nil {
		t.Fatal(err)
	}
	if err := configureCopilotDeepSeek(home); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(copilotEnvPath(home))
	if err != nil {
		t.Fatal(err)
	}

	t.Setenv("DEEPSEEK_API_KEY", "sk-second")
	if err := setupDeepSeekAuth(home); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(copilotEnvPath(home))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("credential refresh changed Copilot provider configuration")
	}
}

func TestConfigureCopilotGitHubClearsBYOKProvider(t *testing.T) {
	home := t.TempDir()
	t.Setenv("DEEPSEEK_API_KEY", "sk-test-deepseek")
	if err := setupDeepSeekAuth(home); err != nil {
		t.Fatal(err)
	}
	if err := configureCopilotDeepSeek(home); err != nil {
		t.Fatal(err)
	}

	if err := configureCopilotGitHub(home); err != nil {
		t.Fatalf("configureCopilotGitHub failed: %v", err)
	}
	if got := currentCopilotProvider(home); got != "github" {
		t.Fatalf("currentCopilotProvider = %q, want github", got)
	}
	envBytes, err := os.ReadFile(copilotEnvPath(home))
	if err != nil {
		t.Fatal(err)
	}
	env := string(envBytes)
	for _, want := range []string{
		"unset COPILOT_PROVIDER_TYPE",
		"unset COPILOT_PROVIDER_BASE_URL",
		"unset COPILOT_PROVIDER_API_KEY",
		"unset COPILOT_MODEL",
	} {
		if !strings.Contains(env, want) {
			t.Fatalf("GitHub provider environment missing %q:\n%s", want, env)
		}
	}
	if strings.Contains(env, copilotDeepSeekBaseURL) || strings.Contains(env, copilotDeepSeekModel) {
		t.Fatalf("DeepSeek provider configuration remained after switching to GitHub:\n%s", env)
	}
}

func TestDeepSeekAPIKeyReadsManagedCredential(t *testing.T) {
	home := t.TempDir()
	t.Setenv("DEEPSEEK_API_KEY", "")

	keyPath := deepSeekCredentialPath(home)
	if err := os.MkdirAll(filepath.Dir(keyPath), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, []byte("sk-managed\n"), 0600); err != nil {
		t.Fatal(err)
	}

	got, err := deepSeekAPIKey(home, filepath.Join(home, ".codex"))
	if err != nil {
		t.Fatal(err)
	}
	if got != "sk-managed" {
		t.Fatalf("deepSeekAPIKey = %q, want sk-managed", got)
	}
}
