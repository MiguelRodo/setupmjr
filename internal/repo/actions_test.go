package repo

import (
	"strings"
	"testing"
)

func TestGeneratedPrebuildWorkflowUsesSupportedVersionInputs(t *testing.T) {
	workflow := actionWorkflows["prebuild-devcontainer"]

	for _, want := range []string{
		"version:\n        description:",
		"bump_type:\n        description:",
		"version_force:\n        description:",
		"version: ${{ inputs.version }}",
		"bump_type: ${{ inputs.bump_type != 'none' && inputs.bump_type || '' }}",
		"version_force: ${{ inputs.version_force }}",
		"uses: MiguelRodo/actions/prebuild-devcontainer@v3",
	} {
		if !strings.Contains(workflow, want) {
			t.Fatalf("generated prebuild workflow missing %q", want)
		}
	}

	if strings.Contains(workflow, "\n      tag:") || strings.Contains(workflow, "\n          tag:") {
		t.Fatal("generated prebuild workflow still uses deprecated tag input")
	}
}

func TestGeneratedReleaseWorkflowsUseConsistentManualVersionControls(t *testing.T) {
	for _, name := range []string{"version-release", "go-version-release", "r-version-release"} {
		workflow := actionWorkflows[name]
		for _, want := range []string{
			"type: string",
			"type: choice\n        default: none\n        options:\n          - none\n          - patch\n          - minor\n          - major",
			"version_force:\n        description:",
			"type: boolean\n        default: false",
			"bump_type: ${{ inputs.bump_type != 'none' && inputs.bump_type || '' }}",
			"version_force: ${{ inputs.version_force }}",
			"uses: MiguelRodo/actions/" + name + "@v3",
		} {
			if !strings.Contains(workflow, want) {
				t.Fatalf("generated %s workflow missing %q", name, want)
			}
		}
	}
}

func TestGeneratedGoReleaseKeepsCredentialsOutOfManualInputs(t *testing.T) {
	workflow := actionWorkflows["go-version-release"]
	manualInputs := strings.SplitN(workflow, "jobs:", 2)[0]

	if strings.Contains(manualInputs, "apt_repo_token:") {
		t.Fatal("generated Go release exposes apt_repo_token as a manual input")
	}
	for _, want := range []string{
		"apt_repo_token: ${{ secrets.APT_REPO_TOKEN }}",
		"apt_signing_key: ${{ secrets.APT_SIGNING_KEY }}",
		"apt_signing_key_passphrase: ${{ secrets.APT_SIGNING_KEY_PASSPHRASE }}",
		"scoop_token: ${{ secrets.SCOOP_TAP_TOKEN }}",
		"homebrew_token: ${{ secrets.HOMEBREW_TAP_TOKEN }}",
	} {
		if !strings.Contains(workflow, want) {
			t.Fatalf("generated Go release workflow missing %q", want)
		}
	}
}

func TestGeneratedAPTPruneUsesRetentionChoice(t *testing.T) {
	workflow := actionWorkflows["apt-repo-prune"]
	if !strings.Contains(workflow, "retention:\n        description:") {
		t.Fatal("generated APT prune workflow missing retention input")
	}
	for _, want := range []string{
		"type: choice\n        default: latest-per-major",
		"- latest-per-major\n          - latest-per-minor\n          - latest",
		"uses: MiguelRodo/actions/apt-repo-prune@v3",
	} {
		if !strings.Contains(workflow, want) {
			t.Fatalf("generated APT prune workflow missing %q", want)
		}
	}
}
