#!/usr/bin/env bats

# Unit tests for scripts/determine-bump-type.sh

SCRIPT="$(cd "$(dirname "$BATS_TEST_FILENAME")/.." && pwd)/determine-bump-type.sh"

# ---------------------------------------------------------------------------
# Argument validation
# ---------------------------------------------------------------------------

@test "fails with no arguments" {
  run bash "$SCRIPT"
  [ "$status" -ne 0 ]
}

@test "fails with only one argument" {
  run bash "$SCRIPT" 1.2.3
  [ "$status" -ne 0 ]
}

# ---------------------------------------------------------------------------
# Patch bump detection
# ---------------------------------------------------------------------------

@test "patch increment detected" {
  run bash "$SCRIPT" 1.2.4 1.2.3
  [ "$status" -eq 0 ]
  [ "$output" = "patch" ]
}

@test "patch from zero detected" {
  run bash "$SCRIPT" 0.0.1 0.0.0
  [ "$status" -eq 0 ]
  [ "$output" = "patch" ]
}

# ---------------------------------------------------------------------------
# Minor bump detection
# ---------------------------------------------------------------------------

@test "minor increment detected" {
  run bash "$SCRIPT" 1.3.0 1.2.5
  [ "$status" -eq 0 ]
  [ "$output" = "minor" ]
}

@test "minor increment with same patch detected as minor" {
  run bash "$SCRIPT" 1.3.5 1.2.5
  [ "$status" -eq 0 ]
  [ "$output" = "minor" ]
}

# ---------------------------------------------------------------------------
# Major bump detection
# ---------------------------------------------------------------------------

@test "major increment detected" {
  run bash "$SCRIPT" 2.0.0 1.9.9
  [ "$status" -eq 0 ]
  [ "$output" = "major" ]
}

@test "major takes precedence over minor when both increase" {
  run bash "$SCRIPT" 2.1.0 1.0.0
  [ "$status" -eq 0 ]
  [ "$output" = "major" ]
}

@test "major takes precedence over patch when both increase" {
  run bash "$SCRIPT" 2.0.1 1.0.0
  [ "$status" -eq 0 ]
  [ "$output" = "major" ]
}
