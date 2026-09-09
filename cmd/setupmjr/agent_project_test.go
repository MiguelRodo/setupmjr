package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

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

func TestHandleProjectGhSkillUsesUniversalUserScope(t *testing.T) {
	originalRunner := runExternalCommand
	defer func() { runExternalCommand = originalRunner }()

	var gotName string
	var gotArgs []string
	runExternalCommand = func(name string, args ...string) error {
		gotName = name
		gotArgs = append([]string(nil), args...)
		return nil
	}

	if err := handleProject([]string{"--gh-skill"}); err != nil {
		t.Fatalf("handleProject failed: %v", err)
	}

	if gotName != "gh" {
		t.Fatalf("command = %q, want gh", gotName)
	}
	wantArgs := []string{
		"skill", "install",
		"MiguelRodo/github-projects-skill", "github-projects",
		"--agent", "universal",
		"--scope", "user",
		"--force",
	}
	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Fatalf("args = %#v, want %#v", gotArgs, wantArgs)
	}
}

func TestProjectFlagIsLowerCase(t *testing.T) {
	if err := handleProject([]string{"--GH-skill"}); err == nil {
		t.Fatal("upper-case --GH-skill unexpectedly succeeded")
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
