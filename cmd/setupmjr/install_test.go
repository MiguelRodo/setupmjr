package main

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestHandleInstallDispatchesAllTargetsInOrder(t *testing.T) {
	originalInstallRepos := installRepos
	originalRunner := runExternalCommand
	defer func() {
		installRepos = originalInstallRepos
		runExternalCommand = originalRunner
	}()

	var calls []string
	installRepos = func() error {
		calls = append(calls, "repos")
		return nil
	}
	runExternalCommand = func(name string, args ...string) error {
		switch name {
		case "gh":
			calls = append(calls, "gh-skill")
			want := []string{
				"skill", "install",
				"MiguelRodo/github-projects-skill", "github-projects",
				"--agent", "universal",
				"--scope", "user",
				"--force",
				"--pin", "main",
			}
			if !reflect.DeepEqual(args, want) {
				t.Fatalf("gh args = %#v, want %#v", args, want)
			}
		case "git":
			calls = append(calls, "pj-clone")
			joined := strings.Join(args, " ")
			if !strings.Contains(joined, "https://github.com/MiguelRodo/pj.git") || !strings.Contains(joined, "--branch v0") {
				t.Fatalf("pj clone did not preserve canonical v0 release semantics: %q", joined)
			}
		case "bash":
			calls = append(calls, "pj-install")
		default:
			t.Fatalf("unexpected command %q", name)
		}
		return nil
	}

	if err := handleInstall([]string{"--pj", "--repos", "--gh-skill"}); err != nil {
		t.Fatalf("handleInstall failed: %v", err)
	}

	want := []string{"repos", "gh-skill", "pj-clone", "pj-install"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls = %#v, want %#v", calls, want)
	}
}

func TestHandleInstallStopsOnFirstError(t *testing.T) {
	originalInstallRepos := installRepos
	originalRunner := runExternalCommand
	defer func() {
		installRepos = originalInstallRepos
		runExternalCommand = originalRunner
	}()

	wantErr := errors.New("repos failed")
	installRepos = func() error { return wantErr }
	calledExternal := false
	runExternalCommand = func(string, ...string) error {
		calledExternal = true
		return nil
	}

	err := handleInstall([]string{"--repos", "--pj", "--gh-skill"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want wrapped %v", err, wantErr)
	}
	if calledExternal {
		t.Fatal("later installers ran after repos failed")
	}
}

func TestHandleInstallRequiresTargetAndRejectsPositionals(t *testing.T) {
	if err := handleInstall(nil); err == nil {
		t.Fatal("install without a target unexpectedly succeeded")
	}
	if err := handleInstall([]string{"repos"}); err == nil {
		t.Fatal("install positional target unexpectedly succeeded")
	}
	if err := handleInstall([]string{"--PJ"}); err == nil {
		t.Fatal("upper-case install flag unexpectedly succeeded")
	}
}

func TestRepoInstallReposIsRemoved(t *testing.T) {
	if err := handleRepo([]string{"install", "repos"}); err == nil {
		t.Fatal("repo install repos unexpectedly succeeded")
	}
}
