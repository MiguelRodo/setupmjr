package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestHandleProjectPjConsumesV0ReleaseLine(t *testing.T) {
	originalRunner := runExternalCommand
	defer func() { runExternalCommand = originalRunner }()

	var calls [][]string
	runExternalCommand = func(name string, args ...string) error {
		calls = append(calls, append([]string{name}, args...))
		return nil
	}

	if err := handleProject([]string{"--pj"}); err != nil {
		t.Fatalf("handleProject failed: %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("expected 2 calls, got %d: %#v", len(calls), calls)
	}

	wantClonePrefix := []string{
		"git", "clone", "--depth", "1",
		"https://github.com/MiguelRodo/pj.git",
		"--branch", "v0", "--single-branch",
	}
	if len(calls[0]) < len(wantClonePrefix)+1 || !reflect.DeepEqual(calls[0][:len(wantClonePrefix)], wantClonePrefix) {
		t.Fatalf("pj clone does not consume the v0 release line: %#v", calls[0])
	}
	if calls[1][0] != "bash" || !strings.HasSuffix(calls[1][1], "install.sh") {
		t.Fatalf("unexpected pj install call: %#v", calls[1])
	}
}
