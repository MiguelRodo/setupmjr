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

func TestSetupCopilotDeepSeekAuthWritesSecureReusableConfiguration(t *testing.T) {
	home := t.TempDir()
	t.Setenv("DEEPSEEK_API_KEY", "sk-test-deepseek")

	if err := setupCopilotDeepSeekAuth(home); err != nil {
		t.Fatalf("setupCopilotDeepSeekAuth failed: %v", err)
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
	if got := currentCopilotEndpoint(home); got != "deepseek" {
		t.Fatalf("currentCopilotEndpoint = %q, want deepseek", got)
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

	if err := setupCopilotDeepSeekAuth(home); err != nil {
		t.Fatalf("second setupCopilotDeepSeekAuth failed: %v", err)
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
