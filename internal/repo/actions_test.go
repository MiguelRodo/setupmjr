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
		"uses: MiguelRodo/actions/prebuild-devcontainer@v2",
	} {
		if !strings.Contains(workflow, want) {
			t.Fatalf("generated prebuild workflow missing %q", want)
		}
	}

	if strings.Contains(workflow, "\n      tag:") || strings.Contains(workflow, "\n          tag:") {
		t.Fatal("generated prebuild workflow still uses deprecated tag input")
	}
}
