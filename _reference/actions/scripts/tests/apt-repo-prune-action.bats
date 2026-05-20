#!/usr/bin/env bats

# Structural tests for apt-repo-prune/action.yml

ACTION_FILE="$(cd "$(dirname "$BATS_TEST_FILENAME")/../.." && pwd)/apt-repo-prune/action.yml"
SELECT_SCRIPT="$(cd "$(dirname "$BATS_TEST_FILENAME")/.." && pwd)/apt-prune-select-versions.sh"

@test "apt-repo-prune action.yml exists" {
  [ -f "$ACTION_FILE" ]
}

@test "apt-repo-prune is a composite action" {
  run grep -F 'using: "composite"' "$ACTION_FILE"
  [ "$status" -eq 0 ]
}

@test "apt-repo-prune validates Linux-only runner" {
  run grep -F 'apt-repo-prune supports Linux runners only' "$ACTION_FILE"
  [ "$status" -eq 0 ]
}

@test "apt-repo-prune does NOT have repo input" {
  run grep -F 'repo:' "$ACTION_FILE"
  [ "$status" -ne 0 ]
}

@test "apt-repo-prune has required input: token" {
  run grep -F 'token:' "$ACTION_FILE"
  [ "$status" -eq 0 ]
  run awk '
    /^  token:$/ { in_block=1; next }
    in_block && /required: true/ { found=1; exit 0 }
    in_block && /^  [^ ]/ { exit 1 }
    END { exit found ? 0 : 1 }
  ' "$ACTION_FILE"
  [ "$status" -eq 0 ]
}

@test "apt-repo-prune has optional input: retention with default latest-per-major" {
  run grep -F 'retention:' "$ACTION_FILE"
  [ "$status" -eq 0 ]
  run awk '
    /^  retention:$/ { in_block=1; next }
    in_block && /default: "latest-per-major"/ { found=1; exit 0 }
    in_block && /^  [^ ]/ { exit 1 }
    END { exit found ? 0 : 1 }
  ' "$ACTION_FILE"
  [ "$status" -eq 0 ]
}

@test "apt-repo-prune has optional input: apt_signing_key" {
  run grep -F 'apt_signing_key:' "$ACTION_FILE"
  [ "$status" -eq 0 ]
}

@test "apt-repo-prune has optional input: apt_signing_key_passphrase" {
  run grep -F 'apt_signing_key_passphrase:' "$ACTION_FILE"
  [ "$status" -eq 0 ]
}

@test "apt-repo-prune checks pool/main and exits 0 (not 1) when absent" {
  run grep -F 'pool/main' "$ACTION_FILE"
  [ "$status" -eq 0 ]
  # When pool/main is absent the repo has no packages: must use exit 0, not exit 1.
  run awk '
    /pool\/main/ { in_block=1; next }
    in_block && /exit 1/ { bad=1; exit 0 }
    in_block && /exit 0/ { good=1; in_block=0; next }
    END { exit (good && !bad) ? 0 : 1 }
  ' "$ACTION_FILE"
  [ "$status" -eq 0 ]
}

@test "apt-repo-prune validates expected dists/stable structure" {
  run grep -F 'dists/stable' "$ACTION_FILE"
  [ "$status" -eq 0 ]
}

@test "apt-repo-prune uses git filter-repo for history rewrite" {
  run grep -F 'git filter-repo' "$ACTION_FILE"
  [ "$status" -eq 0 ]
}

@test "apt-repo-prune uses --invert-paths for removal" {
  run grep -F -- '--invert-paths' "$ACTION_FILE"
  [ "$status" -eq 0 ]
}

@test "apt-repo-prune force-pushes the result" {
  run grep -F 'push --force origin' "$ACTION_FILE"
  [ "$status" -eq 0 ]
}

@test "apt-repo-prune calls the apt-prune-select-versions.sh script" {
  run grep -F 'apt-prune-select-versions.sh' "$ACTION_FILE"
  [ "$status" -eq 0 ]
}

@test "apt-prune-select-versions.sh script exists" {
  [ -f "$SELECT_SCRIPT" ]
}

@test "apt-prune-select-versions.sh has valid bash syntax" {
  bash -n "$SELECT_SCRIPT"
}

@test "apt-repo-prune uses authenticated remote URL with x-access-token" {
  run grep -F 'x-access-token' "$ACTION_FILE"
  [ "$status" -eq 0 ]
}

@test "apt-repo-prune uses github.repository context instead of repo input" {
  run grep -F 'GITHUB_REPOSITORY' "$ACTION_FILE"
  [ "$status" -eq 0 ]
}

@test "apt-repo-prune uses default_branch context for branch, not hardcoded main" {
  run grep -F 'github.event.repository.default_branch' "$ACTION_FILE"
  [ "$status" -eq 0 ]
  run grep -F 'GITHUB_REF_NAME' "$ACTION_FILE"
  [ "$status" -eq 0 ]
  run grep -F -- '--branch main' "$ACTION_FILE"
  [ "$status" -ne 0 ]
}

@test "apt-repo-prune creates passphrase file only when passphrase is non-empty" {
  run awk '
    /if \[ -n "\$APT_SIGNING_KEY_PASSPHRASE" \]/ { in_block=1; next }
    in_block && /^[[:space:]]*fi([[:space:]]|;|$)/ { in_block=0; next }
    in_block && /gpg-passphrase/ { found=1; exit 0 }
    END { exit found ? 0 : 1 }
  ' "$ACTION_FILE"
  [ "$status" -eq 0 ]
}

@test "apt-repo-prune uses passphrase-file option only when GPG_PASSPHRASE_FILE is set" {
  run awk '
    /if \[ -n "\$GPG_PASSPHRASE_FILE" \]/ { in_block=1; next }
    in_block && /^[[:space:]]*fi([[:space:]]|;|$)/ { in_block=0; next }
    in_block && /--passphrase-file/ { found=1; exit 0 }
    END { exit found ? 0 : 1 }
  ' "$ACTION_FILE"
  [ "$status" -eq 0 ]
}

@test "apt-repo-prune cleans up passphrase file only when it was created" {
  run grep -E '\[ -n "\$GPG_PASSPHRASE_FILE" \][[:space:]]+&&[[:space:]]+rm -f' "$ACTION_FILE"
  [ "$status" -eq 0 ]
}

@test "apt-repo-prune plan step has id: plan" {
  run awk '
    /^    - name: Analyze prune plan$/ { in_step=1; next }
    in_step && /^      id: plan$/ { found=1; exit 0 }
    in_step && /^    - name:/ { in_step=0 }
    END { exit found ? 0 : 1 }
  ' "$ACTION_FILE"
  [ "$status" -eq 0 ]
}

@test "apt-repo-prune plan step writes should_prune to GITHUB_OUTPUT" {
  run grep -F 'should_prune' "$ACTION_FILE"
  [ "$status" -eq 0 ]
  run grep -F 'should_prune=true' "$ACTION_FILE"
  [ "$status" -eq 0 ]
  run grep -F 'should_prune=false' "$ACTION_FILE"
  [ "$status" -eq 0 ]
}

@test "apt-repo-prune no-op step is conditional on should_prune != true" {
  run grep -F "steps.plan.outputs.should_prune != 'true'" "$ACTION_FILE"
  [ "$status" -eq 0 ]
}

@test "apt-repo-prune execute step is conditional on should_prune == true" {
  run grep -F "steps.plan.outputs.should_prune == 'true'" "$ACTION_FILE"
  [ "$status" -eq 0 ]
}

@test "apt-repo-prune execute step reads apt_repo_dir from plan outputs" {
  run grep -F 'steps.plan.outputs.apt_repo_dir' "$ACTION_FILE"
  [ "$status" -eq 0 ]
}

@test "apt-repo-prune execute step reads paths_to_remove_file from plan outputs" {
  run grep -F 'steps.plan.outputs.paths_to_remove_file' "$ACTION_FILE"
  [ "$status" -eq 0 ]
}
