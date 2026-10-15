package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func doctorTestEnvironment(t *testing.T) string {
	t.Helper()
	for _, name := range []string{"CODEX_HOME", "COPILOT_PROVIDER_TYPE", "COPILOT_PROVIDER_BASE_URL", "COPILOT_PROVIDER_API_KEY", "COPILOT_GITHUB_TOKEN", "GH_TOKEN", "GITHUB_TOKEN", "DEEPSEEK_API_KEY"} {
		t.Setenv(name, "")
		if err := os.Unsetenv(name); err != nil {
			t.Fatal(err)
		}
	}
	oldLook, oldRun := agentDoctorLookPath, agentDoctorRun
	t.Cleanup(func() { agentDoctorLookPath, agentDoctorRun = oldLook, oldRun })
	agentDoctorLookPath = func(name string) (string, error) { return filepath.Join("bin", name), nil }
	agentDoctorRun = func(path string, args ...string) ([]byte, error) {
		switch strings.Join(args, " ") {
		case "--version":
			return []byte("GitHub Copilot CLI 1.2.3-beta.1\nSECRET_VERSION_OUTPUT"), nil
		case "auth status", "login status":
			return []byte("SECRET_AUTH_OUTPUT"), nil
		default:
			t.Fatalf("unsafe probe: %s %v", path, args)
			return nil, nil
		}
	}
	return t.TempDir()
}

func TestAgentDoctorReadOnlyAndRedacted(t *testing.T) {
	home := doctorTestEnvironment(t)
	var out bytes.Buffer
	if err := runAgentDoctor(home, &out); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"FOUND: claude", "FOUND: codex", "FOUND: copilot", "1.2.3-beta.1", "Codex=openai; Copilot=github; subagent=none", "Copilot authentication unverified", "0 failed"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("missing %q in %s", want, &out)
		}
	}
	if strings.Contains(out.String(), "SECRET") {
		t.Fatal("probe output leaked")
	}
	entries, err := os.ReadDir(home)
	if err != nil || len(entries) != 0 {
		t.Fatalf("doctor changed home: %v, %v", entries, err)
	}
}

func TestAgentDoctorMissingPATH(t *testing.T) {
	home := doctorTestEnvironment(t)
	t.Setenv("PATH", t.TempDir())
	agentDoctorLookPath = exec.LookPath
	agentDoctorRun = func(string, ...string) ([]byte, error) {
		t.Fatal("ran a probe with no executable on PATH")
		return nil, nil
	}
	var out bytes.Buffer
	if err := runAgentDoctor(home, &out); err == nil || !strings.Contains(err.Error(), "3 failed") {
		t.Fatalf("expected all missing executables: %v\n%s", err, &out)
	}
}

func TestAgentDoctorAggregatesFailures(t *testing.T) {
	home := doctorTestEnvironment(t)
	agentDoctorLookPath = func(name string) (string, error) {
		if name == "claude" {
			return "", errors.New("SECRET_PATH_ERROR")
		}
		return name, nil
	}
	agentDoctorRun = func(path string, args ...string) ([]byte, error) {
		if path == "copilot" {
			return []byte("copilot version: v1.34.1"), nil
		} // AWS name collision
		return []byte("SECRET_EXEC_OUTPUT"), errors.New("SECRET_EXEC_ERROR")
	}
	var out bytes.Buffer
	err := runAgentDoctor(home, &out)
	if err == nil || !strings.Contains(err.Error(), "3 failed") {
		t.Fatalf("expected aggregated failures: %v\n%s", err, &out)
	}
	for _, want := range []string{"setupmjr install claude", "setupmjr install codex", "AWS Copilot/name collision"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("missing fix %q: %s", want, &out)
		}
	}
	if strings.Contains(out.String(), "SECRET") {
		t.Fatal("failure details leaked")
	}
}

func TestAgentDoctorAuthenticationFailures(t *testing.T) {
	home := doctorTestEnvironment(t)
	goodRun := agentDoctorRun
	agentDoctorRun = func(path string, args ...string) ([]byte, error) {
		if args[0] != "--version" {
			return []byte("SECRET"), errors.New("SECRET")
		}
		return goodRun(path, args...)
	}
	t.Setenv("COPILOT_GITHUB_TOKEN", "ghp_SECRET")
	var out bytes.Buffer
	if err := runAgentDoctor(home, &out); err == nil || !strings.Contains(err.Error(), "3 failed") {
		t.Fatalf("unexpected result: %v\n%s", err, &out)
	}
	if strings.Contains(out.String(), "SECRET") {
		t.Fatal("credential leaked")
	}
}

func TestAgentDoctorManagedConfiguration(t *testing.T) {
	home := doctorTestEnvironment(t)
	files := map[string]string{
		".codex/config.toml":                 "model_provider = \"deepseek\"\nexperimental_bearer_token = \"SECRET\"\n",
		".codex/AGENTS.md":                   "<!-- setupmjr-subagent-agy:start -->\n",
		".config/setupmjr/agent/copilot.env": "export COPILOT_PROVIDER_BASE_URL='https://api.deepseek.com/anthropic'\n",
	}
	for rel, content := range files {
		path := filepath.Join(home, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	goodRun := agentDoctorRun
	agentDoctorRun = func(path string, args ...string) ([]byte, error) {
		if args[0] == "login" {
			t.Fatal("OpenAI login probed for DeepSeek")
		}
		return goodRun(path, args...)
	}
	agentDoctorLookPath = func(name string) (string, error) {
		if name == "agy" {
			return "", os.ErrNotExist
		}
		return name, nil
	}
	var out bytes.Buffer
	if err := runAgentDoctor(home, &out); err == nil || !strings.Contains(err.Error(), "3 failed") {
		t.Fatalf("unexpected result: %v\n%s", err, &out)
	}
	for _, want := range []string{"Codex=deepseek; Copilot=deepseek; subagent=agy", "DeepSeek Codex credential present", "--auth deepseek", "environment is not active", "agy subagent is not on PATH"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("missing %q: %s", want, &out)
		}
	}
	for rel, want := range files {
		got, err := os.ReadFile(filepath.Join(home, rel))
		if err != nil || string(got) != want {
			t.Errorf("doctor modified %s", rel)
		}
	}
	if strings.Contains(out.String(), "SECRET") {
		t.Fatal("config credential leaked")
	}
}

func TestAgentDoctorCodexHomeOverride(t *testing.T) {
	home := doctorTestEnvironment(t)
	t.Setenv("CODEX_HOME", t.TempDir())
	if err := os.WriteFile(filepath.Join(os.Getenv("CODEX_HOME"), "config.toml"), []byte("model_provider = \"deepseek\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := runAgentDoctor(home, &out); err == nil {
		t.Fatal("missing active credential not detected")
	}
	if !strings.Contains(out.String(), "CODEX_HOME overrides") || !strings.Contains(out.String(), "Active DeepSeek Codex") {
		t.Fatal(out.String())
	}
}

func TestAgentDoctorRejectsMutatingOptions(t *testing.T) {
	for _, args := range [][]string{{"doctor", "--auth", "deepseek"}, {"doctor", "--codex-provider", "deepseek"}, {"doctor", "--subagent"}, {"doctor", "unknown"}} {
		if err := handleAgent(args); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}

func TestAgentDoctorProbeProcess(t *testing.T) {
	mode := os.Getenv("SETUPMJR_DOCTOR_TEST_PROCESS")
	if mode == "" {
		return
	}
	if mode == "hang" {
		time.Sleep(30 * time.Second)
		os.Exit(0)
	}
	input, _ := io.ReadAll(os.Stdin)
	if len(input) != 0 {
		os.Exit(2)
	}
	fmt.Fprint(os.Stdout, strings.Repeat("x", 70*1024))
	fmt.Fprint(os.Stderr, "SECRET_STDERR")
	os.Exit(0)
}

func TestAgentDoctorProbeBounds(t *testing.T) {
	path, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("SETUPMJR_DOCTOR_TEST_PROCESS", "output")
	output, err := runAgentDoctorProbe(path, "-test.run=^TestAgentDoctorProbeProcess$")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(output, bytes.Repeat([]byte("x"), 64*1024)) {
		t.Fatal("output not bounded or stderr leaked")
	}
	t.Setenv("SETUPMJR_DOCTOR_TEST_PROCESS", "hang")
	start := time.Now()
	if _, err := runAgentDoctorProbe(path, "-test.run=^TestAgentDoctorProbeProcess$"); err == nil {
		t.Fatal("hung probe succeeded")
	}
	if time.Since(start) > 15*time.Second {
		t.Fatal("probe timeout not enforced")
	}
}
