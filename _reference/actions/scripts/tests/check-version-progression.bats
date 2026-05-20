#!/usr/bin/env bats

# Unit tests for scripts/check-version-progression.sh

SCRIPT="$(cd "$(dirname "$BATS_TEST_FILENAME")/.." && pwd)/check-version-progression.sh"

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
# Valid progressions (single increment)
# ---------------------------------------------------------------------------

@test "valid patch increment passes" {
  run bash "$SCRIPT" 1.2.4 1.2.3
  [ "$status" -eq 0 ]
  [[ "$output" == *"passed"* ]]
}

@test "valid minor increment with patch reset passes" {
  run bash "$SCRIPT" 1.3.0 1.2.9
  [ "$status" -eq 0 ]
  [[ "$output" == *"passed"* ]]
}

@test "valid major increment with minor and patch reset passes" {
  run bash "$SCRIPT" 2.0.0 1.9.9
  [ "$status" -eq 0 ]
  [[ "$output" == *"passed"* ]]
}

@test "valid first patch from 0.0.0 passes" {
  run bash "$SCRIPT" 0.0.1 0.0.0
  [ "$status" -eq 0 ]
}

# ---------------------------------------------------------------------------
# Invalid: same or older versions
# ---------------------------------------------------------------------------

@test "same version fails" {
  run bash "$SCRIPT" 1.2.3 1.2.3
  [ "$status" -eq 1 ]
  [[ "$output" == *"not ahead"* ]]
}

@test "older patch fails" {
  run bash "$SCRIPT" 1.2.2 1.2.3
  [ "$status" -eq 1 ]
  [[ "$output" == *"not ahead"* ]]
}

@test "older minor fails" {
  run bash "$SCRIPT" 1.1.5 1.2.0
  [ "$status" -eq 1 ]
  [[ "$output" == *"not ahead"* ]]
}

@test "older major fails" {
  run bash "$SCRIPT" 0.9.0 1.0.0
  [ "$status" -eq 1 ]
  [[ "$output" == *"not ahead"* ]]
}

# ---------------------------------------------------------------------------
# Invalid: skipping more than one increment
# ---------------------------------------------------------------------------

@test "skipping a patch increment fails" {
  run bash "$SCRIPT" 1.2.5 1.2.3
  [ "$status" -eq 1 ]
  [[ "$output" == *"more than one increment"* ]]
}

@test "skipping a minor increment fails" {
  run bash "$SCRIPT" 1.4.0 1.2.0
  [ "$status" -eq 1 ]
  [[ "$output" == *"more than one increment"* ]]
}

@test "skipping a major increment fails" {
  run bash "$SCRIPT" 3.0.0 1.0.0
  [ "$status" -eq 1 ]
  [[ "$output" == *"more than one increment"* ]]
}

@test "minor bump without resetting patch fails" {
  run bash "$SCRIPT" 1.3.1 1.2.9
  [ "$status" -eq 1 ]
  [[ "$output" == *"more than one increment"* ]]
}

@test "major bump without resetting minor fails" {
  run bash "$SCRIPT" 2.1.0 1.9.9
  [ "$status" -eq 1 ]
  [[ "$output" == *"more than one increment"* ]]
}
