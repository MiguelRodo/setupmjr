package main

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPlainCopilotAllProviderSwitches(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Bash/Zsh integration is exercised in Linux/WSL, not native PowerShell")
	}
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("Bash unavailable")
	}
	home := t.TempDir()
	t.Setenv("DEEPSEEK_API_KEY", "fake-deepseek-key")
	for _, provider := range []string{"github", "deepseek", "openai"} {
		if err := switchCopilotProvider(home, provider); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(copilotEnvPath(home))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "fake-deepseek-key") {
			t.Fatal("selection stored a literal credential")
		}
		if err := os.WriteFile(filepath.Join(home, provider+".env"), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	bin := filepath.Join(home, "bin")
	if err := os.Mkdir(bin, 0700); err != nil {
		t.Fatal(err)
	}
	fake := `#!/bin/sh
set -eu
for expected do :; done
test "${COPILOT_PROVIDERS_CONFIG:-}" = /dev/null
test -z "${COPILOT_PROVIDER_BEARER_TOKEN:-}${COPILOT_PROVIDER_API_KEY_COMMAND:-}${COPILOT_PROVIDER_HEADERS:-}${COPILOT_PROVIDER_TRANSPORT:-}${OPENAI_ORG_ID:-}${OPENAI_PROJECT_ID:-}${OPENAI_BASE_URL:-}"
test -z "${OPENAI_API_KEY:-}${OPENAI_KEY:-}${DEEPSEEK_API_KEY:-}"
case "$expected" in
providers|--help)
    test -z "${COPILOT_PROVIDER_API_KEY:-}"
    if [ "${SETUPMJR_FAKE_UNSUPPORTED:-}" = 1 ]; then printf unsupported; else
        printf '%s\n' '--model --reasoning-effort --no-auto-update COPILOT_PROVIDER_TYPE COPILOT_PROVIDER_BASE_URL COPILOT_PROVIDER_API_KEY COPILOT_PROVIDER_WIRE_API responses'
    fi
    exit 0 ;;
esac
case "$expected" in
openai)
    test "$COPILOT_PROVIDER_TYPE" = openai
    test "$COPILOT_PROVIDER_BASE_URL" = https://api.openai.com/v1
    test "$COPILOT_PROVIDER_WIRE_API" = responses
    test "$COPILOT_PROVIDER_API_KEY" = fake-openai-key
    test "$COPILOT_MODEL" = gpt-6.1-sol
    test "$2" = --model && test "$3" = gpt-6.1-sol
    test "$4" = --reasoning-effort && test "$5" = medium
    ;;
deepseek)
    test "$COPILOT_PROVIDER_TYPE" = anthropic
    test "$COPILOT_PROVIDER_BASE_URL" = https://api.deepseek.com/anthropic
    test "$COPILOT_PROVIDER_API_KEY" = fake-deepseek-key
    test "$COPILOT_MODEL" = deepseek-flash
    test -z "${COPILOT_PROVIDER_WIRE_API:-}"
    ;;
github)
    test -z "${COPILOT_PROVIDER_TYPE:-}${COPILOT_PROVIDER_BASE_URL:-}${COPILOT_PROVIDER_API_KEY:-}${COPILOT_MODEL:-}${COPILOT_PROVIDER_WIRE_API:-}"
    ;;
*) exit 31 ;;
esac
printf 'OK %s\n' "$expected"
`
	if err := os.WriteFile(filepath.Join(bin, "copilot"), []byte(fake), 0700); err != nil {
		t.Fatal(err)
	}
	script := `set -eu
. "$HOME/.bashrc"
. "$HOME/.config/setupmjr/agent/copilot.sh"
test "$(type -t copilot)" = function
for from in github deepseek openai; do
    cp "$HOME/$from.env" "$HOME/.config/setupmjr/agent/copilot.env"
    copilot "$from"
    for to in github deepseek openai; do
        cp "$HOME/$to.env" "$HOME/.config/setupmjr/agent/copilot.env"
        copilot "$to"
    done
done
test "$COPILOT_PROVIDER_BASE_URL" = parent-stale-endpoint
cp "$HOME/openai.env" "$HOME/.config/setupmjr/agent/copilot.env"
set -x
copilot openai
set +x
export SETUPMJR_FAKE_UNSUPPORTED=1
if copilot openai; then exit 35; fi
unset SETUPMJR_FAKE_UNSUPPORTED
unset OPENAI_API_KEY OPENAI_KEY
if copilot openai; then exit 32; fi
cp "$HOME/deepseek.env" "$HOME/.config/setupmjr/agent/copilot.env"
unset DEEPSEEK_API_KEY
if copilot deepseek; then exit 33; fi
: > "$HOME/.config/setupmjr/agent/copilot.env"
export SETUPMJR_COPILOT_ROUTE=github
if copilot github; then exit 34; fi
`
	cmd := exec.Command(bash, "--noprofile", "--norc", "-c", script)
	cmd.Env = append(copilotShellTestEnvironment(), "HOME="+home, "PATH="+bin+":"+os.Getenv("PATH"),
		"OPENAI_KEY=fake-openai-key", "DEEPSEEK_API_KEY=fake-deepseek-key", "COPILOT_PROVIDER_BASE_URL=parent-stale-endpoint",
		"COPILOT_PROVIDER_API_KEY=parent-stale-key", "COPILOT_PROVIDER_BEARER_TOKEN=stale", "COPILOT_PROVIDER_API_KEY_COMMAND=stale",
		"COPILOT_PROVIDER_HEADERS=stale", "COPILOT_PROVIDER_TRANSPORT=websockets", "COPILOT_PROVIDERS_CONFIG=stale",
		"OPENAI_ORG_ID=stale", "OPENAI_PROJECT_ID=stale", "OPENAI_BASE_URL=stale")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("plain copilot shell check failed: %v\n%s", err, output)
	}
	if strings.Count(string(output), "OK ") != 13 {
		t.Fatalf("unexpected launch count; missing credentials must never reach the binary:\n%s", output)
	}
	if strings.Contains(string(output), "fake-openai-key") || strings.Contains(string(output), "fake-deepseek-key") || strings.Contains(string(output), "parent-stale-key") {
		t.Fatal("traced launch disclosed credential material")
	}
	for _, custom := range []string{"alias copilot='printf user-alias'", "function copilot { printf user-function; }"} {
		cmd := exec.Command(bash, "--noprofile", "--norc", "-c", custom+"\n. \"$HOME/.config/setupmjr/agent/copilot.sh\"\n"+"alias copilot 2>/dev/null || copilot")
		cmd.Env = append(os.Environ(), "HOME="+home)
		output, err := cmd.CombinedOutput()
		if err != nil || !strings.Contains(string(output), "user-") {
			t.Fatal("overwrote an existing copilot alias or function")
		}
	}
}

func copilotShellTestEnvironment() []string {
	var env []string
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(name, "COPILOT_PROVIDER") || strings.HasPrefix(name, "OPENAI_") || name == "DEEPSEEK_API_KEY" || name == "SETUPMJR_COPILOT_WRAPPER" || name == "SETUPMJR_COPILOT_ROUTE" || name == "SETUPMJR_FAKE_UNSUPPORTED" {
			continue
		}
		env = append(env, entry)
	}
	return env
}

func TestCopilotStatusDoesNotExposeCredential(t *testing.T) {
	home := t.TempDir()
	t.Setenv("OPENAI_KEY", "fake-status-key-must-not-be-printed")
	if err := switchCopilotProvider(home, "openai"); err != nil {
		t.Fatal(err)
	}
	previous := os.Stdout
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = write
	printCopilotStatus(home)
	os.Stdout = previous
	_ = write.Close()
	output, _ := io.ReadAll(read)
	_ = read.Close()
	for _, want := range []string{"openai (configured route)", "https://api.openai.com/v1", "responses", "gpt-6.1-sol", "API key: present", "not a verified request"} {
		if !strings.Contains(string(output), want) {
			t.Fatalf("status missing %q", want)
		}
	}
	if strings.Contains(string(output), "fake-status-key") {
		t.Fatal("status disclosed a credential")
	}
}

func TestCopilotManagedFileRejectsSymlinkAndUsesPrivateMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permissions and symlinks")
	}
	home := t.TempDir()
	if err := configureCopilotOpenAI(home); err != nil {
		t.Fatal(err)
	}
	path := copilotEnvPath(home)
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("selection must be private")
	}
	target := filepath.Join(home, "unrelated")
	if err := os.WriteFile(target, []byte("preserve"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
	if err := configureCopilotOpenAI(home); err == nil {
		t.Fatal("followed selection symlink")
	}
	data, err := os.ReadFile(target)
	if err != nil || string(data) != "preserve" {
		t.Fatal("modified unrelated symlink target")
	}
}
