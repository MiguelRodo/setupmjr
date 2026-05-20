#!/usr/bin/env bats

# Unit tests for scripts/apply-version-bump.sh

SCRIPT="$(cd "$(dirname "$BATS_TEST_FILENAME")/.." && pwd)/apply-version-bump.sh"

# ---------------------------------------------------------------------------
# Argument validation
# ---------------------------------------------------------------------------

@test "fails with no arguments" {
  run bash "$SCRIPT"
  [ "$status" -ne 0 ]
}

@test "fails with only bump type and no version" {
  run bash "$SCRIPT" patch
  [ "$status" -ne 0 ]
}

@test "fails with invalid bump type" {
  run bash "$SCRIPT" hotfix 1.2.3
  [ "$status" -eq 1 ]
  [[ "$output" == *"BUMP_TYPE must be"* ]]
}

# ---------------------------------------------------------------------------
# Patch bump
# ---------------------------------------------------------------------------

@test "patch bump increments patch component" {
  run bash "$SCRIPT" patch 1.2.3
  [ "$status" -eq 0 ]
  [ "$output" = "1.2.4" ]
}

@test "patch bump resets nothing" {
  run bash "$SCRIPT" patch 0.0.0
  [ "$status" -eq 0 ]
  [ "$output" = "0.0.1" ]
}

# ---------------------------------------------------------------------------
# Minor bump
# ---------------------------------------------------------------------------

@test "minor bump increments minor and resets patch" {
  run bash "$SCRIPT" minor 1.2.9
  [ "$status" -eq 0 ]
  [ "$output" = "1.3.0" ]
}

@test "minor bump with zero patch" {
  run bash "$SCRIPT" minor 3.0.0
  [ "$status" -eq 0 ]
  [ "$output" = "3.1.0" ]
}

# ---------------------------------------------------------------------------
# Major bump
# ---------------------------------------------------------------------------

@test "major bump increments major and resets minor and patch" {
  run bash "$SCRIPT" major 1.2.3
  [ "$status" -eq 0 ]
  [ "$output" = "2.0.0" ]
}

@test "major bump from zero" {
  run bash "$SCRIPT" major 0.0.0
  [ "$status" -eq 0 ]
  [ "$output" = "1.0.0" ]
}

# ---------------------------------------------------------------------------
# Leading 'v' is stripped
# ---------------------------------------------------------------------------

@test "strips leading v from version" {
  run bash "$SCRIPT" patch v2.3.4
  [ "$status" -eq 0 ]
  [ "$output" = "2.3.5" ]
}
