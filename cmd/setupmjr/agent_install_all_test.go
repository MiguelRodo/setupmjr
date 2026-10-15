package main

import (
	"bytes"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestInstallCodingAgents(t *testing.T) {
	for _, goos := range []string{"linux", "darwin", "windows"} {
		for _, scenario := range []string{"missing", "installed", "mixed", "failure"} {
			t.Run(goos+"/"+scenario, func(t *testing.T) {
				oldLook, oldDownload, oldRun := codingAgentLookPath, codingAgentDownload, codingAgentRun
				t.Cleanup(func() { codingAgentLookPath, codingAgentDownload, codingAgentRun = oldLook, oldDownload, oldRun })
				installed := map[string]bool{}
				if scenario == "installed" {
					installed = map[string]bool{"claude": true, "codex": true, "copilot": true}
				} else if scenario == "mixed" {
					installed["codex"] = true
				}
				codingAgentLookPath = func(name string) (string, error) {
					switch name {
					case "claude", "codex", "copilot":
						if !installed[name] {
							return "", os.ErrNotExist
						}
					}
					return "/tools/" + name, nil
				}
				var downloads, wantDownloads []string
				codingAgentDownload = func(url, path string) error {
					downloads = append(downloads, url)
					return os.WriteFile(path, []byte("vendor installer"), 0600)
				}
				var commands, wantCommands []string
				installErr := errors.New("vendor installer failed")
				codingAgentRun = func(name string, args ...string) error {
					commands = append(commands, name)
					if scenario == "failure" && len(commands) <= 2 {
						return installErr
					}
					return nil
				}
				var out bytes.Buffer
				err := installCodingAgentsForOS(goos, &out)
				if scenario == "failure" {
					if !errors.Is(err, installErr) {
						t.Fatalf("error = %v; want vendor error", err)
					}
					for _, agent := range []string{"claude", "codex"} {
						if !strings.Contains(err.Error(), agent) {
							t.Fatalf("combined error omits %s: %v", agent, err)
						}
					}
				} else if err != nil {
					t.Fatal(err)
				}
				for i, agent := range []string{"claude", "codex", "copilot"} {
					status := "installer completed."
					if installed[agent] {
						status = "already installed at /tools/" + agent + "; skipped."
					} else {
						plan, err := codingAgentInstallPlanFor(agent, goos)
						if err != nil {
							t.Fatal(err)
						}
						wantCommands = append(wantCommands, plan.command)
						if plan.url != "" {
							wantDownloads = append(wantDownloads, plan.url)
						}
						if scenario == "failure" && i < 2 {
							status = "failed:"
						}
					}
					if !strings.Contains(out.String(), agent+": "+status) {
						t.Fatalf("missing %s result %q in %q", agent, status, out.String())
					}
				}
				if !reflect.DeepEqual(commands, wantCommands) || !reflect.DeepEqual(downloads, wantDownloads) {
					t.Fatalf("commands = %v, downloads = %v; want %v, %v", commands, downloads, wantCommands, wantDownloads)
				}
			})
		}
	}
}

func TestHandleInstallAgents(t *testing.T) {
	oldLook, oldDownload, oldRun := codingAgentLookPath, codingAgentDownload, codingAgentRun
	t.Cleanup(func() { codingAgentLookPath, codingAgentDownload, codingAgentRun = oldLook, oldDownload, oldRun })
	var checked []string
	codingAgentLookPath = func(name string) (string, error) {
		checked = append(checked, name)
		return "/tools/" + name, nil
	}
	codingAgentDownload = func(string, string) error { t.Fatal("downloaded installed agent"); return nil }
	codingAgentRun = func(string, ...string) error { t.Fatal("ran installer for installed agent"); return nil }
	if err := handleInstall([]string{"agents"}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(checked, []string{"claude", "codex", "copilot"}) {
		t.Fatalf("checked = %v", checked)
	}
	for _, args := range [][]string{{"agents", "--pj"}, {"agents", "claude"}} {
		if err := handleInstall(args); err == nil {
			t.Fatalf("expected error for install %v", args)
		}
	}
}
