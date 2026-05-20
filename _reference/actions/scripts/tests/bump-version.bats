#!/usr/bin/env bats

# Unit tests for scripts/bump-version.sh
# Git commands are stubbed so no real repository or remote is needed.

SCRIPT="$(cd "$(dirname "$BATS_TEST_FILENAME")/.." && pwd)/bump-version.sh"

setup() {
  STUB_DIR="$(mktemp -d)"
  # Stub git: record every call and always succeed
  cat > "$STUB_DIR/git" << 'EOF'
#!/bin/bash
echo "git $*" >> "$STUB_DIR/git_calls.log"
EOF
  # Expand STUB_DIR inside the stub at creation time
  sed -i "s|\$STUB_DIR|${STUB_DIR}|g" "$STUB_DIR/git"
  chmod +x "$STUB_DIR/git"
  export PATH="$STUB_DIR:$PATH"
  export _STUB_DIR="$STUB_DIR"
}

teardown() {
  rm -rf "$_STUB_DIR"
}

# ---------------------------------------------------------------------------
# Argument validation
# ---------------------------------------------------------------------------

@test "fails with no arguments" {
  run bash "$SCRIPT"
  [ "$status" -eq 1 ]
  [[ "$output" == *"Usage"* ]]
}

@test "fails with only a version and no action" {
  run bash "$SCRIPT" v1.2.3
  [ "$status" -eq 1 ]
  [[ "$output" == *"Usage"* ]]
}

# ---------------------------------------------------------------------------
# Version format validation
# ---------------------------------------------------------------------------

@test "fails when version lacks leading v" {
  run bash "$SCRIPT" 1.2.3 my-action
  [ "$status" -eq 1 ]
  [[ "$output" == *"semantic format"* ]]
}

@test "fails when version is vX.Y (missing patch)" {
  run bash "$SCRIPT" v1.2 my-action
  [ "$status" -eq 1 ]
  [[ "$output" == *"semantic format"* ]]
}

@test "fails when version contains non-numeric component" {
  run bash "$SCRIPT" v1.2.x my-action
  [ "$status" -eq 1 ]
  [[ "$output" == *"semantic format"* ]]
}

# ---------------------------------------------------------------------------
# Successful execution
# ---------------------------------------------------------------------------

@test "succeeds with a valid version and a single action" {
  run bash "$SCRIPT" v1.2.3 my-action
  [ "$status" -eq 0 ]
  [[ "$output" == *"v1.2.3"* ]]
}

@test "success message includes correct major tag" {
  run bash "$SCRIPT" v3.5.7 my-action
  [ "$status" -eq 0 ]
  [[ "$output" == *"v3"* ]]
}

@test "success message includes correct minor tag" {
  run bash "$SCRIPT" v3.5.7 my-action
  [ "$status" -eq 0 ]
  [[ "$output" == *"v3.5"* ]]
}

@test "succeeds with multiple actions and lists them comma-separated" {
  run bash "$SCRIPT" v1.0.0 action1 action2 action3
  [ "$status" -eq 0 ]
  [[ "$output" == *"action1,action2,action3"* ]]
}

# ---------------------------------------------------------------------------
# Git call verification
# ---------------------------------------------------------------------------

@test "creates an annotated tag for the exact version" {
  run bash "$SCRIPT" v2.3.4 my-action
  [ "$status" -eq 0 ]
  grep -q "git tag -a v2.3.4" "$_STUB_DIR/git_calls.log"
}

@test "force-updates the floating minor tag" {
  run bash "$SCRIPT" v2.3.4 my-action
  [ "$status" -eq 0 ]
  grep -q "git tag -fa v2.3" "$_STUB_DIR/git_calls.log"
}

@test "force-updates the floating major tag" {
  run bash "$SCRIPT" v2.3.4 my-action
  [ "$status" -eq 0 ]
  grep -q "git tag -fa v2 -m" "$_STUB_DIR/git_calls.log"
}

@test "pushes the exact version tag to origin" {
  run bash "$SCRIPT" v2.3.4 my-action
  [ "$status" -eq 0 ]
  grep -q "git push origin v2.3.4" "$_STUB_DIR/git_calls.log"
}
