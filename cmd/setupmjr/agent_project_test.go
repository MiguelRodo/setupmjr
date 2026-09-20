package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestParseAgentArgsDefaultsAndAliases(t *testing.T) {
	tests := []struct {
		args         []string
		wantProvider string
		wantSubagent string
	}{
		{args: []string{"-c"}, wantProvider: "openai"},
		{args: []string{"-c", "d"}, wantProvider: "deepseek"},
		{args: []string{"--codex-provider", "chatgpt"}, wantProvider: "openai"},
		{args: []string{"-s"}, wantSubagent: "agy"},
		{args: []string{"-s", "deepseek"}, wantSubagent: "deepseek"},
		{args: []string{"--subagent", "a"}, wantSubagent: "agy"},
	}
	for _, tt := range tests {
		got, err := parseAgentArgs(tt.args)
		if err != nil {
			t.Fatalf("parseAgentArgs(%v): %v", tt.args, err)
		}
		if got.codexProvider != tt.wantProvider || got.subagent != tt.wantSubagent {
			t.Fatalf("parseAgentArgs(%v) = provider %q, subagent %q; want %q, %q", tt.args, got.codexProvider, got.subagent, tt.wantProvider, tt.wantSubagent)
		}
	}
}

func TestParseAgentArgsRejectsUnknownChoices(t *testing.T) {
	if _, err := parseAgentArgs([]string{"-c", "other"}); err == nil {
		t.Fatal("unknown Codex provider unexpectedly succeeded")
	}
	if _, err := parseAgentArgs([]string{"-s", "other"}); err == nil {
		t.Fatal("unknown subagent unexpectedly succeeded")
	}
}

func TestCurrentCodexProviderIgnoresNestedProviderSettings(t *testing.T) {
	home := t.TempDir()
	codexHome := filepath.Join(home, ".codex")
	if err := os.MkdirAll(codexHome, 0700); err != nil {
		t.Fatal(err)
	}
	config := `model = "gpt-5.6-sol"

[profiles.deepseek]
model_provider = "deepseek"
`
	if err := os.WriteFile(filepath.Join(codexHome, "config.toml"), []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	provider, err := currentCodexProvider(home)
	if err != nil {
		t.Fatal(err)
	}
	if provider != "openai" {
		t.Fatalf("provider = %q, want openai", provider)
	}
}

func TestSetupSubagentAgyIsIdempotentAndPreservesExistingInstructions(t *testing.T) {
	home := t.TempDir()
	instructions := filepath.Join(home, ".codex", "AGENTS.md")
	if err := os.MkdirAll(filepath.Dir(instructions), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(instructions, []byte("# Existing instructions\n\nKeep this.\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := setupSubagentAgy(home); err != nil {
		t.Fatalf("first setup failed: %v", err)
	}
	if err := setupSubagentAgy(home); err != nil {
		t.Fatalf("second setup failed: %v", err)
	}

	gotInstructions, err := os.ReadFile(instructions)
	if err != nil {
		t.Fatal(err)
	}
	text := string(gotInstructions)
	if !strings.Contains(text, "# Existing instructions") || !strings.Contains(text, "Keep this.") {
		t.Fatalf("existing instructions were not preserved:\n%s", text)
	}
	if strings.Count(text, "<!-- setupmjr-subagent-agy:start -->") != 1 {
		t.Fatalf("managed instruction block was duplicated:\n%s", text)
	}
	if !strings.Contains(text, "gemini-3.8-flash-high") {
		t.Fatalf("delegated model guidance missing:\n%s", text)
	}

	rules := filepath.Join(home, ".codex", "rules", "default.rules")
	gotRules, err := os.ReadFile(rules)
	if err != nil {
		t.Fatal(err)
	}
	ruleText := string(gotRules)
	if strings.Count(ruleText, "# setupmjr-subagent-agy:start") != 1 {
		t.Fatalf("managed rule block was duplicated:\n%s", ruleText)
	}
	if !strings.Contains(ruleText, `pattern = ["agy"]`) {
		t.Fatalf("agy allow rule missing:\n%s", ruleText)
	}
}

func TestSetupSubagentDeepSeekUsesIsolatedCodexHomeAndReplacesAgy(t *testing.T) {
	home := t.TempDir()
	if err := setupSubagentAgy(home); err != nil {
		t.Fatal(err)
	}

	originalSetup := runDeepSeekSetup
	defer func() { runDeepSeekSetup = originalSetup }()
	var gotCodexHome, gotAction string
	runDeepSeekSetup = func(_ string, codexHome, action string) error {
		gotCodexHome = codexHome
		gotAction = action
		if err := os.MkdirAll(codexHome, 0700); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(codexHome, "config.toml"), []byte("model_provider = \"deepseek\"\nmodel = \"deepseek-flash\"\n"), 0600); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(codexHome, "models.json"), []byte("{}\n"), 0600)
	}

	if err := setupSubagentDeepSeek(home); err != nil {
		t.Fatalf("setupSubagentDeepSeek failed: %v", err)
	}
	wantHome := filepath.Join(home, ".codex-deepseek-subagent")
	if gotCodexHome != wantHome || gotAction != "1" {
		t.Fatalf("DeepSeek setup called with home %q action %q; want %q, 1", gotCodexHome, gotAction, wantHome)
	}

	instructions, err := os.ReadFile(filepath.Join(home, ".codex", "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(instructions)
	if strings.Contains(text, "setupmjr-subagent-agy:start") {
		t.Fatalf("agy block remained after DeepSeek switch:\n%s", text)
	}
	if !strings.Contains(text, "setupmjr-subagent-deepseek:start") || !strings.Contains(text, strconv.Quote(wantHome)) {
		t.Fatalf("DeepSeek block missing or wrong isolated home:\n%s", text)
	}

	rules, err := os.ReadFile(filepath.Join(home, ".codex", "rules", "default.rules"))
	if err != nil {
		t.Fatal(err)
	}
	ruleText := string(rules)
	if strings.Contains(ruleText, "setupmjr-subagent-agy:start") || !strings.Contains(ruleText, strconv.Quote("CODEX_HOME="+wantHome)) {
		t.Fatalf("DeepSeek execution rule not installed cleanly:\n%s", ruleText)
	}

	if err := setupSubagentAgy(home); err != nil {
		t.Fatal(err)
	}
	instructions, err = os.ReadFile(filepath.Join(home, ".codex", "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	text = string(instructions)
	if strings.Contains(text, "setupmjr-subagent-deepseek:start") || !strings.Contains(text, "setupmjr-subagent-agy:start") {
		t.Fatalf("switching back to agy did not replace DeepSeek block:\n%s", text)
	}
}

func TestSwitchCodexProviderSnapshotsConfigurationsForFastSwitching(t *testing.T) {
	home := t.TempDir()
	codexHome := filepath.Join(home, ".codex")
	if err := os.MkdirAll(codexHome, 0700); err != nil {
		t.Fatal(err)
	}
	openAIConfig := "model = \"gpt-5.6-sol\"\nmodel_provider = \"openai\"\n[mcp_servers.example]\ncommand = \"example\"\n"
	if err := os.WriteFile(filepath.Join(codexHome, "config.toml"), []byte(openAIConfig), 0600); err != nil {
		t.Fatal(err)
	}

	originalSetup := runDeepSeekSetup
	defer func() { runDeepSeekSetup = originalSetup }()
	calls := 0
	runDeepSeekSetup = func(_ string, targetHome, action string) error {
		calls++
		if action != "1" {
			t.Fatalf("unexpected installer action %q", action)
		}
		if err := os.WriteFile(filepath.Join(targetHome, "config.toml"), []byte("model = \"deepseek-flash\"\nmodel_provider = \"deepseek\"\n"), 0600); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(targetHome, "models.json"), []byte("{\"models\":[]}\n"), 0600)
	}

	if err := switchCodexProvider(home, "deepseek"); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("installer calls = %d, want 1", calls)
	}
	if provider, _ := currentCodexProvider(home); provider != "deepseek" {
		t.Fatalf("provider after DeepSeek switch = %q", provider)
	}

	if err := switchCodexProvider(home, "openai"); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("OpenAI restore reran installer; calls = %d", calls)
	}
	gotConfig, err := os.ReadFile(filepath.Join(codexHome, "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(gotConfig) != openAIConfig {
		t.Fatalf("OpenAI config not restored exactly:\n%s", gotConfig)
	}
	if _, err := os.Stat(filepath.Join(codexHome, "models.json")); !os.IsNotExist(err) {
		t.Fatalf("DeepSeek models.json remained after OpenAI restore: %v", err)
	}

	if err := switchCodexProvider(home, "deepseek"); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("cached DeepSeek switch reran installer; calls = %d", calls)
	}
}

func TestSwitchCodexProviderCanAdoptExistingDeepSeekInstallation(t *testing.T) {
	home := t.TempDir()
	codexHome := filepath.Join(home, ".codex")
	if err := os.MkdirAll(codexHome, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(codexHome, "config.toml"), []byte("model_provider = \"deepseek\"\nmodel = \"deepseek-flash\"\nexperimental_bearer_token = \"sk-test\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(codexHome, "models.json"), []byte("{}\n"), 0600); err != nil {
		t.Fatal(err)
	}

	originalSetup := runDeepSeekSetup
	defer func() { runDeepSeekSetup = originalSetup }()
	calls := 0
	runDeepSeekSetup = func(_ string, targetHome, action string) error {
		calls++
		if action != "9" {
			t.Fatalf("unexpected installer action %q", action)
		}
		if err := os.WriteFile(filepath.Join(targetHome, "config.toml"), []byte("model = \"gpt-5.6-sol\"\nmodel_provider = \"openai\"\n"), 0600); err != nil {
			return err
		}
		return os.Remove(filepath.Join(targetHome, "models.json"))
	}

	if err := switchCodexProvider(home, "openai"); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("restore installer calls = %d, want 1", calls)
	}
	if provider, _ := currentCodexProvider(home); provider != "openai" {
		t.Fatalf("provider after restore = %q", provider)
	}

	if err := switchCodexProvider(home, "deepseek"); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("saved DeepSeek snapshot was not reused; calls = %d", calls)
	}
	if provider, _ := currentCodexProvider(home); provider != "deepseek" {
		t.Fatalf("provider after cached DeepSeek restore = %q", provider)
	}
}

func TestUpdateManagedBlockRejectsMalformedMarkers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "AGENTS.md")
	if err := os.WriteFile(path, []byte("<!-- start -->\nunterminated\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := updateManagedBlock(path, "<!-- start -->", "<!-- end -->", "new"); err == nil {
		t.Fatal("malformed managed block unexpectedly succeeded")
	}
}
